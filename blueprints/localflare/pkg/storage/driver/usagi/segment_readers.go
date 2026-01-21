package usagi

import (
	"os"
	"sync"
)

type readerPool struct {
	mu    sync.Mutex
	pool  *sync.Pool
	files []*os.File
}

func newReaderPool() *readerPool {
	return &readerPool{
		pool: &sync.Pool{},
	}
}

type segmentReaderPools struct {
	mu    sync.Mutex
	pools map[string]*readerPool
}

func newSegmentReaderPools() *segmentReaderPools {
	return &segmentReaderPools{
		pools: make(map[string]*readerPool),
	}
}

func (p *segmentReaderPools) get(key string, openFn func() (*os.File, error)) (*os.File, func(), error) {
	p.mu.Lock()
	rp := p.pools[key]
	if rp == nil {
		rp = newReaderPool()
		p.pools[key] = rp
	}
	p.mu.Unlock()

	if v := rp.pool.Get(); v != nil {
		f := v.(*os.File)
		return f, func() { rp.pool.Put(f) }, nil
	}

	f, err := openFn()
	if err != nil {
		return nil, nil, err
	}
	rp.mu.Lock()
	rp.files = append(rp.files, f)
	rp.mu.Unlock()
	return f, func() { rp.pool.Put(f) }, nil
}

func (p *segmentReaderPools) closeAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, rp := range p.pools {
		rp.mu.Lock()
		for _, f := range rp.files {
			_ = f.Close()
		}
		rp.files = nil
		rp.mu.Unlock()
	}
	p.pools = make(map[string]*readerPool)
}
