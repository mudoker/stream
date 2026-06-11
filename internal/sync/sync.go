package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"stream/internal/db"
	"stream/internal/model"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type syncRequest struct {
	includePull bool
}

type SyncEngine struct {
	mu                   sync.RWMutex
	localDB              *db.JSONDB
	oauthConfig          *oauth2.Config
	token                *oauth2.Token
	srv                  *calendar.Service
	isOnline             bool
	syncChan             chan syncRequest
	settingsChan         chan struct{}
	stopChan             chan struct{}
	logCallback          func(string)
	authCompleteCallback func()
	rateLimitedUntil     time.Time
}

func NewSyncEngine(localDB *db.JSONDB, logCallback func(string), authCompleteCallback func()) (*SyncEngine, error) {
	engine := &SyncEngine{
		localDB:              localDB,
		syncChan:             make(chan syncRequest, 1),
		settingsChan:         make(chan struct{}, 1),
		stopChan:             make(chan struct{}),
		logCallback:          logCallback,
		authCompleteCallback: authCompleteCallback,
	}

	if logCallback == nil {
		engine.logCallback = func(s string) {}
	}

	if err := engine.initOAuth(); err != nil {
		engine.logCallback(fmt.Sprintf("GCal Sync: offline mode (%v)", err))
	} else {
		engine.logCallback("GCal Sync initialized.")
	}

	return engine, nil
}

func (s *SyncEngine) NotifySettingsChanged() {
	select {
	case s.settingsChan <- struct{}{}:
	default:
	}
}

func (s *SyncEngine) initOAuth() error {
	secretPath := filepath.Join(s.localDB.GetConfigDir(), "client_secrets.json")
	if _, err := os.Stat(secretPath); os.IsNotExist(err) {
		return errors.New("client_secrets.json not found in ~/.config/stream/")
	}

	secretData, err := os.ReadFile(secretPath)
	if err != nil {
		return fmt.Errorf("read client secrets error: %w", err)
	}

	config, err := google.ConfigFromJSON(secretData, calendar.CalendarScope)
	if err != nil {
		return fmt.Errorf("parse client secrets error: %w", err)
	}
	s.oauthConfig = config

	tokenPath := filepath.Join(s.localDB.GetConfigDir(), "credentials.json")
	if _, err := os.Stat(tokenPath); err == nil {
		tokenData, err := os.ReadFile(tokenPath)
		if err == nil {
			var tok oauth2.Token
			if err := json.Unmarshal(tokenData, &tok); err == nil {
				s.token = &tok
				if err := s.createService(); err == nil {
					s.isOnline = true
					return nil
				}
			}
		}
	}

	return errors.New("token credentials.json not found, need authentication")
}

func (s *SyncEngine) createService() error {
	if s.oauthConfig == nil {
		return errors.New("oauthConfig is nil")
	}
	ctx := context.Background()
	client := s.oauthConfig.Client(ctx, s.token)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return err
	}
	s.srv = srv
	return nil
}

func (s *SyncEngine) getSyncMode() model.GCalSyncMode {
	return s.localDB.GetUserSettings().NormalizedGCalSync().GCalSyncMode
}

func (s *SyncEngine) getSyncInterval() time.Duration {
	secs := s.localDB.GetUserSettings().NormalizedGCalSync().GCalSyncIntervalSeconds
	if secs <= 0 {
		secs = 5
	}
	return time.Duration(secs) * time.Second
}

// StartAuthServer starts the local web server to intercept Google OAuth2 callback
func (s *SyncEngine) StartAuthServer(port int) (string, error) {
	if s.oauthConfig == nil {
		return "", errors.New("no client_secrets.json loaded, cannot authorize")
	}

	s.oauthConfig.RedirectURL = fmt.Sprintf("http://localhost:%d", port)
	authURL := s.oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	htmlPath := filepath.Join(s.localDB.GetConfigDir(), "auth.html")
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta http-equiv="refresh" content="0; url=%s">
    <title>Stream Authentication Redirect</title>
</head>
<body>
    <p>Redirecting to Google OAuth... If it does not redirect automatically, <a href="%s">click here</a>.</p>
    <script>window.location.href = "%s";</script>
</body>
</html>`, authURL, authURL, authURL)
	_ = os.WriteFile(htmlPath, []byte(htmlContent), 0644)

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return "", fmt.Errorf("failed to bind to port %d: %w", port, err)
	}

	go func() {
		mux := http.NewServeMux()
		var server *http.Server
		server = &http.Server{
			Handler: mux,
		}

		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if code == "" {
				s.logCallback("OAuth Callback: Missing authorization code.")
				io.WriteString(w, "Error: Missing authorization code.")
				return
			}

			if s.oauthConfig == nil {
				s.logCallback("OAuth Callback: Config is nil.")
				io.WriteString(w, "Error: OAuth configuration is nil.")
				return
			}

			tok, err := s.oauthConfig.Exchange(context.Background(), code)
			if err != nil {
				s.logCallback(fmt.Sprintf("OAuth Callback Exchange Error: %v", err))
				io.WriteString(w, fmt.Sprintf("Exchange Token Error: %v", err))
				return
			}

			tokenPath := filepath.Join(s.localDB.GetConfigDir(), "credentials.json")
			tokData, err := json.MarshalIndent(tok, "", "  ")
			if err != nil {
				s.logCallback(fmt.Sprintf("OAuth Callback formatting Error: %v", err))
				io.WriteString(w, fmt.Sprintf("Error formatting token: %v", err))
				return
			}
			if err := os.WriteFile(tokenPath, tokData, 0600); err != nil {
				s.logCallback(fmt.Sprintf("OAuth Callback save Error: %v", err))
				io.WriteString(w, fmt.Sprintf("Error saving credentials to %s: %v", tokenPath, err))
				return
			}

			s.mu.Lock()
			s.token = tok
			s.isOnline = true
			err = s.createService()
			s.mu.Unlock()

			if err != nil {
				s.logCallback(fmt.Sprintf("OAuth Callback service creation failed: %v", err))
				io.WriteString(w, fmt.Sprintf("<h1>Authorization Success, but API client error</h1><p>%v</p>", err))
			} else {
				s.logCallback("OAuth Callback: Authorization successful.")
				io.WriteString(w, "<h1>Authorization Successful!</h1><p>You can close this tab and return to the terminal.</p>")
				if s.authCompleteCallback != nil {
					s.authCompleteCallback()
				}
			}

			_ = os.Remove(filepath.Join(s.localDB.GetConfigDir(), "auth.html"))

			go func() {
				time.Sleep(1 * time.Second)
				server.Shutdown(context.Background())
			}()

			s.TriggerPushSync()
		})

		server.Serve(listener)
	}()

	return authURL, nil
}

func (s *SyncEngine) TriggerPushSync() {
	s.enqueueSync(syncRequest{includePull: false})
}

func (s *SyncEngine) TriggerFullSync() {
	s.enqueueSync(syncRequest{includePull: true})
}

func (s *SyncEngine) enqueueSync(req syncRequest) {
	select {
	case s.syncChan <- req:
	default:
		select {
		case <-s.syncChan:
		default:
		}
		s.syncChan <- req
	}
}

func (s *SyncEngine) StartDaemon() {
	go func() {
		<-s.stopChan
	}()
}

func (s *SyncEngine) Stop() {
	close(s.stopChan)
}

func (s *SyncEngine) IsOnline() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isOnline && s.srv != nil
}

func (s *SyncEngine) ensureService() (*calendar.Service, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srv != nil {
		return s.srv, nil
	}

	if err := s.initOAuth(); err != nil {
		s.isOnline = false
		return nil, err
	}
	return s.srv, nil
}

func (s *SyncEngine) setOnline(online bool) {
	s.mu.Lock()
	s.isOnline = online
	s.mu.Unlock()
}

func isRateLimitError(err error) bool {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		if apiErr.Code == 429 {
			return true
		}
		if apiErr.Code == 403 {
			for _, e := range apiErr.Errors {
				if e.Reason == "rateLimitExceeded" || e.Reason == "userRateLimitExceeded" {
					return true
				}
			}
			return strings.Contains(strings.ToLower(apiErr.Message), "rate limit")
		}
	}
	return false
}

func (s *SyncEngine) handleRateLimit(err error) {
	backoff := 60 * time.Second
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		if retryAfter := apiErr.Header.Get("Retry-After"); retryAfter != "" {
			if secs, convErr := strconv.Atoi(retryAfter); convErr == nil && secs > 0 {
				backoff = time.Duration(secs) * time.Second
			}
		}
	}
	s.rateLimitedUntil = time.Now().Add(backoff)
	s.logCallback(fmt.Sprintf("GCal rate limit hit. Retrying in %ds. Changes stay queued.", int(backoff.Seconds())))
}

func (s *SyncEngine) isRateLimited() bool {
	return time.Now().Before(s.rateLimitedUntil)
}

func (s *SyncEngine) sync(includePull bool) {
	if s.isRateLimited() {
		return
	}

	mode := s.getSyncMode()
	if mode == model.GCalSyncNone && !includePull {
		return
	}

	srv, err := s.ensureService()
	if err != nil {
		if includePull {
			s.logCallback("GCal Sync: credentials unavailable, staying in local mode.")
		}
		return
	}

	if mode != model.GCalSyncNone {
		s.logCallback("Sync Engine: Checking connection...")
		_, err = srv.Calendars.Get("primary").Do()
		if err != nil {
			if isRateLimitError(err) {
				s.handleRateLimit(err)
				return
			}
			s.setOnline(false)
			s.logCallback("Sync Engine: Offline. Anchored changes queued locally.")
			return
		}
		s.setOnline(true)

		s.logCallback("Sync Engine: Online. Replaying anchored task ledger...")
		ledger := s.localDB.GetLedger()
		if len(ledger) > 0 {
			s.logCallback(fmt.Sprintf("Sync Engine: Replaying %d operations...", len(ledger)))
			replayed := 0
			for _, entry := range ledger {
				err := s.replayEntry(srv, entry)
				if err != nil {
					if isRateLimitError(err) {
						s.handleRateLimit(err)
						return
					}
					if s.handleStaleLedgerEntry(srv, entry, err) {
						_ = s.localDB.RemoveLedgerEntry(entry.ID)
						replayed++
						continue
					}
					if s.handleSkippableLedgerEntry(entry, err) {
						_ = s.localDB.RemoveLedgerEntry(entry.ID)
						replayed++
						continue
					}
					s.logCallback(fmt.Sprintf("Sync Ledger Error on %s (%s): %v", entry.Op, entry.TaskUUID, err))
					return
				}
				_ = s.localDB.RemoveLedgerEntry(entry.ID)
				replayed++
			}
			if replayed > 0 {
				s.logCallback(fmt.Sprintf("Sync Engine: Processed %d ledger operations.", replayed))
			}
		}
	}

	if includePull {
		s.logCallback("Sync Engine: Pulling remote updates (manual sync)...")
		if err := s.pullRemoteUpdates(srv); err != nil {
			if isRateLimitError(err) {
				s.handleRateLimit(err)
				return
			}
			s.logCallback(fmt.Sprintf("Sync Engine Pull Error: %v", err))
		} else {
			s.logCallback("Sync Engine: Remote pull complete.")
		}
	}
}

func (s *SyncEngine) replayEntry(srv *calendar.Service, entry db.LedgerEntry) error {
	if entry.Op != "DELETE" && !model.IsGCalSyncable(entry.Task) {
		return fmt.Errorf("non-anchored task")
	}

	switch entry.Op {
	case "CREATE":
		if !s.localDB.TaskExists(entry.TaskUUID) {
			return db.ErrTaskNotFound
		}
		return s.createRemoteEvent(srv, entry.Task)
	case "UPDATE":
		if !s.localDB.TaskExists(entry.TaskUUID) {
			return db.ErrTaskNotFound
		}
		return s.updateRemoteEvent(srv, entry.Task)
	case "DELETE":
		return s.deleteRemoteEvent(srv, entry.Task)
	}
	return nil
}

func (s *SyncEngine) handleSkippableLedgerEntry(entry db.LedgerEntry, err error) bool {
	if err != nil && err.Error() == "non-anchored task" {
		s.logCallback(fmt.Sprintf("Sync: skipping non-anchored ledger entry for %s.", entry.TaskUUID))
		return true
	}
	return false
}

func (s *SyncEngine) handleStaleLedgerEntry(srv *calendar.Service, entry db.LedgerEntry, err error) bool {
	if !errors.Is(err, db.ErrTaskNotFound) {
		return false
	}

	switch entry.Op {
	case "CREATE":
		s.logCallback(fmt.Sprintf("Sync: skipping stale CREATE for deleted task %s.", entry.TaskUUID))
		return true
	case "UPDATE":
		if entry.Task.GCalMetadata.EventID != "" {
			s.logCallback(fmt.Sprintf("Sync: task %s deleted locally, removing remote event.", entry.TaskUUID))
			if delErr := s.deleteRemoteEvent(srv, entry.Task); delErr != nil {
				s.logCallback(fmt.Sprintf("Sync: remote cleanup failed for %s: %v", entry.TaskUUID, delErr))
				return false
			}
			return true
		}
		s.logCallback(fmt.Sprintf("Sync: skipping stale UPDATE for deleted task %s.", entry.TaskUUID))
		return true
	default:
		return false
	}
}

func (s *SyncEngine) createRemoteEvent(srv *calendar.Service, task model.Task) error {
	if !model.IsGCalSyncable(task) {
		return nil
	}
	event := s.taskToEvent(task)
	res, err := srv.Events.Insert("primary", event).Do()
	if err != nil {
		return err
	}

	task.GCalMetadata.EventID = res.Id
	task.GCalMetadata.ETag = res.Etag
	task.GCalMetadata.SequenceID = res.Sequence

	return s.localDB.UpdateTaskNoLedger(task)
}

func (s *SyncEngine) updateRemoteEvent(srv *calendar.Service, task model.Task) error {
	if !model.IsGCalSyncable(task) {
		return nil
	}
	if task.GCalMetadata.EventID == "" {
		return s.createRemoteEvent(srv, task)
	}

	event := s.taskToEvent(task)
	res, err := srv.Events.Update("primary", task.GCalMetadata.EventID, event).Do()
	if err != nil {
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 410 {
			return s.createRemoteEvent(srv, task)
		}
		return err
	}

	task.GCalMetadata.ETag = res.Etag
	task.GCalMetadata.SequenceID = res.Sequence
	return s.localDB.UpdateTaskNoLedger(task)
}

func (s *SyncEngine) deleteRemoteEvent(srv *calendar.Service, task model.Task) error {
	if task.GCalMetadata.EventID == "" {
		return nil
	}
	err := srv.Events.Delete("primary", task.GCalMetadata.EventID).Do()
	if err != nil {
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 410 {
			return nil
		}
		return err
	}
	return nil
}

func (s *SyncEngine) taskToEvent(task model.Task) *calendar.Event {
	event := &calendar.Event{
		Summary:     task.Title,
		Description: task.Description,
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: map[string]string{
				"uuid":            task.UUID,
				"priority":        string(task.Priority),
				"story_points":    strconv.Itoa(task.StoryPoints),
				"lifecycle_state": string(task.LifecycleState),
				"scheduling_type": string(task.SchedulingType),
			},
		},
		Start: &calendar.EventDateTime{
			DateTime: task.TimeWindow.Start.Format(time.RFC3339),
			TimeZone: "UTC",
		},
		End: &calendar.EventDateTime{
			DateTime: task.TimeWindow.End.Format(time.RFC3339),
			TimeZone: "UTC",
		},
	}
	return event
}

func (s *SyncEngine) defaultWorkspaceUUID() string {
	workspaces := s.localDB.GetWorkspaces()
	if len(workspaces) > 0 {
		return workspaces[0].UUID
	}
	return ""
}

func (s *SyncEngine) pullRemoteUpdates(srv *calendar.Service) error {
	timeMin := time.Now().AddDate(0, 0, -30).Format(time.RFC3339)
	events, err := srv.Events.List("primary").TimeMin(timeMin).ShowDeleted(true).Do()
	if err != nil {
		return err
	}

	localTasks := s.localDB.GetTasks()
	localByGCalID := make(map[string]model.Task)
	localByTitleTime := make(map[string]model.Task)
	for _, t := range localTasks {
		if t.GCalMetadata.EventID != "" {
			localByGCalID[t.GCalMetadata.EventID] = t
		}
		if model.IsGCalSyncable(t) {
			key := fmt.Sprintf("%s|%s|%s", strings.ToLower(t.Title), t.TimeWindow.Start.UTC().Format(time.RFC3339), t.TimeWindow.End.UTC().Format(time.RFC3339))
			localByTitleTime[key] = t
		}
	}

	defaultWS := s.defaultWorkspaceUUID()

	for _, item := range events.Items {
		if item.Status == "cancelled" {
			if local, exists := localByGCalID[item.Id]; exists && model.IsGCalSyncable(local) {
				_ = s.localDB.DeleteTask(local.UUID)
			}
			continue
		}

		uuidVal := ""
		priorityVal := model.P2
		spVal := 1
		stateVal := model.StateScheduled
		schedVal := model.Anchored

		if item.ExtendedProperties != nil && item.ExtendedProperties.Private != nil {
			uuidVal = item.ExtendedProperties.Private["uuid"]
			if p, ok := item.ExtendedProperties.Private["priority"]; ok {
				priorityVal = model.Priority(p)
			}
			if spStr, ok := item.ExtendedProperties.Private["story_points"]; ok {
				if sp, err := strconv.Atoi(spStr); err == nil {
					spVal = sp
				}
			}
			if st, ok := item.ExtendedProperties.Private["lifecycle_state"]; ok {
				stateVal = model.LifecycleState(st)
			}
			if sc, ok := item.ExtendedProperties.Private["scheduling_type"]; ok {
				schedVal = model.SchedulingType(sc)
			}
		}

		var start, end time.Time
		if item.Start != nil {
			start, _ = time.Parse(time.RFC3339, item.Start.DateTime)
			if start.IsZero() && item.Start.Date != "" {
				start, _ = time.Parse("2006-01-02", item.Start.Date)
			}
		}
		if item.End != nil {
			end, _ = time.Parse(time.RFC3339, item.End.DateTime)
			if end.IsZero() && item.Start != nil && item.Start.Date != "" {
				end = start.Add(24 * time.Hour)
			}
		}

		if schedVal != model.Anchored || start.IsZero() {
			continue
		}

		localTask, exists := localByGCalID[item.Id]
		if !exists {
			key := fmt.Sprintf("%s|%s|%s", strings.ToLower(item.Summary), start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
			if t, ok := localByTitleTime[key]; ok {
				localTask = t
				exists = true
			}
		}

		if exists {
			// GCal is source of truth, override local
			localTask.Title = item.Summary
			localTask.Description = item.Description
			localTask.TimeWindow.Start = start
			localTask.TimeWindow.End = end
			localTask.Priority = priorityVal
			localTask.StoryPoints = spVal
			localTask.LifecycleState = stateVal
			localTask.SchedulingType = schedVal
			localTask.GCalMetadata.EventID = item.Id
			localTask.GCalMetadata.ETag = item.Etag
			localTask.GCalMetadata.SequenceID = item.Sequence

			_ = s.localDB.UpdateTaskNoLedger(localTask)
		} else {
			if uuidVal == "" {
				uuidVal = uuid.New().String()
			}
			newTask := model.Task{
				UUID:           uuidVal,
				WorkspaceUUID:  defaultWS,
				Title:          item.Summary,
				Description:    item.Description,
				Priority:       priorityVal,
				StoryPoints:    spVal,
				SchedulingType: schedVal,
				TimeWindow: model.TimeWindow{
					Start: start,
					End:   end,
				},
				LifecycleState: stateVal,
				GCalMetadata: model.GCalMetadata{
					EventID:    item.Id,
					ETag:       item.Etag,
					SequenceID: item.Sequence,
				},
			}
			_ = s.localDB.AddTaskNoLedger(newTask)
		}
	}

	return nil
}

func (s *SyncEngine) pushLocalUpdates(srv *calendar.Service) error {
	// 1. Process deletions from the ledger first to remove deleted local tasks from GCal
	ledger := s.localDB.GetLedger()
	for _, entry := range ledger {
		if entry.Op == "DELETE" {
			if entry.Task.GCalMetadata.EventID != "" {
				s.logCallback(fmt.Sprintf("Sync: Deleting GCal event '%s'...", entry.Task.Title))
				_ = s.deleteRemoteEvent(srv, entry.Task)
			}
		}
		_ = s.localDB.RemoveLedgerEntry(entry.ID)
	}

	// 2. Fetch GCal events to avoid duplicates
	timeMin := time.Now().AddDate(0, 0, -30).Format(time.RFC3339)
	events, err := srv.Events.List("primary").TimeMin(timeMin).ShowDeleted(true).Do()
	if err != nil {
		return err
	}

	gcalByEventID := make(map[string]*calendar.Event)
	gcalByTitleTime := make(map[string]*calendar.Event)
	for _, item := range events.Items {
		if item.Status == "cancelled" {
			continue
		}
		gcalByEventID[item.Id] = item
		var start, end time.Time
		if item.Start != nil {
			start, _ = time.Parse(time.RFC3339, item.Start.DateTime)
		}
		if item.End != nil {
			end, _ = time.Parse(time.RFC3339, item.End.DateTime)
		}
		if !start.IsZero() && !end.IsZero() {
			key := fmt.Sprintf("%s|%s|%s", strings.ToLower(item.Summary), start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
			gcalByTitleTime[key] = item
		}
	}

	// 3. Scan local database for ANCHORED tasks and push
	localTasks := s.localDB.GetTasks()
	for _, t := range localTasks {
		if !model.IsGCalSyncable(t) {
			continue
		}

		var matchedEvent *calendar.Event
		if t.GCalMetadata.EventID != "" {
			matchedEvent = gcalByEventID[t.GCalMetadata.EventID]
		}
		if matchedEvent == nil {
			// Try title+time matching
			key := fmt.Sprintf("%s|%s|%s", strings.ToLower(t.Title), t.TimeWindow.Start.UTC().Format(time.RFC3339), t.TimeWindow.End.UTC().Format(time.RFC3339))
			matchedEvent = gcalByTitleTime[key]
		}

		if matchedEvent != nil {
			// Exists on GCal: Update it (local is source of truth)
			t.GCalMetadata.EventID = matchedEvent.Id
			if err := s.updateRemoteEvent(srv, t); err != nil {
				s.logCallback(fmt.Sprintf("Sync: Failed to update GCal event '%s': %v", t.Title, err))
			}
		} else {
			// Does not exist on GCal: Create it
			if err := s.createRemoteEvent(srv, t); err != nil {
				s.logCallback(fmt.Sprintf("Sync: Failed to create GCal event '%s': %v", t.Title, err))
			}
		}
	}

	return nil
}

func (s *SyncEngine) ManualPull() {
	if s.isRateLimited() {
		s.logCallback("GCal Sync: Rate limited. Please wait.")
		return
	}

	srv, err := s.ensureService()
	if err != nil {
		s.logCallback(fmt.Sprintf("GCal Sync Error: %v", err))
		return
	}

	s.logCallback("GCal Sync: Pulling remote updates...")
	if err := s.pullRemoteUpdates(srv); err != nil {
		if isRateLimitError(err) {
			s.handleRateLimit(err)
			return
		}
		s.logCallback(fmt.Sprintf("GCal Sync: Pull failed: %v", err))
	} else {
		s.logCallback("GCal Sync: Pull complete.")
	}
}

func (s *SyncEngine) ManualPush() {
	if s.isRateLimited() {
		s.logCallback("GCal Sync: Rate limited. Please wait.")
		return
	}

	srv, err := s.ensureService()
	if err != nil {
		s.logCallback(fmt.Sprintf("GCal Sync Error: %v", err))
		return
	}

	s.logCallback("GCal Sync: Pushing local updates...")
	if err := s.pushLocalUpdates(srv); err != nil {
		if isRateLimitError(err) {
			s.handleRateLimit(err)
			return
		}
		s.logCallback(fmt.Sprintf("GCal Sync: Push failed: %v", err))
	} else {
		s.logCallback("GCal Sync: Push complete.")
	}
}
