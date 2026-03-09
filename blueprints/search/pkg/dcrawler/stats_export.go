package dcrawler

// Exported accessors for Stats fields used by the dashboard scrape pipeline.

func (s *Stats) Success() int64    { return s.success.Load() }
func (s *Stats) Failed() int64     { return s.failed.Load() }
func (s *Stats) Timeout() int64    { return s.timeout.Load() }
func (s *Stats) Bytes() int64      { return s.bytes.Load() }
func (s *Stats) InFlight() int64   { return s.inFlight.Load() }
func (s *Stats) LinksFound() int64 { return s.linksFound.Load() }

func (s *Stats) FrontierLen() int {
	if s.frontierLen == nil {
		return 0
	}
	return s.frontierLen()
}
