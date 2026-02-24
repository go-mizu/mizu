package hn

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ParquetDownloadProgress reports progress for a parquet snapshot download.
type ParquetDownloadProgress struct {
	Path         string
	RemoteSize   int64
	LocalSize    int64 // current on-disk size
	SessionBytes int64 // bytes downloaded during this invocation
	Resumed      bool
	Complete     bool
	Skipped      bool
	SpeedBPS     float64
	Elapsed      time.Duration
	Detail       string
}

// ParquetDownloadResult describes the local parquet download outcome.
type ParquetDownloadResult struct {
	Path         string
	Remote       *RemoteParquetInfo
	LocalSize    int64
	Downloaded   int64
	Resumed      bool
	Skipped      bool
	UsedRangeGET bool
}

func (c Config) DownloadParquet(ctx context.Context, force bool, cb func(ParquetDownloadProgress)) (*ParquetDownloadResult, error) {
	cfg := c.WithDefaults()
	if err := cfg.EnsureRawDirs(); err != nil {
		return nil, fmt.Errorf("prepare directories: %w", err)
	}
	remote, err := cfg.HeadParquet(ctx)
	if err != nil {
		return nil, err
	}

	dest := cfg.RawParquetPath()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return nil, fmt.Errorf("create raw dir: %w", err)
	}

	if force {
		_ = os.Remove(dest)
	}

	localSize, _ := fileSize(dest)
	if remote.Size > 0 {
		switch {
		case localSize == remote.Size && localSize > 0:
			if cb != nil {
				cb(ParquetDownloadProgress{Path: dest, RemoteSize: remote.Size, LocalSize: localSize, Resumed: false, Complete: true, Skipped: true})
			}
			return &ParquetDownloadResult{Path: dest, Remote: remote, LocalSize: localSize, Downloaded: 0, Skipped: true}, nil
		case localSize > remote.Size:
			return nil, fmt.Errorf("local file larger than remote (%d > %d); delete %s or use --force", localSize, remote.Size, dest)
		case localSize > 0 && !remote.AcceptRanges:
			return nil, fmt.Errorf("remote source does not advertise range support; cannot resume partial file %s", dest)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.ParquetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create GET request: %w", err)
	}
	resumed := localSize > 0
	if resumed {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", localSize))
	}
	resp, err := cfg.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET parquet source: %w", err)
	}
	defer resp.Body.Close()

	usedRange := false
	switch {
	case resumed && resp.StatusCode == http.StatusPartialContent:
		usedRange = true
	case resumed && resp.StatusCode == http.StatusOK:
		// Server ignored Range. Only safe to restart if local file is empty, which it isn't.
		return nil, fmt.Errorf("server ignored range request and returned 200 for resume download")
	case !resumed && resp.StatusCode == http.StatusOK:
		// OK
	default:
		return nil, fmt.Errorf("GET parquet source returned %d", resp.StatusCode)
	}

	flags := os.O_CREATE | os.O_WRONLY
	if resumed {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	out, err := os.OpenFile(dest, flags, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open destination: %w", err)
	}
	defer out.Close()

	start := time.Now()
	lastTick := time.Now()
	var sessionBytes int64
	buf := make([]byte, 4*1024*1024)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			wn, werr := out.Write(buf[:n])
			if werr != nil {
				return nil, fmt.Errorf("write destination: %w", werr)
			}
			sessionBytes += int64(wn)
			localNow := localSize + sessionBytes
			if cb != nil && (time.Since(lastTick) >= 500*time.Millisecond) {
				elapsed := time.Since(start)
				speed := 0.0
				if elapsed > 0 {
					speed = float64(sessionBytes) / elapsed.Seconds()
				}
				cb(ParquetDownloadProgress{
					Path:         dest,
					RemoteSize:   remote.Size,
					LocalSize:    localNow,
					SessionBytes: sessionBytes,
					Resumed:      resumed,
					SpeedBPS:     speed,
					Elapsed:      elapsed,
				})
				lastTick = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("read response body: %w", readErr)
		}
	}

	if err := out.Close(); err != nil {
		return nil, fmt.Errorf("close destination: %w", err)
	}

	finalSize, _ := fileSize(dest)
	if remote.Size > 0 && finalSize != remote.Size {
		return nil, fmt.Errorf("download incomplete: got %d bytes, expected %d", finalSize, remote.Size)
	}
	if cb != nil {
		elapsed := time.Since(start)
		speed := 0.0
		if elapsed > 0 {
			speed = float64(sessionBytes) / elapsed.Seconds()
		}
		cb(ParquetDownloadProgress{
			Path:         dest,
			RemoteSize:   remote.Size,
			LocalSize:    finalSize,
			SessionBytes: sessionBytes,
			Resumed:      resumed,
			Complete:     true,
			SpeedBPS:     speed,
			Elapsed:      elapsed,
			Detail:       strings.TrimSpace(resp.Header.Get("ETag")),
		})
	}

	return &ParquetDownloadResult{
		Path:         dest,
		Remote:       remote,
		LocalSize:    finalSize,
		Downloaded:   sessionBytes,
		Resumed:      resumed,
		UsedRangeGET: usedRange,
	}, nil
}
