package usagi

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	segmentDirName   = ".usagi-segments"
	segmentFilePref  = "segment-"
	segmentFileExt   = ".usg"
	segmentIDDigits  = 6
	defaultSegSizeMB = 64
)

func (b *bucket) segmentDir() string {
	return filepath.Join(b.dir, segmentDirName)
}

func segmentFileName(id int64) string {
	return fmt.Sprintf("%s%0*d%s", segmentFilePref, segmentIDDigits, id, segmentFileExt)
}

func parseSegmentID(name string) (int64, bool) {
	if !strings.HasPrefix(name, segmentFilePref) || !strings.HasSuffix(name, segmentFileExt) {
		return 0, false
	}
	num := strings.TrimSuffix(strings.TrimPrefix(name, segmentFilePref), segmentFileExt)
	id, err := strconv.ParseInt(num, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func (b *bucket) listSegments() ([]int64, error) {
	entries, err := os.ReadDir(b.segmentDir())
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if id, ok := parseSegmentID(e.Name()); ok {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func (b *bucket) segmentPath(id int64) string {
	return filepath.Join(b.segmentDir(), segmentFileName(id))
}
