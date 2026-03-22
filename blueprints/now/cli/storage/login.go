package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	oauthClientID = "storage-cli"
	oauthScope    = "storage:read storage:write storage:admin"
	callbackPath  = "/callback"
	authTimeout   = 5 * time.Minute
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate via browser (OAuth)",
		Long: `Opens your browser to authenticate with Liteio Storage.
Uses OAuth 2.0 with PKCE. Your credentials never touch the CLI.
The token is saved locally and expires after 90 days.`,
		Example: "  storage login",
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			return loginOAuth(d)
		}),
	}
}

func loginOAuth(d *Deps) error {
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return &CLIError{Code: ExitError, Msg: "failed to generate PKCE verifier"}
	}
	codeChallenge := computeCodeChallenge(codeVerifier)

	state, err := generateState()
	if err != nil {
		return &CLIError{Code: ExitError, Msg: "failed to generate state"}
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return &CLIError{Code: ExitError, Msg: "failed to start local server", Hint: err.Error()}
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d%s", port, callbackPath)

	authURL := fmt.Sprintf("%s/oauth/authorize?%s",
		d.Config.Endpoint,
		url.Values{
			"response_type":         {"code"},
			"client_id":             {oauthClientID},
			"redirect_uri":          {redirectURI},
			"code_challenge":        {codeChallenge},
			"code_challenge_method": {"S256"},
			"scope":                 {oauthScope},
			"state":                 {state},
		}.Encode(),
	)

	type callbackResult struct {
		code string
		err  error
	}
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		if q.Get("state") != state {
			w.WriteHeader(400)
			fmt.Fprint(w, errorHTML("State mismatch", "This request may have been tampered with. Please try again."))
			resultCh <- callbackResult{err: fmt.Errorf("state mismatch")}
			return
		}

		if errCode := q.Get("error"); errCode != "" {
			desc := q.Get("error_description")
			if desc == "" {
				desc = errCode
			}
			w.WriteHeader(400)
			fmt.Fprint(w, errorHTML("Authorization failed", desc))
			resultCh <- callbackResult{err: fmt.Errorf("oauth error: %s", desc)}
			return
		}

		code := q.Get("code")
		if code == "" {
			w.WriteHeader(400)
			fmt.Fprint(w, errorHTML("Missing code", "No authorization code received."))
			resultCh <- callbackResult{err: fmt.Errorf("no authorization code")}
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, successHTML())
		resultCh <- callbackResult{code: code}
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener)

	d.Out.Info("Opening", "browser for authentication...")

	if !openBrowser(authURL) {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "  Open this URL in your browser:\n\n")
		fmt.Fprintf(os.Stderr, "  %s\n\n", authURL)
		fmt.Fprintln(os.Stderr, d.Out.dim("  Waiting for authentication..."))
	} else {
		fmt.Fprintln(os.Stderr, d.Out.dim("  Waiting for authentication..."))
	}

	ctx, cancel := context.WithTimeout(context.Background(), authTimeout)
	defer cancel()

	var result callbackResult
	select {
	case result = <-resultCh:
	case <-ctx.Done():
		server.Shutdown(context.Background())
		return &CLIError{Code: ExitError, Msg: "authentication timed out", Hint: "Try again with: storage login"}
	}

	server.Shutdown(context.Background())

	if result.err != nil {
		return &CLIError{Code: ExitAuth, Msg: "authentication failed", Hint: result.err.Error()}
	}

	tokenResp, err := exchangeCode(d, result.code, redirectURI, codeVerifier)
	if err != nil {
		return err
	}

	if err := SaveToken(tokenResp.AccessToken); err != nil {
		return err
	}

	expiresIn := ""
	if tokenResp.ExpiresIn > 0 {
		days := tokenResp.ExpiresIn / 86400
		if days > 0 {
			expiresIn = fmt.Sprintf(" (expires in %d days)", days)
		}
	}
	d.Out.Info("Authenticated", "token saved to "+TokenFile()+expiresIn)
	return nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

func exchangeCode(d *Deps, code, redirectURI, codeVerifier string) (*tokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {oauthClientID},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequest("POST", d.Config.Endpoint+"/oauth/token",
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, &CLIError{Code: ExitNetwork, Msg: "failed to create token request"}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := d.Client.HTTPClient.Do(req)
	if err != nil {
		return nil, &CLIError{Code: ExitNetwork, Msg: "failed to exchange code", Hint: err.Error()}
	}
	defer resp.Body.Close()

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, &CLIError{Code: ExitError, Msg: "failed to parse token response"}
	}

	if tokenResp.AccessToken == "" {
		return nil, &CLIError{Code: ExitAuth, Msg: "no access token in response"}
	}

	return &tokenResp, nil
}

func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func computeCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(url string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	default:
		return false
	}
	return cmd.Start() == nil
}

func successHTML() string {
	return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Authenticated</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:system-ui,sans-serif;background:#fafafa;color:#111;
display:flex;align-items:center;justify-content:center;min-height:100vh}
@media(prefers-color-scheme:dark){body{background:#111;color:#eee}.card{background:#1a1a1a;border-color:#333}.sub{color:#999}}
.card{background:#fff;border:1px solid #ddd;padding:2.5rem;width:100%;max-width:420px;text-align:center}
h1{font-size:1.1rem;font-weight:600;margin-bottom:0.5rem}
.sub{font-size:0.85rem;color:#666;line-height:1.5}
.check{font-size:2rem;margin-bottom:1rem}
.brand{font-size:0.75rem;text-transform:uppercase;letter-spacing:0.08em;color:#999;margin-bottom:1rem}
</style></head><body>
<div class="card">
<div class="brand">Liteio Storage</div>
<div class="check">&#10003;</div>
<h1>Authentication successful</h1>
<p class="sub">You can close this tab and return to your terminal.</p>
</div>
<script>setTimeout(()=>window.close(),2000)</script>
</body></html>`
}

func errorHTML(title, message string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:system-ui,sans-serif;background:#fafafa;color:#111;
display:flex;align-items:center;justify-content:center;min-height:100vh}
@media(prefers-color-scheme:dark){body{background:#111;color:#eee}.card{background:#1a1a1a;border-color:#333}.sub{color:#999}}
.card{background:#fff;border:1px solid #ddd;padding:2.5rem;width:100%%;max-width:420px;text-align:center}
h1{font-size:1.1rem;font-weight:600;margin-bottom:0.5rem}
.sub{font-size:0.85rem;color:#666;line-height:1.5}
.brand{font-size:0.75rem;text-transform:uppercase;letter-spacing:0.08em;color:#999;margin-bottom:1rem}
</style></head><body>
<div class="card">
<div class="brand">Liteio Storage</div>
<h1>%s</h1>
<p class="sub">%s</p>
</div>
</body></html>`, html.EscapeString(title), html.EscapeString(title), html.EscapeString(message))
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove saved credentials",
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RemoveToken(); err != nil {
				if os.IsNotExist(err) {
					d.Out.Info("Already", "logged out")
					return nil
				}
				return err
			}
			d.Out.Info("Logged out", "token removed from "+TokenFile())
			return nil
		}),
	}
}

func newTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "token [token]",
		Short: "Show or set authentication token",
		Long: `Show the current token source, or save a token directly.

Useful for CI/headless environments:
  storage token sk_your_api_key_here`,
		Args: cobra.MaximumNArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()

			if len(args) == 0 {
				cfg := d.Config
				if cfg.Token == "" {
					return &CLIError{
						Code: ExitAuth,
						Msg:  "no token configured",
						Hint: "Run 'storage login' to authenticate\nOr set directly: storage token <token>",
					}
				}

				source := "unknown"
				if globalFlags.token != "" {
					source = "--token flag"
				} else if os.Getenv("STORAGE_TOKEN") != "" {
					source = "$STORAGE_TOKEN"
				} else if _, err := os.Stat(TokenFile()); err == nil {
					source = TokenFile()
				}

				truncated := cfg.Token
				if len(truncated) > 12 {
					truncated = truncated[:12] + "..."
				}
				fmt.Printf("%s %s\n", d.Out.dim("source:"), source)
				fmt.Printf("%s  %s\n", d.Out.dim("token:"), truncated)
				return nil
			}

			if err := SaveToken(args[0]); err != nil {
				return err
			}
			d.Out.Info("Saved", "token stored in "+TokenFile())
			return nil
		}),
	}
}
