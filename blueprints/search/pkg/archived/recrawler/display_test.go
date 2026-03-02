package recrawler

import "testing"

func TestRecordDomainSkipReason_DNSTimeoutIsNotTimeoutKill(t *testing.T) {
	s := NewStats(10, 1, "test")

	s.RecordDomainSkipReason("dns_timeout")

	if got := s.timeoutSkip.Load(); got != 0 {
		t.Fatalf("timeout-kill-skip = %d, want 0 for dns_timeout", got)
	}
	if got := s.domainSkip.Load(); got != 1 {
		t.Fatalf("dead/domain skip = %d, want 1 for dns_timeout", got)
	}
}

func TestRecordDomainSkipBatchReason_HTTPTimeoutKilledCountsSeparately(t *testing.T) {
	s := NewStats(10, 1, "test")

	s.RecordDomainSkipBatchReason("http_timeout_killed", 7)
	s.RecordDomainSkipBatchReason("domain_http_timeout_killed", 3) // legacy/reclassified label

	if got := s.timeoutSkip.Load(); got != 10 {
		t.Fatalf("timeout-kill-skip = %d, want 10", got)
	}
	if got := s.domainSkip.Load(); got != 0 {
		t.Fatalf("dead/domain skip = %d, want 0", got)
	}
}
