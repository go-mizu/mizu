//go:build onnx

package onnx

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	hfBase    = "https://huggingface.co"
	modelRepo = "sentence-transformers/all-MiniLM-L6-v2"

	modelFile = "model.onnx"
	vocabFile = "vocab.txt"

	modelURL = hfBase + "/" + modelRepo + "/resolve/main/onnx/model.onnx"
	vocabURL = hfBase + "/" + modelRepo + "/resolve/main/vocab.txt"
)

// EnsureModel downloads model.onnx and vocab.txt to dir if they don't exist.
func EnsureModel(dir string) (modelPath, vocabPath string, err error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("onnx: mkdir %s: %w", dir, err)
	}

	modelPath = filepath.Join(dir, modelFile)
	vocabPath = filepath.Join(dir, vocabFile)

	if err := downloadIfMissing(modelURL, modelPath, "model.onnx (~90MB)"); err != nil {
		return "", "", err
	}
	if err := downloadIfMissing(vocabURL, vocabPath, "vocab.txt"); err != nil {
		return "", "", err
	}
	return modelPath, vocabPath, nil
}

func downloadIfMissing(url, path, desc string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}

	fmt.Fprintf(os.Stderr, "onnx: downloading %s from %s\n", desc, url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("onnx: download %s: %w", desc, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("onnx: download %s: HTTP %d", desc, resp.StatusCode)
	}

	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("onnx: create %s: %w", tmp, err)
	}

	n, err := io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmp)
		return fmt.Errorf("onnx: write %s: %w", desc, err)
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("onnx: rename %s: %w", desc, err)
	}

	fmt.Fprintf(os.Stderr, "onnx: downloaded %s (%d bytes)\n", desc, n)
	return nil
}
