// storage — Cross-platform CLI for storage.liteio.dev
//
// Build: go build -o storage ./cmd/storage
// Cross:
//   GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o storage-linux-amd64   ./cmd/storage
//   GOOS=linux   GOARCH=arm64 go build -ldflags="-s -w" -o storage-linux-arm64   ./cmd/storage
//   GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o storage-darwin-amd64  ./cmd/storage
//   GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o storage-darwin-arm64  ./cmd/storage
//   GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o storage-windows-amd64.exe ./cmd/storage
//   GOOS=windows GOARCH=arm64 go build -ldflags="-s -w" -o storage-windows-arm64.exe ./cmd/storage

package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	version    = "1.0.0"
	defaultURL = "https://storage.liteio.dev"
)

var (
	baseURL   = defaultURL
	tok       string
	jsonMode  bool
	quietMode bool
	hasTTY    bool
	bold, dim, rst, red, grn, ylw, blu, cyn string
)

// ── config paths ──────────────────────────────────────────────────────

func cfgDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "storage")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "storage")
}

func tokenPath() string { return filepath.Join(cfgDir(), "token") }
func keyPath() string   { return filepath.Join(cfgDir(), "key") }
func actorPath() string { return filepath.Join(cfgDir(), "actor") }

func loadToken() {
	if tok != "" {
		return
	}
	if t := os.Getenv("STORAGE_TOKEN"); t != "" {
		tok = t
		return
	}
	if b, err := os.ReadFile(tokenPath()); err == nil {
		tok = strings.TrimSpace(string(b))
	}
}

func requireToken() {
	loadToken()
	if tok == "" {
		die("not authenticated", "Run 'storage login' or 'storage token <TOKEN>'")
	}
}

func saveFile(path, content string) {
	os.MkdirAll(cfgDir(), 0700)
	os.WriteFile(path, []byte(content), 0600)
}

func loadPrivateKey() (ed25519.PrivateKey, error) {
	b, err := os.ReadFile(keyPath())
	if err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil {
		return nil, err
	}
	return ed25519.PrivateKey(data), nil
}

func loadActor() string {
	b, _ := os.ReadFile(actorPath())
	return strings.TrimSpace(string(b))
}

// ── output ────────────────────────────────────────────────────────────

func initOutput() {
	fi, err := os.Stdout.Stat()
	hasTTY = err == nil && fi.Mode()&os.ModeCharDevice != 0
	hasColor := hasTTY && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	if hasColor {
		bold = "\033[1m"
		dim = "\033[2m"
		rst = "\033[0m"
		red = "\033[31m"
		grn = "\033[32m"
		ylw = "\033[33m"
		blu = "\033[34m"
		cyn = "\033[36m"
	}
}

func info(label, msg string) {
	if !quietMode {
		fmt.Fprintf(os.Stderr, "  %s%s%s %s\n", grn, label, rst, msg)
	}
}

func warn(msg string) {
	fmt.Fprintf(os.Stderr, "%swarning:%s %s\n", ylw, rst, msg)
}

func die(msg string, hints ...string) {
	fmt.Fprintf(os.Stderr, "%serror:%s %s\n", red, rst, msg)
	for _, h := range hints {
		fmt.Fprintf(os.Stderr, "  %s\n", h)
	}
	os.Exit(1)
}

func humanSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func relTime(ms int64) string {
	d := time.Now().Unix() - ms/1000
	switch {
	case d < 0:
		return "in the future"
	case d < 60:
		return "just now"
	case d < 3600:
		return fmt.Sprintf("%dm ago", d/60)
	case d < 86400:
		return fmt.Sprintf("%dh ago", d/3600)
	case d < 604800:
		return fmt.Sprintf("%dd ago", d/86400)
	default:
		return fmt.Sprintf("%dw ago", d/604800)
	}
}

// ── HTTP client ───────────────────────────────────────────────────────

func apiDo(method, path string, body any) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

type apiErr struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func apiReq(method, path string, body any) []byte {
	data, code, err := apiDo(method, path, body)
	if err != nil {
		die("network error", "Could not reach "+baseURL)
	}
	if code >= 400 {
		var ae apiErr
		json.Unmarshal(data, &ae)
		msg := ae.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", code)
		}
		switch code {
		case 401:
			die("authentication failed", msg, "Run 'storage login' to re-authenticate")
		case 403:
			die("permission denied", msg)
		case 404:
			die("not found", msg)
		default:
			die(fmt.Sprintf("request failed (%d)", code), msg)
		}
	}
	return data
}

func apiUpload(path string, r io.Reader, contentType string) []byte {
	req, err := http.NewRequest("PUT", baseURL+path, r)
	if err != nil {
		die("request error: " + err.Error())
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		die("upload failed", "Network error")
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var ae apiErr
		json.Unmarshal(data, &ae)
		die(fmt.Sprintf("upload failed (%d)", resp.StatusCode), ae.Message)
	}
	return data
}

func apiDownload(path string, w io.Writer) int64 {
	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		die("request error: " + err.Error())
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		die("download failed", "Network error")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		var ae apiErr
		json.Unmarshal(data, &ae)
		msg := ae.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		die("download failed", msg)
	}
	n, _ := io.Copy(w, resp.Body)
	return n
}

// ── login ─────────────────────────────────────────────────────────────

func cmdLogin(args []string) {
	actor := ""
	if len(args) > 0 {
		actor = args[0]
	}
	if actor == "" {
		actor = loadActor()
	}

	privKey, keyErr := loadPrivateKey()

	// Need to register — determine actor name
	if keyErr != nil {
		if actor == "" {
			u, _ := user.Current()
			if u != nil && u.Username != "" {
				actor = u.Username
			} else {
				actor = fmt.Sprintf("user-%d", time.Now().UnixNano()%10000)
			}
			if hasTTY {
				fmt.Fprintf(os.Stderr, "%sactor name:%s [%s] ", bold, rst, actor)
				var input string
				fmt.Scanln(&input)
				if input = strings.TrimSpace(input); input != "" {
					actor = input
				}
			}
		}

		pub, priv, _ := ed25519.GenerateKey(nil)
		pubB64 := base64.StdEncoding.EncodeToString(pub)

		_, code, netErr := apiDo("POST", "/auth/register", map[string]string{
			"actor": actor, "public_key": pubB64, "type": "human",
		})
		if netErr != nil {
			die("network error", netErr.Error())
		}
		if code == 409 {
			die("actor '"+actor+"' already registered",
				"Use a different name, or set a token: storage token <TOKEN>")
		}
		if code >= 400 {
			die("registration failed")
		}

		privKey = priv
		saveFile(keyPath(), base64.StdEncoding.EncodeToString(privKey))
		saveFile(actorPath(), actor)
		info("Registered", actor)
	}

	if actor == "" {
		die("no actor name saved", "Run: storage login <name>")
	}

	// Challenge/verify
	chData, code, err := apiDo("POST", "/auth/challenge", map[string]string{"actor": actor})
	if err != nil {
		die("network error", err.Error())
	}
	if code >= 400 {
		die("login failed", "Could not get challenge for '"+actor+"'")
	}

	var ch struct {
		ChallengeID string `json:"challenge_id"`
		Nonce       string `json:"nonce"`
	}
	json.Unmarshal(chData, &ch)

	sig := ed25519.Sign(privKey, []byte(ch.Nonce))
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	vData, code, err := apiDo("POST", "/auth/verify", map[string]any{
		"challenge_id": ch.ChallengeID,
		"actor":        actor,
		"signature":    sigB64,
	})
	if err != nil {
		die("network error", err.Error())
	}
	if code >= 400 {
		die("verification failed", "Signature mismatch — try removing ~/.config/storage/key and re-registering")
	}

	var sess struct {
		Token string `json:"token"`
	}
	json.Unmarshal(vData, &sess)

	saveFile(tokenPath(), sess.Token)
	saveFile(actorPath(), actor)
	info("Authenticated", "Token saved to "+tokenPath())
}

func cmdLogout() {
	if err := os.Remove(tokenPath()); err == nil {
		info("Logged out", "Token removed")
	} else {
		info("Already", "logged out")
	}
}

func cmdToken(args []string) {
	if len(args) == 0 {
		loadToken()
		if tok == "" {
			die("no token configured", "Run 'storage login' to authenticate")
		}
		src := tokenPath()
		if os.Getenv("STORAGE_TOKEN") != "" {
			src = "$STORAGE_TOKEN"
		}
		fmt.Printf("%ssource:%s %s\n", dim, rst, src)
		fmt.Printf("%stoken:%s  %s...\n", dim, rst, tok[:min(12, len(tok))])
		return
	}
	saveFile(tokenPath(), args[0])
	info("Saved", "Token stored in "+tokenPath())
}

// ── file operations ───────────────────────────────────────────────────

func cmdLs(args []string) {
	requireToken()
	prefix := ""
	if len(args) > 0 {
		prefix = args[0]
	}

	path := "/ls"
	if prefix != "" {
		path = "/ls/" + prefix
	}

	data := apiReq("GET", path, nil)

	if jsonMode {
		fmt.Println(string(data))
		return
	}

	var resp struct {
		Prefix  string `json:"prefix"`
		Entries []struct {
			Name      string `json:"name"`
			Type      string `json:"type"`
			Size      int64  `json:"size,omitempty"`
			UpdatedAt int64  `json:"updated_at,omitempty"`
		} `json:"entries"`
		Truncated bool `json:"truncated"`
	}
	json.Unmarshal(data, &resp)

	if len(resp.Entries) == 0 {
		info("Empty", prefix)
		return
	}

	fmt.Printf("%s%-40s %10s  %-24s  %s%s\n", bold, "NAME", "SIZE", "TYPE", "MODIFIED", rst)
	for _, e := range resp.Entries {
		size := "-"
		mod := ""
		if e.Type != "directory" {
			size = humanSize(e.Size)
			if e.UpdatedAt > 0 {
				mod = relTime(e.UpdatedAt)
			}
		}
		fmt.Printf("%-40s %10s  %-24s  %s\n", e.Name, size, e.Type, mod)
	}
	if resp.Truncated {
		info("Truncated", "More results available")
	}
}

func cmdPut(args []string) {
	requireToken()
	if len(args) < 1 {
		die("file required", "Usage: storage put <file> [path]")
	}

	file := args[0]
	dest := ""
	if len(args) > 1 {
		dest = args[1]
	} else if file != "-" {
		dest = filepath.Base(file)
	} else {
		die("path required when reading from stdin", "Usage: echo data | storage put - path/file.txt")
	}

	var r io.Reader
	var contentType string

	if file == "-" {
		r = os.Stdin
		contentType = "application/octet-stream"
	} else {
		f, err := os.Open(file)
		if err != nil {
			die("file not found: " + file)
		}
		defer f.Close()
		r = f
		ext := filepath.Ext(file)
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	data := apiUpload("/f/"+dest, r, contentType)

	if jsonMode {
		fmt.Println(string(data))
		return
	}

	var resp struct {
		Path string `json:"path"`
		Size int64  `json:"size"`
	}
	json.Unmarshal(data, &resp)
	info("Uploaded", fmt.Sprintf("%s (%s)", resp.Path, humanSize(resp.Size)))
}

func cmdGet(args []string) {
	requireToken()
	if len(args) < 1 {
		die("path required", "Usage: storage get <path> [local-file]")
	}

	src := args[0]
	dest := ""
	if len(args) > 1 {
		dest = args[1]
	} else {
		dest = filepath.Base(src)
	}

	if dest == "-" {
		apiDownload("/f/"+src, os.Stdout)
		return
	}

	dir := filepath.Dir(dest)
	if dir != "." && dir != "" {
		os.MkdirAll(dir, 0755)
	}

	f, err := os.Create(dest)
	if err != nil {
		die("cannot create " + dest)
	}
	defer f.Close()
	n := apiDownload("/f/"+src, f)
	info("Downloaded", fmt.Sprintf("%s (%s)", filepath.Base(src), humanSize(n)))
}

func cmdCat(args []string) {
	if len(args) < 1 {
		die("path required", "Usage: storage cat <path>")
	}
	requireToken()
	apiDownload("/f/"+args[0], os.Stdout)
}

func cmdRm(args []string) {
	requireToken()
	if len(args) < 1 {
		die("path required", "Usage: storage rm <path...>")
	}
	for _, p := range args {
		if strings.HasPrefix(p, "-") {
			continue
		}
		apiReq("DELETE", "/f/"+p, nil)
		info("Deleted", p)
	}
}

func cmdMv(args []string) {
	requireToken()
	if len(args) < 2 {
		die("usage: storage mv <from> <to>")
	}
	data := apiReq("POST", "/mv", map[string]string{"from": args[0], "to": args[1]})
	if jsonMode {
		fmt.Println(string(data))
		return
	}
	info("Moved", args[0]+" -> "+args[1])
}

func cmdFind(args []string) {
	requireToken()
	if len(args) < 1 {
		die("query required", "Usage: storage find <query>")
	}

	data := apiReq("GET", "/find?q="+url.QueryEscape(strings.Join(args, " ")), nil)

	if jsonMode {
		fmt.Println(string(data))
		return
	}

	var resp struct {
		Results []struct {
			Path string `json:"path"`
			Name string `json:"name"`
		} `json:"results"`
	}
	json.Unmarshal(data, &resp)

	if len(resp.Results) == 0 {
		info("No results", "")
		return
	}
	for _, r := range resp.Results {
		fmt.Println(r.Path)
	}
}

func cmdShare(args []string) {
	requireToken()
	if len(args) < 1 {
		die("path required", "Usage: storage share <path> [--ttl 3600]")
	}

	path := args[0]
	ttl := 3600
	for i, a := range args {
		if (a == "--ttl" || a == "--expires" || a == "-x") && i+1 < len(args) {
			if v, err := strconv.Atoi(args[i+1]); err == nil {
				ttl = v
			}
		}
	}

	data := apiReq("POST", "/share", map[string]any{"path": path, "ttl": ttl})

	if jsonMode {
		fmt.Println(string(data))
		return
	}

	var resp struct {
		URL string `json:"url"`
		TTL int    `json:"ttl"`
	}
	json.Unmarshal(data, &resp)
	fmt.Println(resp.URL)
	if !quietMode {
		info("Expires", fmt.Sprintf("in %ds", resp.TTL))
	}
}

func cmdStat() {
	requireToken()
	data := apiReq("GET", "/stat", nil)

	if jsonMode {
		fmt.Println(string(data))
		return
	}

	var resp struct {
		Files int64 `json:"files"`
		Bytes int64 `json:"bytes"`
	}
	json.Unmarshal(data, &resp)
	fmt.Printf("%s%-12s%s %d\n", dim, "Files:", rst, resp.Files)
	fmt.Printf("%s%-12s%s %s\n", dim, "Used:", rst, humanSize(resp.Bytes))
}

// ── API key management ────────────────────────────────────────────────

func cmdKey(args []string) {
	if len(args) < 1 {
		die("subcommand required", "Usage: storage key <create|list|revoke>")
	}

	switch args[0] {
	case "create", "new":
		requireToken()
		name := "default"
		var prefix string
		rest := args[1:]
		for i := 0; i < len(rest); i++ {
			switch rest[i] {
			case "--prefix", "-p":
				if i+1 < len(rest) {
					i++
					prefix = rest[i]
				}
			default:
				if !strings.HasPrefix(rest[i], "-") {
					name = rest[i]
				}
			}
		}
		body := map[string]any{"name": name}
		if prefix != "" {
			body["prefix"] = prefix
		}
		data := apiReq("POST", "/auth/keys", body)

		if jsonMode {
			fmt.Println(string(data))
			return
		}

		var resp struct {
			Token string `json:"token"`
		}
		json.Unmarshal(data, &resp)
		fmt.Printf("\n%sAPI Key:%s %s\n\n", bold, rst, resp.Token)
		fmt.Printf("%sSave this key — it won't be shown again.%s\n", dim, rst)
		fmt.Printf("Use: %sexport STORAGE_TOKEN=%s%s\n", cyn, resp.Token, rst)

	case "list", "ls":
		requireToken()
		data := apiReq("GET", "/auth/keys", nil)

		if jsonMode {
			fmt.Println(string(data))
			return
		}

		var resp struct {
			Keys []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Prefix    string `json:"prefix"`
				CreatedAt int64  `json:"created_at"`
			} `json:"keys"`
		}
		json.Unmarshal(data, &resp)

		if len(resp.Keys) == 0 {
			info("No API keys", "Create one with: storage key create <name>")
			return
		}
		fmt.Printf("%s%-10s %-20s %-20s  %s%s\n", bold, "ID", "NAME", "PREFIX", "CREATED", rst)
		for _, k := range resp.Keys {
			fmt.Printf("%-10s %-20s %-20s  %s\n", k.ID, k.Name, k.Prefix, relTime(k.CreatedAt))
		}

	case "revoke", "delete", "rm":
		requireToken()
		if len(args) < 2 {
			die("key ID required", "Usage: storage key revoke <id>")
		}
		apiReq("DELETE", "/auth/keys/"+args[1], nil)
		info("Revoked", "API key "+args[1])

	default:
		die("unknown key command: "+args[0], "Available: create, list, revoke")
	}
}

// ── help & version ────────────────────────────────────────────────────

func cmdVersion() {
	fmt.Println("storage " + version)
}

func cmdHelp() {
	fmt.Printf(`%sstorage%s — CLI for storage.liteio.dev
%s%s%s

%sUSAGE%s
  storage <command> [options] [args]

%sCOMMANDS%s
  %slogin%s [name]                Authenticate (Ed25519 key pair)
  %slogout%s                     Remove saved credentials
  %stoken%s [<token>]             Show or set authentication token

  %sls%s [path]                   List directory contents
  %sput%s <file> [path]           Upload a file (or stdin with -)
  %sget%s <path> [dest]           Download a file
  %scat%s <path>                  Print file to stdout
  %srm%s <path...>               Delete files
  %smv%s <from> <to>              Move/rename a file
  %sfind%s <query>               Search by name
  %sshare%s <path>               Create a temporary link
  %sstat%s                       Show storage usage

  %skey create%s <name>           Create an API key
  %skey list%s                    List API keys
  %skey revoke%s <id>             Revoke an API key

%sFLAGS%s
  --json, -j          JSON output
  --quiet, -q         Suppress non-essential output
  --token, -t <tok>   Use specific token
  --endpoint <url>    API endpoint (default: %s)
  --no-color          Disable colors
  --version, -V       Print version
  --help, -h          Show this help

%sEXAMPLES%s
  storage login
  storage put report.pdf docs/report.pdf
  storage ls docs/
  storage get docs/report.pdf
  storage share docs/report.pdf --ttl 86400
  echo "hello" | storage put - greetings/hello.txt

%sENV%s
  STORAGE_TOKEN       API key or session token
  STORAGE_ENDPOINT    API base URL
  NO_COLOR            Disable colored output
`,
		bold, rst, dim, defaultURL, rst,
		bold, rst,
		bold, rst,
		bold, rst, bold, rst, bold, rst,
		bold, rst, bold, rst, bold, rst, bold, rst, bold, rst, bold, rst, bold, rst, bold, rst, bold, rst,
		bold, rst, bold, rst, bold, rst,
		bold, rst, defaultURL,
		bold, rst,
		bold, rst,
	)
}

// ── main ──────────────────────────────────────────────────────────────

func main() {
	initOutput()

	if e := os.Getenv("STORAGE_ENDPOINT"); e != "" {
		baseURL = e
	}

	var args []string
	osArgs := os.Args[1:]
	for i := 0; i < len(osArgs); i++ {
		switch osArgs[i] {
		case "--json", "-j":
			jsonMode = true
		case "--quiet", "-q":
			quietMode = true
		case "--no-color":
			bold = ""
			dim = ""
			rst = ""
			red = ""
			grn = ""
			ylw = ""
			blu = ""
			cyn = ""
		case "--token", "-t":
			if i+1 < len(osArgs) {
				i++
				tok = osArgs[i]
			}
		case "--endpoint":
			if i+1 < len(osArgs) {
				i++
				baseURL = osArgs[i]
			}
		case "--version", "-V":
			cmdVersion()
			os.Exit(0)
		case "--help", "-h":
			if len(args) == 0 {
				cmdHelp()
				os.Exit(0)
			}
			args = append(args, osArgs[i])
		default:
			args = append(args, osArgs[i])
		}
	}

	if len(args) == 0 {
		cmdHelp()
		os.Exit(0)
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "login":
		cmdLogin(cmdArgs)
	case "logout":
		cmdLogout()
	case "token":
		cmdToken(cmdArgs)
	case "ls", "list":
		cmdLs(cmdArgs)
	case "put", "upload", "push":
		cmdPut(cmdArgs)
	case "get", "download", "pull":
		cmdGet(cmdArgs)
	case "cat", "read":
		cmdCat(cmdArgs)
	case "rm", "delete", "del":
		cmdRm(cmdArgs)
	case "mv", "move", "rename":
		cmdMv(cmdArgs)
	case "find", "search":
		cmdFind(cmdArgs)
	case "share", "sign":
		cmdShare(cmdArgs)
	case "stat", "stats":
		cmdStat()
	case "key", "keys":
		cmdKey(cmdArgs)
	case "help":
		cmdHelp()
	case "version":
		cmdVersion()
	default:
		die("unknown command: "+cmd, "Run 'storage help' for available commands")
	}
}
