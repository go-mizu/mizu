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

type segmentRef struct {
	shard int
	id    int64
}

func (b *bucket) segmentDir() string {
	return filepath.Join(b.dir, segmentDirName)
}

func segmentFileName(shard int, id int64) string {
	if shard < 0 {
		shard = 0
	}
	return fmt.Sprintf("%s%d-%0*d%s", segmentFilePref, shard, segmentIDDigits, id, segmentFileExt)
}

func parseSegmentID(name string) (segmentRef, bool) {
	if !strings.HasPrefix(name, segmentFilePref) || !strings.HasSuffix(name, segmentFileExt) {
		return segmentRef{}, false
	}
	raw := strings.TrimSuffix(strings.TrimPrefix(name, segmentFilePref), segmentFileExt)
	shard := 0
	idPart := raw
	if parts := strings.Split(raw, "-"); len(parts) == 2 {
		if n, err := strconv.Atoi(parts[0]); err == nil && n >= 0 {
			shard = n
			idPart = parts[1]
		}
	}
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil {
		return segmentRef{}, false
	}
	return segmentRef{shard: shard, id: id}, true
}

func (b *bucket) listSegments() ([]segmentRef, error) {
	entries, err := os.ReadDir(b.segmentDir())
	if err != nil {
		return nil, err
	}
	refs := make([]segmentRef, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if ref, ok := parseSegmentID(e.Name()); ok {
			refs = append(refs, ref)
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].shard != refs[j].shard {
			return refs[i].shard < refs[j].shard
		}
		return refs[i].id < refs[j].id
	})
	return refs, nil
}

func (b *bucket) segmentPath(shard int, id int64) string {
	return filepath.Join(b.segmentDir(), segmentFileName(shard, id))
}

func segmentKey(shard int, id int64) string {
	return fmt.Sprintf("%d:%d", shard, id)
}
