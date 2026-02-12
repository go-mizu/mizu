package openlibrarydump

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const openLibraryDataBase = "https://openlibrary.org/data"

type DumpSpec struct {
	Name        string
	LatestURL   string
	ResolvedURL string
	SizeBytes   int64
}

func ResolveLatestDumpSpecs(ctx context.Context) ([]DumpSpec, error) {
	input := []DumpSpec{
		{Name: "authors", LatestURL: openLibraryDataBase + "/ol_dump_authors_latest.txt.gz"},
		{Name: "works", LatestURL: openLibraryDataBase + "/ol_dump_works_latest.txt.gz"},
		{Name: "editions", LatestURL: openLibraryDataBase + "/ol_dump_editions_latest.txt.gz"},
	}
	out := make([]DumpSpec, 0, len(input))
	for _, s := range input {
		resolved, size, err := resolveURLAndSize(ctx, s.LatestURL)
		if err != nil {
			return nil, fmt.Errorf("resolve %s dump: %w", s.Name, err)
		}
		s.ResolvedURL = resolved
		s.SizeBytes = size
		out = append(out, s)
	}
	return out, nil
}

func DownloadSpec(ctx context.Context, spec DumpSpec, targetDir string) (string, error) {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("create target dir: %w", err)
	}
	filename := filenameFromURL(spec.ResolvedURL)
	if filename == "" {
		return "", fmt.Errorf("could not infer filename for %s", spec.ResolvedURL)
	}
	target := filepath.Join(targetDir, filename)
	if done, err := ensureReusableTarget(targetDir, spec, target); err == nil && done {
		return target, nil
	}

	runOnce := func() error {
		if tool, err := exec.LookPath("wget"); err == nil {
			cmd := exec.CommandContext(ctx, tool,
				"-c",
				"--tries=0",
				"--timeout=30",
				"--read-timeout=30",
				"--waitretry=2",
				"--retry-connrefused",
				"--progress=bar:force:noscroll",
				"-O", target,
				spec.ResolvedURL,
			)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("wget download failed for %s: %w", spec.Name, err)
			}
			return nil
		}

		curlPath, err := exec.LookPath("curl")
		if err != nil {
			return fmt.Errorf("neither wget nor curl is available")
		}
		cmd := exec.CommandContext(ctx, curlPath,
			"-L",
			"--fail",
			"--retry", "1000",
			"--retry-all-errors",
			"--retry-delay", "3",
			"--continue-at", "-",
			"-o", target,
			spec.ResolvedURL,
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("curl download failed for %s: %w", spec.Name, err)
		}
		return nil
	}

	stallCount := 0
	for {
		done, currentSize, err := isComplete(target, spec.SizeBytes)
		if err != nil {
			return "", err
		}
		if done {
			return target, nil
		}

		before := currentSize
		if err := runOnce(); err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			_, after, statErr := isComplete(target, spec.SizeBytes)
			if statErr != nil {
				return "", statErr
			}
			if after > before {
				fmt.Fprintf(os.Stderr, "[WARN] %s download interrupted, resuming (%s/%s)\n", spec.Name, FormatBytes(after), FormatBytes(spec.SizeBytes))
				stallCount = 0
				continue
			}
			stallCount++
			if stallCount >= 3 {
				return "", err
			}
			fmt.Fprintf(os.Stderr, "[WARN] %s download made no progress, retrying (%d/3)\n", spec.Name, stallCount)
			time.Sleep(2 * time.Second)
			continue
		}

		done, _, err = isComplete(target, spec.SizeBytes)
		if err != nil {
			return "", err
		}
		if done {
			return target, nil
		}
	}
}

func isComplete(path string, expected int64) (bool, int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, err
	}
	size := info.Size()
	if expected > 0 && size >= expected {
		return true, size, nil
	}
	return false, size, nil
}

func resolveURLAndSize(ctx context.Context, rawURL string) (string, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return "", 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	size := int64(0)
	if cl := strings.TrimSpace(resp.Header.Get("Content-Length")); cl != "" {
		if n, parseErr := strconv.ParseInt(cl, 10, 64); parseErr == nil {
			size = n
		}
	}
	return resp.Request.URL.String(), size, nil
}

func filenameFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	base := filepath.Base(u.Path)
	if base == "." || base == "/" || base == "" {
		return ""
	}
	return base
}

func FormatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func ensureReusableTarget(targetDir string, spec DumpSpec, target string) (bool, error) {
	alias := filepath.Join(targetDir, fmt.Sprintf("ol_dump_%s_latest.txt.gz", spec.Name))
	targetSize := int64(-1)
	if info, err := os.Stat(target); err == nil {
		targetSize = info.Size()
	}
	if targetSize >= 0 && (spec.SizeBytes == 0 || targetSize >= spec.SizeBytes) {
		return true, nil
	}

	if info, err := os.Stat(alias); err == nil {
		aliasSize := info.Size()
		if spec.SizeBytes == 0 || aliasSize >= spec.SizeBytes || aliasSize > targetSize {
			_ = os.Remove(target)
			if err := os.Rename(alias, target); err != nil {
				return false, fmt.Errorf("reuse latest alias for %s: %w", spec.Name, err)
			}
			if spec.SizeBytes == 0 || aliasSize >= spec.SizeBytes {
				return true, nil
			}
			return false, nil
		}
	}

	if targetSize >= 0 {
		if spec.SizeBytes == 0 || targetSize >= spec.SizeBytes {
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

func validateDownloadedSize(spec DumpSpec, path string) error {
	if spec.SizeBytes == 0 {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() != spec.SizeBytes {
		return fmt.Errorf("%s size mismatch: got %d want %d", spec.Name, info.Size(), spec.SizeBytes)
	}
	return nil
}
