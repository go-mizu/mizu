//go:build windows

// File: driver/local/delete_optimized_windows.go
// Windows fallback for delete operations - uses standard library functions.
package local

import "os"

// deleteWithUnlink falls back to os.Remove on Windows.
func deleteWithUnlink(path string) error {
	return os.Remove(path)
}

// deleteRecursiveFast falls back to os.RemoveAll on Windows.
func deleteRecursiveFast(root string) error {
	return os.RemoveAll(root)
}

// batchDeleteFiles falls back to sequential deletes on Windows.
func batchDeleteFiles(files []string) error {
	for _, f := range files {
		os.Remove(f)
	}
	return nil
}
