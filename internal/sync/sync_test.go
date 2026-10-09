package sync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"stream/internal/db"
	"stream/internal/model"

	"golang.org/x/oauth2"
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
		SchedulingType: model.Event,
		Source:         model.SourceGCal,
		TimeWindow: model.TimeWindow{
			Start: time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 6, 11, 13, 0, 0, 0, time.UTC),
		},
		GCalMetadata: model.GCalMetadata{
			EventID: "event-id-2",
		},
	}
	_ = localDB.AddTaskNoLedger(task2)

	taskAnchored := model.Task{
		UUID:           "local-anchored-task",
		WorkspaceUUID:  "ws-1",
		Title:          "Crucial Anchored Task",
		SchedulingType: model.Anchored,
		Source:         model.SourceStream,
		TimeWindow: model.TimeWindow{
			Start: time.Date(2026, 6, 11, 15, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 6, 11, 16, 0, 0, 0, time.UTC),
		},
		GCalMetadata: model.GCalMetadata{
			EventID: "event-id-anchored-cancelled",
		},
	}
	_ = localDB.AddTaskNoLedger(taskAnchored)

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
				// We return events:
				// 1. event-id-1: updated title
				// 2. event-id-2: status is cancelled (should delete GCal event local-task-2)
				// 3. event-id-anchored-cancelled: status cancelled (should deanchor, NOT delete local-anchored-task)
				// 4. event-id-3: new event (should create new local task)
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
							"id": "event-id-anchored-cancelled",
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

	// Check GCal event local-task-2 deleted
	_, ok = localDB.GetTask("local-task-2")
	if ok {
		t.Errorf("expected local-task-2 to be deleted by pull")
	}

	// Check anchored task is NOT deleted, but safely deanchored to Floating
	tAnc, ok := localDB.GetTask("local-anchored-task")
	if !ok {
		t.Errorf("expected local-anchored-task to NOT be deleted by pull")
	} else if tAnc.SchedulingType != model.Floating {
		t.Errorf("expected local-anchored-task to be deanchored to Floating, got %v", tAnc.SchedulingType)
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
	now := time.Now()
	t1 := model.Task{
		UUID:           "local-task-1",
		WorkspaceUUID:  "ws-1",
		Title:          "Task 1 Pushed",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: now.Add(1 * time.Hour),
			End:   now.Add(2 * time.Hour),
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
			Start: now.Add(3 * time.Hour),
			End:   now.Add(4 * time.Hour),
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

	var patchCalled, postCalled, deleteCalled bool
	transport := &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if req.Method == "GET" && strings.Contains(req.URL.Path, "/calendars/primary/events") {
				respBody := fmt.Sprintf(`{
					"items": [
						{
							"id": "event-id-1",
							"summary": "Old Task 1 Summary",
							"status": "confirmed",
							"start": {"dateTime": "%s"},
							"end": {"dateTime": "%s"}
						},
						{
							"id": "event-id-deleted",
							"summary": "Deleted Task Summary",
							"status": "confirmed"
						}
					]
				}`, now.Add(1*time.Hour).Format(time.RFC3339), now.Add(2*time.Hour).Format(time.RFC3339))
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

			if (req.Method == "PATCH" || req.Method == "PUT") && strings.Contains(req.URL.Path, "/events/event-id-1") {
				patchCalled = true
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
	if !patchCalled {
		t.Errorf("expected PATCH/PUT request for event-id-1")
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

func TestFormatSyncError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "googleapi error 401",
			err:      &googleapi.Error{Code: 401, Message: "Unauthorized"},
			expected: "API error 401: Unauthorized",
		},
		{
			name: "url error with lookup failure",
			err: &url.Error{
				Op:  "Get",
				URL: "https://www.googleapis.com/calendar/v3/calendars/primary/events?alt=json&prettyPrint=false&showDeleted=true&timeMin=2026-05-20T18%3A12%3A07%2B0",
				Err: &net.OpError{
					Op:  "dial",
					Net: "tcp",
					Err: &net.DNSError{
						Err: "no such host",
					},
				},
			},
			expected: "network unreachable (offline)",
		},
		{
			name: "generic oauth error",
			err:  errors.New("oauth2: cannot fetch token: 401 Unauthorized"),
			expected: "OAuth token refresh failed (check connection/credentials)",
		},
		{
			name:     "generic offline error string",
			err:      errors.New("dial tcp: lookup www.googleapis.com: no such host"),
			expected: "network unreachable (offline)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := formatSyncError(tc.err)
			if !strings.Contains(actual, tc.expected) && !strings.Contains(tc.expected, actual) {
				t.Errorf("expected string containing %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestSyncEngine_AutoSyncDaemon(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "stream")
	_ = os.MkdirAll(configDir, 0755)
	_ = os.WriteFile(filepath.Join(configDir, "client_secrets.json"), []byte(`{"installed":{"client_id":"123"}}`), 0644)
	_ = os.WriteFile(filepath.Join(configDir, "credentials.json"), []byte(`{"access_token":"tok"}`), 0600)

	localDB, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	ws := model.Workspace{UUID: "ws-1", Name: "Default"}
	_ = localDB.AddWorkspace(ws)

	// Set sync mode to two-way with very short interval (1s)
	settings := localDB.GetUserSettings()
	settings.GCalSyncMode = model.GCalSyncTwoWay
	settings.GCalSyncIntervalSeconds = 1
	_ = localDB.UpdateUserSettings(settings)

	// Local task to push
	task1 := model.Task{
		UUID:           "local-task-auto",
		WorkspaceUUID:  "ws-1",
		Title:          "Auto Task Local",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 6, 11, 11, 0, 0, 0, time.UTC),
		},
	}
	_ = localDB.AddTask(task1)

	var logged []string
	engine, err := NewSyncEngine(localDB, func(s string) {
		logged = append(logged, s)
	}, nil)
	if err != nil {
		t.Fatalf("NewSyncEngine failed: %v", err)
	}

	var postCalled bool
	transport := &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if req.Method == "GET" && strings.Contains(req.URL.Path, "/events") {
				respBody := `{
					"items": [
						{
							"id": "remote-evt-auto-2",
							"summary": "Remote Event Auto",
							"status": "confirmed",
							"start": {"dateTime": "2026-06-11T14:00:00Z"},
							"end": {"dateTime": "2026-06-11T15:00:00Z"},
							"extendedProperties": {
								"private": {
									"uuid": "remote-auto-uuid",
									"priority": "P1",
									"story_points": "2",
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
			if req.Method == "GET" && strings.Contains(req.URL.Path, "/calendars/primary") {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
					Header:     make(http.Header),
				}, nil
			}
			if req.Method == "POST" && strings.Contains(req.URL.Path, "/events") {
				postCalled = true
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(`{"id":"evt-auto-1"}`)),
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

	engine.StartDaemon()
	defer engine.Stop()

	// Wait for daemon to execute sync cycle
	time.Sleep(150 * time.Millisecond)

	// Check post was called for local task
	if !postCalled {
		t.Errorf("expected local task to be pushed automatically by daemon")
	}

	// Check remote task was pulled into local DB
	pulledTask, ok := localDB.GetTask("remote-auto-uuid")
	if !ok {
		t.Errorf("expected remote task to be pulled automatically by daemon")
	} else if pulledTask.Title != "Remote Event Auto" {
		t.Errorf("expected pulled task title 'Remote Event Auto', got %q", pulledTask.Title)
	}
}
func TestSyncEngine_StaleDataConflictResolution(t *testing.T) {
	localDB, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	engine, err := NewSyncEngine(localDB, nil, nil)
	if err != nil {
		t.Fatalf("failed to init sync engine: %v", err)
	}

	// 1. Task exists locally and was updated locally more recently than remote
	localTask := model.Task{
		UUID:           "task-conflict-1",
		Title:          "Local Fresh Title",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 6, 11, 11, 0, 0, 0, time.UTC),
		},
		GCalMetadata: model.GCalMetadata{
			EventID: "event-conflict-1",
		},
	}
	_ = localDB.AddTask(localTask)

	// Remote item has an older updated timestamp and older title
	remoteListJSON := `{
		"kind": "calendar#events",
		"items": [
			{
				"id": "event-conflict-1",
				"status": "confirmed",
				"summary": "Stale Remote Title",
				"updated": "2026-06-11T08:00:00.000Z",
				"start": {"dateTime": "2026-06-11T10:00:00Z"},
				"end": {"dateTime": "2026-06-11T11:00:00Z"},
				"extendedProperties": {
					"private": {
						"uuid": "task-conflict-1",
						"source": "stream",
						"scheduling_type": "ANCHORED"
					}
				}
			}
		]
	}`

	transport := &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/events") && req.Method == "GET" {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(remoteListJSON)),
					Header:     make(http.Header),
				}, nil
			}
			return nil, fmt.Errorf("unexpected: %s", req.URL.Path)
		},
	}

	client := &http.Client{Transport: transport}
	srv, _ := calendar.NewService(context.Background(), option.WithHTTPClient(client))

	err = engine.pullRemoteUpdates(srv)
	if err != nil {
		t.Fatalf("pullRemoteUpdates failed: %v", err)
	}

	// Verify local task title was NOT overwritten with stale remote title
	taskAfterPull, _ := localDB.GetTask("task-conflict-1")
	if taskAfterPull.Title != "Local Fresh Title" {
		t.Errorf("expected local title to be preserved ('Local Fresh Title'), got %q", taskAfterPull.Title)
	}

	// 2. Pending delete prevention: Task was deleted locally, remote list still has it
	_ = localDB.DeleteTask("task-conflict-1") // records DELETE in ledger

	deleteRemoteCalled := false
	transportDelete := &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/events") && req.Method == "GET" {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(remoteListJSON)),
					Header:     make(http.Header),
				}, nil
			}
			if strings.Contains(req.URL.Path, "/events/event-conflict-1") && req.Method == "DELETE" {
				deleteRemoteCalled = true
				return &http.Response{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
					Header:     make(http.Header),
				}, nil
			}
			return nil, fmt.Errorf("unexpected: %s %s", req.Method, req.URL.Path)
		},
	}
	clientDelete := &http.Client{Transport: transportDelete}
	srvDelete, _ := calendar.NewService(context.Background(), option.WithHTTPClient(clientDelete))

	err = engine.pullRemoteUpdates(srvDelete)
	if err != nil {
		t.Fatalf("pullRemoteUpdates on pending delete failed: %v", err)
	}

	// Task should not be resurrected in local DB
	if _, exists := localDB.GetTask("task-conflict-1"); exists {
		t.Errorf("expected locally deleted task NOT to be resurrected by remote pull")
	}
	if !deleteRemoteCalled {
		t.Errorf("expected deleteRemoteEvent to be triggered for pending deleted task")
	}
}

func TestSyncEngine_RateLimitProgression(t *testing.T) {
	localDB, _ := db.NewJSONDB()
	engine, _ := NewSyncEngine(localDB, nil, nil)

	if engine.isRateLimited() {
		t.Errorf("expected not rate limited initially")
	}

	// First rate limit: backoff should be 5s
	err429 := &googleapi.Error{Code: 429}
	engine.handleRateLimit(err429)

	if !engine.isRateLimited() {
		t.Errorf("expected rate limited after handleRateLimit")
	}
	if engine.consecutiveRateLimits != 1 {
		t.Errorf("expected consecutiveRateLimits to be 1, got %d", engine.consecutiveRateLimits)
	}

	// Second rate limit: backoff increases exponentially
	engine.handleRateLimit(err429)
	if engine.consecutiveRateLimits != 2 {
		t.Errorf("expected consecutiveRateLimits to be 2, got %d", engine.consecutiveRateLimits)
	}

	// Reset backoff on success
	engine.resetRateLimitBackoff()
	if engine.consecutiveRateLimits != 0 {
		t.Errorf("expected consecutiveRateLimits to reset to 0, got %d", engine.consecutiveRateLimits)
	}
}

func TestCompactLedger_Deduplication(t *testing.T) {
	// Scenario 1: CREATE + UPDATE + UPDATE -> single CREATE with latest task payload
	e1 := db.LedgerEntry{ID: "e1", Op: "CREATE", TaskUUID: "t1", Task: model.Task{UUID: "t1", Title: "Initial"}}
	e2 := db.LedgerEntry{ID: "e2", Op: "UPDATE", TaskUUID: "t1", Task: model.Task{UUID: "t1", Title: "Mid"}}
	e3 := db.LedgerEntry{ID: "e3", Op: "UPDATE", TaskUUID: "t1", Task: model.Task{UUID: "t1", Title: "Final"}}

	// Scenario 2: CREATE + DELETE (without remote EventID) -> NOOP (0 network calls)
	e4 := db.LedgerEntry{ID: "e4", Op: "CREATE", TaskUUID: "t2", Task: model.Task{UUID: "t2", Title: "Shortlived"}}
	e5 := db.LedgerEntry{ID: "e5", Op: "DELETE", TaskUUID: "t2", Task: model.Task{UUID: "t2", Title: "Shortlived"}}

	// Scenario 3: UPDATE + UPDATE + DELETE -> single DELETE
	e6 := db.LedgerEntry{ID: "e6", Op: "UPDATE", TaskUUID: "t3", Task: model.Task{UUID: "t3", Title: "Update1", GCalMetadata: model.GCalMetadata{EventID: "ev-3"}}}
	e7 := db.LedgerEntry{ID: "e7", Op: "UPDATE", TaskUUID: "t3", Task: model.Task{UUID: "t3", Title: "Update2", GCalMetadata: model.GCalMetadata{EventID: "ev-3"}}}
	e8 := db.LedgerEntry{ID: "e8", Op: "DELETE", TaskUUID: "t3", Task: model.Task{UUID: "t3", Title: "Update2", GCalMetadata: model.GCalMetadata{EventID: "ev-3"}}}

	entries := []db.LedgerEntry{e1, e2, e3, e4, e5, e6, e7, e8}
	compacted := compactLedger(entries)

	if len(compacted) != 3 {
		t.Fatalf("expected 3 compacted ops, got %d", len(compacted))
	}

	// Verify t1 is CREATE with Title "Final" and all 3 IDs
	if compacted[0].Op != "CREATE" || compacted[0].Task.Title != "Final" || len(compacted[0].SourceEntryIDs) != 3 {
		t.Errorf("unexpected t1 compaction: %+v", compacted[0])
	}

	// Verify t2 is NOOP with 2 IDs
	if compacted[1].Op != "NOOP" || len(compacted[1].SourceEntryIDs) != 2 {
		t.Errorf("unexpected t2 compaction: %+v", compacted[1])
	}

	// Verify t3 is DELETE with 3 IDs
	if compacted[2].Op != "DELETE" || compacted[2].Task.GCalMetadata.EventID != "ev-3" || len(compacted[2].SourceEntryIDs) != 3 {
		t.Errorf("unexpected t3 compaction: %+v", compacted[2])
	}
}

func TestIsTaskAndEventEqual(t *testing.T) {
	start := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 11, 11, 0, 0, 0, time.UTC)

	task := model.Task{
		UUID:           "task-eq-1",
		Title:          "Team Sync",
		Description:    "Weekly sync meeting",
		Location:       "Room 404",
		Priority:       model.P1,
		StoryPoints:    2,
		LifecycleState: model.StateScheduled,
		SchedulingType: model.Anchored,
		TimeWindow:     model.TimeWindow{Start: start, End: end},
	}

	event := &calendar.Event{
		Summary:     "Team Sync",
		Description: "Weekly sync meeting",
		Location:    "Room 404",
		Start:       &calendar.EventDateTime{DateTime: start.Format(time.RFC3339)},
		End:         &calendar.EventDateTime{DateTime: end.Format(time.RFC3339)},
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: map[string]string{
				"uuid":            "task-eq-1",
				"priority":        "P1",
				"story_points":    "2",
				"lifecycle_state": "SCHEDULED",
				"scheduling_type": "ANCHORED",
			},
		},
	}

	if !isTaskAndEventEqual(task, event) {
		t.Errorf("expected task and event to be equal")
	}

	// Modify title -> should not be equal
	taskDiff := task
	taskDiff.Title = "Team Standup"
	if isTaskAndEventEqual(taskDiff, event) {
		t.Errorf("expected different title to return false")
	}

	// Modify time -> should not be equal
	taskDiffTime := task
	taskDiffTime.TimeWindow.End = end.Add(30 * time.Minute)
	if isTaskAndEventEqual(taskDiffTime, event) {
		t.Errorf("expected different time to return false")
	}
}

func TestNormalizeTitleTimeKey(t *testing.T) {
	start := time.Date(2026, 6, 11, 10, 0, 0, 123456, time.UTC)
	end := time.Date(2026, 6, 11, 11, 0, 0, 999999, time.UTC)

	k1 := normalizeTitleTimeKey("  Review PR   #42  ", start, end)
	k2 := normalizeTitleTimeKey("review pr #42", start.Truncate(time.Second), end.Truncate(time.Second))

	if k1 != k2 {
		t.Errorf("expected normalized keys to match: %q vs %q", k1, k2)
	}
}

func TestBatchRemoveLedgerEntries(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	localDB, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	t1 := model.Task{UUID: "t1", Title: "Task 1", SchedulingType: model.Anchored}
	t2 := model.Task{UUID: "t2", Title: "Task 2", SchedulingType: model.Anchored}
	_ = localDB.AddTask(t1)
	_ = localDB.AddTask(t2)

	ledger := localDB.GetLedger()
	if len(ledger) != 2 {
		t.Fatalf("expected 2 ledger entries, got %d", len(ledger))
	}

	// Batch remove both
	ids := []string{ledger[0].ID, ledger[1].ID}
	if err := localDB.RemoveLedgerEntries(ids); err != nil {
		t.Fatalf("RemoveLedgerEntries failed: %v", err)
	}

	if len(localDB.GetLedger()) != 0 {
		t.Errorf("expected ledger to be empty after batch remove, got %d", len(localDB.GetLedger()))
	}
}

func TestTokenPersisterOnRefresh(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "credentials.json")

	var updatedToken *oauth2.Token
	persister := &tokenPersister{
		base:      oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "refreshed-token-123"}),
		tokenPath: tokenPath,
		onUpdate: func(tok *oauth2.Token) {
			updatedToken = tok
		},
	}

	tok, err := persister.Token()
	if err != nil {
		t.Fatalf("Token failed: %v", err)
	}
	if tok.AccessToken != "refreshed-token-123" {
		t.Errorf("expected refreshed token, got %q", tok.AccessToken)
	}
	if updatedToken == nil || updatedToken.AccessToken != "refreshed-token-123" {
		t.Errorf("expected onUpdate to be called with refreshed token")
	}

	// Check file written
	data, err := os.ReadFile(tokenPath)
	if err != nil || !strings.Contains(string(data), "refreshed-token-123") {
		t.Errorf("expected credentials.json to be saved with new token")
	}
}

func TestCompactLedger_DeleteCreateTransition(t *testing.T) {
	entries := []db.LedgerEntry{
		{
			ID:       "e1",
			Op:       "DELETE",
			TaskUUID: "task-1",
			Task: model.Task{
				UUID:         "task-1",
				Title:        "Original Title",
				GCalMetadata: model.GCalMetadata{EventID: "evt-1"},
			},
		},
		{
			ID:       "e2",
			Op:       "CREATE",
			TaskUUID: "task-1",
			Task: model.Task{
				UUID:           "task-1",
				Title:          "Re-anchored Title",
				SchedulingType: model.Anchored,
				GCalMetadata:   model.GCalMetadata{EventID: "evt-1"},
			},
		},
	}

	compacted := compactLedger(entries)
	if len(compacted) != 1 {
		t.Fatalf("expected 1 compacted operation, got %d", len(compacted))
	}
	if compacted[0].Op != "UPDATE" {
		t.Errorf("expected compacted op to be UPDATE, got %s", compacted[0].Op)
	}
	if compacted[0].Task.Title != "Re-anchored Title" {
		t.Errorf("expected task title to be Re-anchored Title, got %s", compacted[0].Task.Title)
	}
	if len(compacted[0].SourceEntryIDs) != 2 {
		t.Errorf("expected 2 source entry IDs, got %d", len(compacted[0].SourceEntryIDs))
	}
}

func TestPullRemoteUpdates_PreserveLifecycleState(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	localDB, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	ws := model.Workspace{UUID: "ws-1", Name: "Default"}
	_ = localDB.AddWorkspace(ws)

	activeTask := model.Task{
		UUID:           "active-task-1",
		WorkspaceUUID:  "ws-1",
		Title:          "In-Progress Task",
		LifecycleState: model.StateActive,
		SchedulingType: model.Anchored,
		Source:         model.SourceStream,
		TimeWindow: model.TimeWindow{
			Start: time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 6, 11, 11, 0, 0, 0, time.UTC),
		},
		GCalMetadata: model.GCalMetadata{
			EventID: "gcal-evt-active",
		},
	}
	_ = localDB.AddTaskNoLedger(activeTask)

	engine, err := NewSyncEngine(localDB, nil, nil)
	if err != nil {
		t.Fatalf("NewSyncEngine failed: %v", err)
	}

	// Remote event returns without lifecycle_state in extendedProperties
	transport := &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			respBody := `{
				"items": [
					{
						"id": "gcal-evt-active",
						"summary": "In-Progress Task Updated on Remote",
						"status": "confirmed",
						"start": {"dateTime": "2026-06-11T10:00:00Z"},
						"end": {"dateTime": "2026-06-11T11:00:00Z"}
					}
				]
			}`
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(respBody)),
				Header:     make(http.Header),
			}, nil
		},
	}

	client := &http.Client{Transport: transport}
	srv, _ := calendar.NewService(context.Background(), option.WithHTTPClient(client))

	err = engine.pullRemoteUpdates(srv)
	if err != nil {
		t.Fatalf("pullRemoteUpdates failed: %v", err)
	}

	updated, ok := localDB.GetTask("active-task-1")
	if !ok {
		t.Fatalf("expected task to exist")
	}
	if updated.LifecycleState != model.StateActive {
		t.Errorf("expected LifecycleState to be preserved as ACTIVE, got %s", updated.LifecycleState)
	}
	if updated.Title != "In-Progress Task Updated on Remote" {
		t.Errorf("expected Title to be updated, got %s", updated.Title)
	}
}





