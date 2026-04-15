package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const (
	defaultListenAddr  = "127.0.0.1:8787"
	defaultRedirectURI = "http://127.0.0.1:8787/oauth2callback"
)

func main() {
	_ = godotenv.Load(".env.local")
	_ = godotenv.Load()

	var (
		listenAddr  string
		redirectURI string
		codeInput   string
		printOnly   bool
		timeout     time.Duration
	)

	flag.StringVar(&listenAddr, "listen", defaultListenAddr, "local address for the OAuth callback server")
	flag.StringVar(&redirectURI, "redirect-uri", defaultRedirectURI, "OAuth redirect URI used during authorization")
	flag.StringVar(&codeInput, "code", "", "authorization code or full callback URL to exchange without starting the local server")
	flag.BoolVar(&printOnly, "print-only", false, "print the auth URL and exit without starting the callback server")
	flag.DurationVar(&timeout, "timeout", 3*time.Minute, "how long to wait for the OAuth callback")
	flag.Parse()

	clientID := strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_OAUTH_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_OAUTH_CLIENT_SECRET"))
	if clientID == "" || clientSecret == "" {
		log.Fatalf("set GOOGLE_DRIVE_OAUTH_CLIENT_ID and GOOGLE_DRIVE_OAUTH_CLIENT_SECRET in .env.local or the environment first")
	}

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{drive.DriveScope},
		RedirectURL:  redirectURI,
	}

	if codeInput != "" {
		refreshToken, err := exchangeAuthorizationCode(context.Background(), cfg, codeInput)
		if err != nil {
			log.Fatal(err)
		}
		printRefreshToken(refreshToken)
		return
	}

	state, err := randomState()
	if err != nil {
		log.Fatalf("generate oauth state: %v", err)
	}

	authURL := cfg.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.SetAuthURLParam("include_granted_scopes", "true"),
	)

	if printOnly {
		printPrintOnlyInstructions(authURL, redirectURI)
		return
	}

	refreshToken, err := runLocalOAuthFlow(cfg, state, authURL, listenAddr, redirectURI, timeout)
	if err != nil {
		log.Fatal(err)
	}

	printRefreshToken(refreshToken)
}

func runLocalOAuthFlow(cfg *oauth2.Config, expectedState, authURL, listenAddr, redirectURI string, timeout time.Duration) (string, error) {
	redirectPath, err := redirectPathFromURI(redirectURI)
	if err != nil {
		return "", err
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(redirectPath, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "failed to parse callback", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("parse callback query: %w", err):
			default:
			}
			return
		}

		if callbackErr := strings.TrimSpace(r.FormValue("error")); callbackErr != "" {
			http.Error(w, "Google returned an OAuth error", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("google oauth error: %s", callbackErr):
			default:
			}
			return
		}

		if state := strings.TrimSpace(r.FormValue("state")); state != expectedState {
			http.Error(w, "invalid oauth state", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("invalid oauth state"):
			default:
			}
			return
		}

		code := strings.TrimSpace(r.FormValue("code"))
		if code == "" {
			http.Error(w, "authorization code not found", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("authorization code not found in callback"):
			default:
			}
			return
		}

		_, _ = fmt.Fprintf(w, "<html><body><h1>OAuth complete</h1><p>You can return to the terminal now.</p><p>Code received for redirect <code>%s</code>.</p></body></html>", html.EscapeString(redirectURI))

		select {
		case codeCh <- code:
		default:
		}
	})

	server := &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case errCh <- fmt.Errorf("start callback server: %w", err):
			default:
			}
		}
	}()

	fmt.Printf("Open this URL in your browser and authorize Google Drive access:\n\n%s\n\n", authURL)
	fmt.Printf("Waiting for callback on http://%s%s\n", listenAddr, redirectPath)
	fmt.Println("If you use a Web application OAuth client in Google Cloud, add this redirect URI to the client settings first.")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	select {
	case <-waitCtx.Done():
		if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("oauth callback timed out after %s", timeout)
		}
		return "", fmt.Errorf("oauth flow cancelled")
	case err := <-errCh:
		return "", err
	case code := <-codeCh:
		return exchangeAuthorizationCode(waitCtx, cfg, code)
	}
}

func exchangeAuthorizationCode(ctx context.Context, cfg *oauth2.Config, codeInput string) (string, error) {
	code, err := extractAuthorizationCode(codeInput)
	if err != nil {
		return "", err
	}

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("exchange authorization code: %w", err)
	}

	refreshToken := strings.TrimSpace(token.RefreshToken)
	if refreshToken == "" {
		return "", fmt.Errorf("google did not return a refresh token; revoke the existing app grant, then retry with prompt=consent")
	}

	return refreshToken, nil
}

func extractAuthorizationCode(input string) (string, error) {
	clean := strings.TrimSpace(input)
	if clean == "" {
		return "", fmt.Errorf("authorization code is empty")
	}

	if strings.Contains(clean, "://") {
		parsed, err := url.Parse(clean)
		if err != nil {
			return "", fmt.Errorf("parse callback URL: %w", err)
		}
		code := strings.TrimSpace(parsed.Query().Get("code"))
		if code == "" {
			return "", fmt.Errorf("callback URL does not contain code")
		}
		return code, nil
	}

	return clean, nil
}

func redirectPathFromURI(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("parse redirect URI: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("redirect URI must be absolute, got %q", raw)
	}
	if parsed.Path == "" {
		return "/", nil
	}
	return parsed.Path, nil
}

func randomState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func printPrintOnlyInstructions(authURL, redirectURI string) {
	fmt.Printf("Open this URL in your browser and authorize Google Drive access:\n\n%s\n\n", authURL)
	fmt.Printf("Redirect URI: %s\n", redirectURI)
	fmt.Println("After Google redirects back, copy the full callback URL and run:")
	fmt.Println(`  go run ./cmd/google_drive_oauth --code '<PASTE_FULL_CALLBACK_URL>'`)
}

func printRefreshToken(refreshToken string) {
	fmt.Println()
	fmt.Println("Google Drive OAuth refresh token obtained successfully.")
	fmt.Println("Add this to .env.local:")
	fmt.Printf("GOOGLE_DRIVE_OAUTH_REFRESH_TOKEN=%s\n", refreshToken)
}
