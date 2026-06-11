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
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

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
