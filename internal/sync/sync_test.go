package sync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"stream/internal/db"
	"stream/internal/model"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type mockTransport struct {
	roundTrip func(*http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTrip(req)
}

func TestSyncEngineHelpers(t *testing.T) {
	// 1. isRateLimitError
	err429 := &googleapi.Error{Code: 429}
	if !isRateLimitError(err429) {
		t.Errorf("expected 429 to be recognized as rate limit error")
	}

	err403Rate := &googleapi.Error{
		Code: 403,
		Errors: []googleapi.ErrorItem{
			{Reason: "rateLimitExceeded"},
		},
	}
	if !isRateLimitError(err403Rate) {
		t.Errorf("expected 403 rateLimitExceeded to be recognized as rate limit error")
	}

	errRegular := errors.New("other error")
	if isRateLimitError(errRegular) {
		t.Errorf("expected regular error not to be recognized as rate limit error")
	}

	// 2. taskToEvent
	task := model.Task{
		UUID:           "task-123",
		Title:          "Plan Sprint",
		Description:    "Discuss scope",
		Priority:       model.P0,
		StoryPoints:    3,
		LifecycleState: model.StateScheduled,
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 6, 11, 11, 0, 0, 0, time.UTC),
		},
	}

	engine := &SyncEngine{
		logCallback: func(s string) {},
	}
	event := engine.taskToEvent(task)
	if event.Summary != "Plan Sprint" || event.Description != "Discuss scope" {
		t.Errorf("taskToEvent field mismatch: Summary=%q, Description=%q", event.Summary, event.Description)
	}
	if event.ExtendedProperties == nil || event.ExtendedProperties.Private == nil {
		t.Fatalf("expected private extended properties")
	}
	if event.ExtendedProperties.Private["uuid"] != "task-123" {
		t.Errorf("expected UUID in private extended properties to be task-123")
	}

	// 3. rate limit tracking
	engine.rateLimitedUntil = time.Now().Add(-1 * time.Second)
	if engine.isRateLimited() {
		t.Errorf("expected isRateLimited to be false for past rate limit time")
	}

	engine.rateLimitedUntil = time.Now().Add(5 * time.Second)
	if !engine.isRateLimited() {
		t.Errorf("expected isRateLimited to be true for future rate limit time")
	}

	// 4. skippable and stale ledger entry helpers
	nonAnchoredErr := errors.New("non-anchored task")
	entry := db.LedgerEntry{
		TaskUUID: "t1",
		Op:       "CREATE",
	}
	if !engine.handleSkippableLedgerEntry(entry, nonAnchoredErr) {
		t.Errorf("expected handleSkippableLedgerEntry to return true for non-anchored task error")
	}
	if engine.handleSkippableLedgerEntry(entry, errors.New("other error")) {
		t.Errorf("expected handleSkippableLedgerEntry to return false for other errors")
	}

	// 5. Offline NewSyncEngine
	t.Setenv("HOME", t.TempDir())
	localDB, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	engine2, err := NewSyncEngine(localDB, nil, nil)
	if err != nil {
		t.Fatalf("failed to create sync engine: %v", err)
	}
	if engine2.IsOnline() {
		t.Errorf("expected sync engine to be offline without credentials")
	}
}

func TestSyncEngine_StartAuthServer_Errors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	localDB, _ := db.NewJSONDB()
	engine, _ := NewSyncEngine(localDB, nil, nil)

	// No config loaded
	_, err := engine.StartAuthServer(8080)
	if err == nil || !strings.Contains(err.Error(), "no client_secrets.json loaded") {
		t.Errorf("expected error when starting auth server without client secrets, got: %v", err)
	}
}

func TestSyncEngine_ManualPullAndPush_Offline(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	localDB, _ := db.NewJSONDB()
	var logs []string
	engine, _ := NewSyncEngine(localDB, func(s string) {
		logs = append(logs, s)
	}, nil)

	engine.ManualPull()
	if len(logs) == 0 || !strings.Contains(logs[len(logs)-1], "GCal Sync Error") {
		t.Errorf("expected GCal Sync Error log, got %v", logs)
	}

	logs = nil
	engine.ManualPush()
	if len(logs) == 0 || !strings.Contains(logs[len(logs)-1], "GCal Sync Error") {
		t.Errorf("expected GCal Sync Error log, got %v", logs)
	}
}

func TestSyncEngine_ManualPull_Success(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create client_secrets.json and credentials.json so initOAuth succeeds
	configDir := filepath.Join(tmpDir, ".config", "stream")
	_ = os.MkdirAll(configDir, 0755)
	_ = os.WriteFile(filepath.Join(configDir, "client_secrets.json"), []byte(`{"installed":{"client_id":"123","client_secret":"abc","auth_uri":"https://auth","token_uri":"https://token"}}`), 0644)
	_ = os.WriteFile(filepath.Join(configDir, "credentials.json"), []byte(`{"access_token":"tok","token_type":"Bearer","refresh_token":"ref","expiry":"3000-01-01T00:00:00Z"}`), 0600)

	localDB, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	// Setup initial workspace
	ws := model.Workspace{UUID: "ws-1", Name: "Default"}
	_ = localDB.AddWorkspace(ws)

	// Setup initial local tasks
	task1 := model.Task{
		UUID:           "local-task-1",
		WorkspaceUUID:  "ws-1",
		Title:          "Local Event 1",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 6, 11, 11, 0, 0, 0, time.UTC),
		},
		GCalMetadata: model.GCalMetadata{
			EventID: "event-id-1",
		},
	}
	_ = localDB.AddTaskNoLedger(task1)

	task2 := model.Task{
		UUID:           "local-task-2",
		WorkspaceUUID:  "ws-1",
		Title:          "Local Event 2",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 6, 11, 13, 0, 0, 0, time.UTC),
		},
		GCalMetadata: model.GCalMetadata{
			EventID: "event-id-2",
		},
	}
	_ = localDB.AddTaskNoLedger(task2)

	var logged []string
	engine, err := NewSyncEngine(localDB, func(s string) {
		logged = append(logged, s)
	}, nil)
	if err != nil {
		t.Fatalf("NewSyncEngine failed: %v", err)
	}

	// We create a mock transport for calendar events
	transport := &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if req.Method == "GET" && strings.Contains(req.URL.Path, "/calendars/primary/events") {
				// We return three events:
				// 1. event-id-1: updated title
				// 2. event-id-2: status is cancelled (should delete local-task-2)
				// 3. event-id-3: new event (should create new local task)
				respBody := `{
					"items": [
						{
							"id": "event-id-1",
							"summary": "Updated Local Event 1",
							"status": "confirmed",
							"start": {"dateTime": "2026-06-11T10:00:00Z"},
							"end": {"dateTime": "2026-06-11T11:00:00Z"},
							"extendedProperties": {
								"private": {
									"uuid": "local-task-1",
									"priority": "P0",
									"story_points": "5",
									"lifecycle_state": "SCHEDULED",
									"scheduling_type": "ANCHORED"
								}
							}
						},
						{
							"id": "event-id-2",
							"status": "cancelled"
						},
						{
							"id": "event-id-3",
							"summary": "Remote Event 3",
							"status": "confirmed",
							"start": {"dateTime": "2026-06-11T14:00:00Z"},
							"end": {"dateTime": "2026-06-11T15:00:00Z"},
							"extendedProperties": {
								"private": {
									"uuid": "remote-uuid-3",
									"priority": "P1",
									"story_points": "3",
									"lifecycle_state": "SCHEDULED",
									"scheduling_type": "ANCHORED"
								}
							}
						}
					]
				}`
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(respBody)),
					Header:     make(http.Header),
				}, nil
			}
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		},
	}

	client := &http.Client{Transport: transport}
	srv, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		t.Fatalf("failed to create calendar service: %v", err)
	}

	engine.srv = srv
	engine.isOnline = true

	engine.ManualPull()

	// Check local-task-1 updated
	t1, ok := localDB.GetTask("local-task-1")
	if !ok {
		t.Errorf("expected local-task-1 to exist")
	} else {
		if t1.Title != "Updated Local Event 1" || t1.Priority != model.P0 || t1.StoryPoints != 5 {
			t.Errorf("local-task-1 was not correctly updated by pull: %+v", t1)
		}
	}

	// Check local-task-2 deleted
	_, ok = localDB.GetTask("local-task-2")
	if ok {
		t.Errorf("expected local-task-2 to be deleted by pull")
	}

	// Check remote-uuid-3 created
	t3, ok := localDB.GetTask("remote-uuid-3")
	if !ok {
		t.Errorf("expected remote-uuid-3 to be created by pull")
	} else {
		if t3.Title != "Remote Event 3" || t3.Priority != model.P1 || t3.StoryPoints != 3 {
			t.Errorf("remote-uuid-3 was not correctly created by pull: %+v", t3)
		}
	}
}

func TestSyncEngine_ManualPush_Success(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "stream")
	_ = os.MkdirAll(configDir, 0755)
	_ = os.WriteFile(filepath.Join(configDir, "client_secrets.json"), []byte(`{"installed":{"client_id":"123"}}`), 0644)
	_ = os.WriteFile(filepath.Join(configDir, "credentials.json"), []byte(`{"access_token":"tok"}`), 0600)

	localDB, _ := db.NewJSONDB()

	// Add workspace
	_ = localDB.AddWorkspace(model.Workspace{UUID: "ws-1", Name: "Default"})

	// Setup tasks:
	// 1. local-task-1: Has EventID, exists on remote -> Should trigger UPDATE
	// 2. local-task-2: No EventID -> Should trigger INSERT
	// 3. local-task-3: Floating -> Should not sync at all
	t1 := model.Task{
		UUID:           "local-task-1",
		WorkspaceUUID:  "ws-1",
		Title:          "Task 1 Pushed",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 6, 11, 11, 0, 0, 0, time.UTC),
		},
		GCalMetadata: model.GCalMetadata{
			EventID: "event-id-1",
		},
	}
	_ = localDB.AddTaskNoLedger(t1)

	t2 := model.Task{
		UUID:           "local-task-2",
		WorkspaceUUID:  "ws-1",
		Title:          "Task 2 Pushed",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 6, 11, 13, 0, 0, 0, time.UTC),
		},
	}
	_ = localDB.AddTaskNoLedger(t2)

	t3 := model.Task{
		UUID:           "local-task-3",
		WorkspaceUUID:  "ws-1",
		Title:          "Task 3 Pushed (Floating)",
		SchedulingType: model.Floating,
	}
	_ = localDB.AddTaskNoLedger(t3)

	// Ledger has a deletion entry
	_ = localDB.AddTask(model.Task{
		UUID:           "deleted-task",
		Title:          "Deleted Task",
		SchedulingType: model.Anchored,
		GCalMetadata: model.GCalMetadata{
			EventID: "event-id-deleted",
		},
	})
	_ = localDB.DeleteTask("deleted-task")

	var logged []string
	engine, _ := NewSyncEngine(localDB, func(s string) {
		logged = append(logged, s)
	}, nil)

	var putCalled, postCalled, deleteCalled bool
	transport := &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if req.Method == "GET" && strings.Contains(req.URL.Path, "/calendars/primary/events") {
				respBody := `{
					"items": [
						{
							"id": "event-id-1",
							"summary": "Old Task 1 Summary",
							"status": "confirmed",
							"start": {"dateTime": "2026-06-11T10:00:00Z"},
							"end": {"dateTime": "2026-06-11T11:00:00Z"}
						},
						{
							"id": "event-id-deleted",
							"summary": "Deleted Task Summary",
							"status": "confirmed"
						}
					]
				}`
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(respBody)),
					Header:     make(http.Header),
				}, nil
			}

			if req.Method == "DELETE" && strings.Contains(req.URL.Path, "/events/event-id-deleted") {
				deleteCalled = true
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
					Header:     make(http.Header),
				}, nil
			}

			if req.Method == "PUT" && strings.Contains(req.URL.Path, "/events/event-id-1") {
				putCalled = true
				respBody := `{"id": "event-id-1", "etag": "new-etag", "sequence": 2}`
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(respBody)),
					Header:     make(http.Header),
				}, nil
			}

			if req.Method == "POST" && strings.Contains(req.URL.Path, "/events") {
				postCalled = true
				respBody := `{"id": "event-id-new", "etag": "new-etag", "sequence": 1}`
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(respBody)),
					Header:     make(http.Header),
				}, nil
			}

			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		},
	}

	client := &http.Client{Transport: transport}
	srv, _ := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	engine.srv = srv
	engine.isOnline = true

	engine.ManualPush()

	if !deleteCalled {
		t.Errorf("expected DELETE request for event-id-deleted")
	}
	if !putCalled {
		t.Errorf("expected PUT request for event-id-1")
	}
	if !postCalled {
		t.Errorf("expected POST request to create event for local-task-2")
	}

	// Verify ledger is cleared
	ledger := localDB.GetLedger()
	if len(ledger) > 0 {
		t.Errorf("expected ledger to be cleared after push, got %d items", len(ledger))
	}
}

func TestSyncEngine_HandleRateLimits(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	localDB, _ := db.NewJSONDB()
	var logs []string
	engine, _ := NewSyncEngine(localDB, func(s string) {
		logs = append(logs, s)
	}, nil)

	// Set client_secrets and credentials so ensureService resolves
	configDir := filepath.Join(localDB.GetConfigDir())
	_ = os.MkdirAll(configDir, 0755)
	_ = os.WriteFile(filepath.Join(configDir, "client_secrets.json"), []byte(`{"installed":{"client_id":"123"}}`), 0644)
	_ = os.WriteFile(filepath.Join(configDir, "credentials.json"), []byte(`{"access_token":"tok"}`), 0600)

	// We trigger standard rate limit handling
	apiErr := &googleapi.Error{
		Code: 429,
		Header: http.Header{
			"Retry-After": []string{"10"},
		},
	}
	engine.handleRateLimit(apiErr)

	if !engine.isRateLimited() {
		t.Errorf("expected isRateLimited to return true after handleRateLimit")
	}

	found := false
	for _, l := range logs {
		if strings.Contains(l, "Retrying in 10s") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected logs to contain retry message, got %v", logs)
	}

	// Manual Pull when rate limited should log warning and return
	logs = nil
	engine.ManualPull()
	if len(logs) == 0 || !strings.Contains(logs[0], "Rate limited. Please wait.") {
		t.Errorf("expected rate limit message in pull log, got %v", logs)
	}

	// Manual Push when rate limited should log warning and return
	logs = nil
	engine.ManualPush()
	if len(logs) == 0 || !strings.Contains(logs[0], "Rate limited. Please wait.") {
		t.Errorf("expected rate limit message in push log, got %v", logs)
	}
}

func TestSyncEngine_TriggerSyncAndDaemons(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	localDB, _ := db.NewJSONDB()
	engine, _ := NewSyncEngine(localDB, nil, nil)

	engine.TriggerPushSync()
	engine.TriggerFullSync()

	// Stop / Start daemon check
	engine.StartDaemon()
	engine.Stop()

	// Settings notify
	engine.NotifySettingsChanged()
}

func TestSyncEngine_StartAuthServer_Callback(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "stream")
	_ = os.MkdirAll(configDir, 0755)
	_ = os.WriteFile(filepath.Join(configDir, "client_secrets.json"), []byte(`{"installed":{"client_id":"123","client_secret":"abc","auth_uri":"https://auth","token_uri":"https://token","redirect_uris":["http://localhost"]}}`), 0644)

	localDB, _ := db.NewJSONDB()
	var logs []string
	engine, _ := NewSyncEngine(localDB, func(s string) {
		logs = append(logs, s)
	}, nil)

	// Get a free port
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	authURL, err := engine.StartAuthServer(port)
	if err != nil {
		t.Fatalf("StartAuthServer failed: %v", err)
	}
	if !strings.Contains(authURL, "state-token") {
		t.Errorf("expected state-token in authURL, got %s", authURL)
	}

	// Wait briefly for the server to start
	time.Sleep(10 * time.Millisecond)

	// Trigger callback with empty code
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/?code=", port))
	if err == nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(string(body), "Error: Missing authorization code") {
			t.Errorf("expected Missing authorization code response, got %s", string(body))
		}
	}

	// Trigger callback with valid-looking code but OAuth exchange fails (since config has dummy URLs)
	resp2, err := http.Get(fmt.Sprintf("http://localhost:%d/?code=validcode", port))
	if err == nil {
		body, _ := io.ReadAll(resp2.Body)
		resp2.Body.Close()
		if !strings.Contains(string(body), "Exchange Token Error") {
			t.Errorf("expected Exchange Token Error, got %s", string(body))
		}
	}
}

func TestSyncEngine_Sync_Replay(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "stream")
	_ = os.MkdirAll(configDir, 0755)
	_ = os.WriteFile(filepath.Join(configDir, "client_secrets.json"), []byte(`{"installed":{"client_id":"123"}}`), 0644)
	_ = os.WriteFile(filepath.Join(configDir, "credentials.json"), []byte(`{"access_token":"tok"}`), 0600)

	localDB, _ := db.NewJSONDB()
	_ = localDB.AddWorkspace(model.Workspace{UUID: "ws-1", Name: "Default"})

	// Enable sync mode so replay loop runs
	settings := localDB.GetUserSettings()
	settings.GCalSyncMode = model.GCalSyncTwoWay
	_ = localDB.UpdateUserSettings(settings)

	// Create a task that exists locally
	t1 := model.Task{
		UUID:           "task-1",
		WorkspaceUUID:  "ws-1",
		Title:          "Task 1",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: time.Now(),
			End:   time.Now().Add(1 * time.Hour),
		},
	}
	_ = localDB.AddTaskNoLedger(t1)

	// Append some ledger entries manually
	// 1. CREATE task-1
	_ = localDB.AddTask(t1) // will add a CREATE to the ledger

	engine, _ := NewSyncEngine(localDB, nil, nil)

	// Mock transport
	transport := &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if req.Method == "GET" && strings.Contains(req.URL.Path, "/calendars/primary") {
				// Calendar check
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
					Header:     make(http.Header),
				}, nil
			}
			if req.Method == "POST" && strings.Contains(req.URL.Path, "/events") {
				// Insert
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(`{"id":"evt-1"}`)),
					Header:     make(http.Header),
				}, nil
			}
			return nil, fmt.Errorf("unexpected: %s", req.URL.Path)
		},
	}

	client := &http.Client{Transport: transport}
	srv, _ := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	engine.srv = srv
	engine.isOnline = true

	// Call sync directly
	engine.sync(false)

	// Ledger should be empty now
	if len(localDB.GetLedger()) > 0 {
		t.Errorf("expected ledger to be replayed and cleared, got %d", len(localDB.GetLedger()))
	}
}
