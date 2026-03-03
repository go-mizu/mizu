//go:build !linux && !darwin

package lotus

import "os"

func mmapFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func mmapRelease(data []byte) error {
	return nil
}
