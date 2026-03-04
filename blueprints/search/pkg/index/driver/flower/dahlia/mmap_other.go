//go:build !linux && !darwin

package dahlia

import "os"

func mmapFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func munmapFile(data []byte) error {
	return nil
}
