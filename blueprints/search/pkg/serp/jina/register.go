package jina

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/go-mizu/mizu/blueprints/search/pkg/serp"
)

//go:embed get_key.py
var getKeyScript []byte

func init() {
	serp.RegisterRegistrar("jina", &registrar{})
}

type registrar struct{}

var jinaKeyRe = regexp.MustCompile(`jina_[a-f0-9]{32}[a-zA-Z0-9_-]+`)

// Register gets a Jina AI API key by running the embedded Python patchright script.
//
// Strategy:
//  1. Launch patchright browser, navigate to jina.ai/?newKey
//  2. context.route() intercepts keygen.jina.ai POST, captures cf-turnstile-response
//  3. If rate-limited (429), replays keygen POST through SOCKS5 proxies
//  4. Returns key with 10M tokens (10-year trial)
//
// Requirements: python3 + patchright (pip install patchright)
func (r *registrar) Register(email, password string, verbose bool) (string, error) {
	// Write embedded script to temp file
	tmpDir, err := os.MkdirTemp("", "jina-key-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "get_key.py")
	if err := os.WriteFile(scriptPath, getKeyScript, 0700); err != nil {
		return "", fmt.Errorf("write script: %w", err)
	}

	args := []string{scriptPath}
	// Headless on Linux (no display server)
	if runtime.GOOS == "linux" {
		args = append(args, "--headless")
	}
	if verbose {
		args = append(args, "--verbose")
	}
	args = append(args, "--timeout", "90")

	if verbose {
		fmt.Printf("  running: python3 %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("python3", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outStr := string(output)
		if strings.Contains(outStr, "RATE_IP_LIMIT") || strings.Contains(outStr, "rate limit") {
			return "", fmt.Errorf("keygen.jina.ai rate limited — the script should auto-retry via proxy")
		}
		if verbose {
			fmt.Printf("  script output:\n%s\n", outStr)
		}
		return "", fmt.Errorf("get_key.py failed: %w\n%s", err, outStr)
	}

	// Parse output: KEY:<key>
	outStr := strings.TrimSpace(string(output))
	for _, line := range strings.Split(outStr, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "KEY:") {
			key := strings.TrimPrefix(line, "KEY:")
			if jinaKeyRe.MatchString(key) {
				return key, nil
			}
		}
	}

	return "", fmt.Errorf("no key found in script output:\n%s", outStr)
}

func (r *registrar) VerifyAndGetKey(email, password, emailBody string, verbose bool) (string, error) {
	return "", fmt.Errorf("jina does not require email verification")
}
