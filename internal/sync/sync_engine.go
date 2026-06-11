package sync

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"stream/internal/db"
	"stream/internal/model"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
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
