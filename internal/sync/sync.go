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
	"sync"
	"time"

	"stream/internal/db"
	"stream/internal/model"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type SyncEngine struct {
	mu          sync.RWMutex
	localDB     *db.JSONDB
	oauthConfig *oauth2.Config
	token       *oauth2.Token
	srv         *calendar.Service
	isOnline    bool
	syncChan    chan struct{}
	stopChan    chan struct{}
	logCallback func(string)
}

func NewSyncEngine(localDB *db.JSONDB, logCallback func(string)) (*SyncEngine, error) {
	engine := &SyncEngine{
		localDB:     localDB,
		syncChan:    make(chan struct{}, 1),
		stopChan:    make(chan struct{}),
		logCallback: logCallback,
	}

	if logCallback == nil {
		engine.logCallback = func(s string) {}
	}

	// Try loading client secrets and token
	if err := engine.initOAuth(); err != nil {
		engine.logCallback(fmt.Sprintf("GCal Sync disabled: %v", err))
	} else {
		engine.logCallback("GCal Sync initialized.")
	}

	return engine, nil
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

	// Load existing token
	tokenPath := filepath.Join(s.localDB.GetConfigDir(), "credentials.json")
	if _, err := os.Stat(tokenPath); err == nil {
		tokenData, err := os.ReadFile(tokenPath)
		if err == nil {
			var tok oauth2.Token
			if err := json.Unmarshal(tokenData, &tok); err == nil {
				s.token = &tok
				s.isOnline = true
				if err := s.createService(); err == nil {
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

// StartAuthServer starts the local web server to intercept Google OAuth2 callback
func (s *SyncEngine) StartAuthServer(port int) (string, error) {
	if s.oauthConfig == nil {
		return "", errors.New("no client_secrets.json loaded, cannot authorize")
	}

	// Adjust redirect URL dynamically
	s.oauthConfig.RedirectURL = fmt.Sprintf("http://localhost:%d", port)
	authURL := s.oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

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

			// Save credentials
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
			}

			// Shutdown server in background
			go func() {
				time.Sleep(1 * time.Second)
				server.Shutdown(context.Background())
			}()

			s.TriggerSync()
		})

		server.Serve(listener)
	}()

	return authURL, nil
}

func (s *SyncEngine) TriggerSync() {
	select {
	case s.syncChan <- struct{}{}:
	default:
	}
}

func (s *SyncEngine) StartDaemon() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		// Run initial sync
		s.sync()

		for {
			select {
			case <-ticker.C:
				s.sync()
			case <-s.syncChan:
				s.sync()
			case <-s.stopChan:
				ticker.Stop()
				return
			}
		}
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

func (s *SyncEngine) sync() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srv == nil {
		// Try re-initializing or checking if service can be spun up
		if err := s.initOAuth(); err != nil {
			s.isOnline = false
			return
		}
	}

	s.logCallback("Sync Engine: Checking connection...")
	// Validate connectivity with a quick API call
	_, err := s.srv.Calendars.Get("primary").Do()
	if err != nil {
		s.isOnline = false
		s.logCallback("Sync Engine: Offline. Running in local fallback mode.")
		return
	}
	s.isOnline = true
	s.logCallback("Sync Engine: Online. Replaying local transaction ledger...")

	// 1. Replay Local Ledger
	ledger := s.localDB.GetLedger()
	if len(ledger) > 0 {
		s.logCallback(fmt.Sprintf("Sync Engine: Replaying %d operations...", len(ledger)))
		for _, entry := range ledger {
			if err := s.replayEntry(entry); err != nil {
				s.logCallback(fmt.Sprintf("Sync Ledger Error on %s (%s): %v", entry.Op, entry.TaskUUID, err))
				// Stop replaying on first failure to maintain order, retry later
				return
			}
		}
		s.localDB.ClearLedger()
		s.logCallback("Sync Engine: Local ledger replayed successfully.")
	}

	// 2. Perform Delta Sync from Remote
	s.logCallback("Sync Engine: Pulling remote delta updates...")
	if err := s.pullRemoteUpdates(); err != nil {
		s.logCallback(fmt.Sprintf("Sync Engine Pull Error: %v", err))
	} else {
		s.logCallback("Sync Engine: Delta sync complete.")
	}
}

func (s *SyncEngine) replayEntry(entry db.LedgerEntry) error {
	switch entry.Op {
	case "CREATE":
		return s.createRemoteEvent(entry.Task)
	case "UPDATE":
		return s.updateRemoteEvent(entry.Task)
	case "DELETE":
		return s.deleteRemoteEvent(entry.Task)
	}
	return nil
}

func (s *SyncEngine) createRemoteEvent(task model.Task) error {
	event := s.taskToEvent(task)
	res, err := s.srv.Events.Insert("primary", event).Do()
	if err != nil {
		return err
	}

	// Update GCal metadata on local copy
	task.GCalMetadata.EventID = res.Id
	task.GCalMetadata.ETag = res.Etag
	task.GCalMetadata.SequenceID = res.Sequence

	// We temporarily unlock to update DB, but s.localDB has internal locks
	return s.localDB.UpdateTask(task)
}

func (s *SyncEngine) updateRemoteEvent(task model.Task) error {
	if task.GCalMetadata.EventID == "" {
		// Try to create it if it didn't exist remotely
		return s.createRemoteEvent(task)
	}

	event := s.taskToEvent(task)
	res, err := s.srv.Events.Update("primary", task.GCalMetadata.EventID, event).Do()
	if err != nil {
		// If event deleted on server, recreate it
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 410 {
			return s.createRemoteEvent(task)
		}
		return err
	}

	task.GCalMetadata.ETag = res.Etag
	task.GCalMetadata.SequenceID = res.Sequence
	return s.localDB.UpdateTask(task)
}

func (s *SyncEngine) deleteRemoteEvent(task model.Task) error {
	if task.GCalMetadata.EventID == "" {
		return nil
	}
	err := s.srv.Events.Delete("primary", task.GCalMetadata.EventID).Do()
	if err != nil {
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 410 {
			// Already deleted remotely
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
	}

	if task.SchedulingType == model.Anchored {
		event.Start = &calendar.EventDateTime{
			DateTime: task.TimeWindow.Start.Format(time.RFC3339),
			TimeZone: "UTC",
		}
		event.End = &calendar.EventDateTime{
			DateTime: task.TimeWindow.End.Format(time.RFC3339),
			TimeZone: "UTC",
		}
	} else {
		// All day event or simple reminder event
		today := time.Now().Format("2006-01-02")
		event.Start = &calendar.EventDateTime{Date: today}
		event.End = &calendar.EventDateTime{Date: today}
	}

	return event
}

func (s *SyncEngine) pullRemoteUpdates() error {
	// Simple polling of recent updates (e.g. modified in the last 1 day or similar)
	timeMin := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	events, err := s.srv.Events.List("primary").TimeMin(timeMin).ShowDeleted(true).Do()
	if err != nil {
		return err
	}

	localTasks := s.localDB.GetTasks()
	localByGCalID := make(map[string]model.Task)
	for _, t := range localTasks {
		if t.GCalMetadata.EventID != "" {
			localByGCalID[t.GCalMetadata.EventID] = t
		}
	}

	for _, item := range events.Items {
		if item.Status == "cancelled" {
			// Remote deletion
			if local, exists := localByGCalID[item.Id]; exists {
				s.localDB.DeleteTask(local.UUID)
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

		start, _ := time.Parse(time.RFC3339, item.Start.DateTime)
		end, _ := time.Parse(time.RFC3339, item.End.DateTime)
		if start.IsZero() && item.Start.Date != "" {
			// All day event, parse date
			start, _ = time.Parse("2006-01-02", item.Start.Date)
			end = start.Add(24 * time.Hour)
		}

		localTask, exists := localByGCalID[item.Id]
		if exists {
			// Check sequence or etag
			if item.Sequence > localTask.GCalMetadata.SequenceID || item.Etag != localTask.GCalMetadata.ETag {
				localTask.Title = item.Summary
				localTask.Description = item.Description
				localTask.TimeWindow.Start = start
				localTask.TimeWindow.End = end
				localTask.Priority = priorityVal
				localTask.StoryPoints = spVal
				localTask.LifecycleState = stateVal
				localTask.SchedulingType = schedVal
				localTask.GCalMetadata.ETag = item.Etag
				localTask.GCalMetadata.SequenceID = item.Sequence

				s.localDB.UpdateTask(localTask)
			}
		} else {
			// External event created on remote calendar
			newTask := model.Task{
				UUID:           uuidVal,
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
			s.localDB.AddTask(newTask)
		}
	}

	return nil
}
