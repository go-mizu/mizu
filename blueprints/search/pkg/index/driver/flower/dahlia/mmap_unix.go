//go:build linux || darwin

package dahlia

import (
	"os"

	"golang.org/x/sys/unix"
)

func mmapFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	if size == 0 {
		return nil, nil
	}
	data, err := unix.Mmap(int(f.Fd()), 0, int(size), unix.PROT_READ, unix.MAP_PRIVATE)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func munmapFile(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return unix.Munmap(data)
}
