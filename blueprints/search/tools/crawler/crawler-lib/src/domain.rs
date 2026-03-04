use crate::types::SeedURL;

/// A batch of URLs belonging to a single domain.
#[derive(Debug)]
pub struct DomainBatch {
    pub domain: String,
    pub urls: Vec<SeedURL>,
}

/// Sort seeds by domain, yield contiguous domain batches.
/// Consumes the input vector (sorts in-place to avoid allocation).
pub fn group_by_domain(mut seeds: Vec<SeedURL>) -> Vec<DomainBatch> {
    seeds.sort_unstable_by(|a, b| a.domain.cmp(&b.domain));

    let mut batches = Vec::new();
    let mut current_domain = String::new();
    let mut current_urls: Vec<SeedURL> = Vec::new();

    for seed in seeds {
        if seed.domain != current_domain {
            if !current_urls.is_empty() {
                batches.push(DomainBatch {
                    domain: std::mem::take(&mut current_domain),
                    urls: std::mem::take(&mut current_urls),
                });
            }
            current_domain.clone_from(&seed.domain);
        }
        current_urls.push(seed);
    }
    if !current_urls.is_empty() {
        batches.push(DomainBatch {
            domain: current_domain,
            urls: current_urls,
        });
    }
    batches
}

/// Interleave URLs from multiple domain batches in round-robin order.
///
/// Example: batches A=[1,2], B=[1] → [A1, B1, A2]
/// This is O(total_urls) — no wasted iterations for exhausted domains.
pub fn interleave_by_domain(batches: Vec<DomainBatch>) -> Vec<SeedURL> {
    use std::collections::VecDeque;
    let mut queue: VecDeque<VecDeque<SeedURL>> = batches
        .into_iter()
        .map(|b| VecDeque::from(b.urls))
        .collect();
    let mut result = Vec::new();
    while let Some(mut urls) = queue.pop_front() {
        if let Some(url) = urls.pop_front() {
            result.push(url);
            if !urls.is_empty() {
                queue.push_back(urls);
            }
        }
    }
    result
}

/// Per-domain state tracking during crawl.
/// Used by engine workers to decide when to abandon a domain.
pub struct DomainState {
    pub successes: u64,
    pub timeouts: u64,
}

impl DomainState {
    pub fn new() -> Self {
        Self { successes: 0, timeouts: 0 }
    }

    /// Check if domain should be abandoned based on config rules.
    /// Matches Go's keepalive.go abandonment logic:
    /// - domain_fail_threshold: N rounds of all-timeout (threshold * inner_n)
    /// - domain_dead_probe: N timeouts with 0 success -> dead
    /// - domain_stall_ratio: timeouts >= successes * ratio -> stalling
    pub fn should_abandon(
        &self,
        domain_fail_threshold: usize,
        domain_dead_probe: usize,
        domain_stall_ratio: usize,
        inner_n: usize,
    ) -> bool {
        // DomainFailThreshold: N rounds of all-timeout
        if domain_fail_threshold > 0 {
            let effective = (domain_fail_threshold * inner_n.max(1)) as u64;
            if self.timeouts >= effective {
                return true;
            }
        }
        // DomainDeadProbe: N timeouts with 0 success -> dead domain
        if domain_dead_probe > 0 && self.timeouts >= domain_dead_probe as u64 {
            if self.successes == 0 {
                return true; // dead
            }
            // Stall ratio: timeouts >= successes * ratio
            if domain_stall_ratio > 0
                && self.successes > 0
                && self.timeouts >= self.successes * domain_stall_ratio as u64
            {
                return true; // stalling
            }
        }
        false
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_group_by_domain() {
        let seeds = vec![
            SeedURL { url: "https://b.com/1".into(), domain: "b.com".into() },
            SeedURL { url: "https://a.com/1".into(), domain: "a.com".into() },
            SeedURL { url: "https://a.com/2".into(), domain: "a.com".into() },
            SeedURL { url: "https://c.com/1".into(), domain: "c.com".into() },
            SeedURL { url: "https://b.com/2".into(), domain: "b.com".into() },
        ];
        let batches = group_by_domain(seeds);
        assert_eq!(batches.len(), 3);
        assert_eq!(batches[0].domain, "a.com");
        assert_eq!(batches[0].urls.len(), 2);
        assert_eq!(batches[1].domain, "b.com");
        assert_eq!(batches[1].urls.len(), 2);
        assert_eq!(batches[2].domain, "c.com");
        assert_eq!(batches[2].urls.len(), 1);
    }

    #[test]
    fn test_domain_state_dead_probe() {
        let mut state = DomainState::new();
        state.timeouts = 2;
        assert!(state.should_abandon(0, 2, 0, 4)); // dead: 0 successes + 2 timeouts >= probe(2)
    }

    #[test]
    fn test_domain_state_stall_ratio() {
        let mut state = DomainState::new();
        state.successes = 1;
        state.timeouts = 20;
        assert!(state.should_abandon(0, 2, 20, 4)); // stalling: 20 >= 1*20
    }

    #[test]
    fn test_domain_state_fail_threshold() {
        let mut state = DomainState::new();
        state.timeouts = 12;
        // threshold=3, inner_n=4, effective=12
        assert!(state.should_abandon(3, 0, 0, 4));
    }

    #[test]
    fn test_domain_state_not_abandoned() {
        let mut state = DomainState::new();
        state.successes = 5;
        state.timeouts = 3;
        assert!(!state.should_abandon(3, 10, 20, 4));
    }

    #[test]
    fn test_zero_stall_ratio_never_abandons_alive_domain() {
        let mut state = DomainState::new();
        state.successes = 1;
        state.timeouts = 1_000_000; // extreme stall — should NOT abandon if ratio=0
        // stall_ratio=0 means disabled; only dead_probe can trigger abandonment here
        // dead_probe=0 → disabled; fail_threshold=0 → disabled
        assert!(
            !state.should_abandon(0, 0, 0, 4),
            "stall_ratio=0 must never abandon regardless of timeout count"
        );
    }

    #[test]
    fn test_zero_stall_ratio_with_dead_probe_still_abandons_dead_domain() {
        // When stall_ratio=0 but dead_probe is set, truly dead domains ARE abandoned.
        let mut state = DomainState::new();
        state.successes = 0;
        state.timeouts = 2;
        // dead_probe=2, successes=0 → should abandon (dead domain)
        assert!(
            state.should_abandon(0, 2, 0, 4),
            "dead_probe=2 with 0 successes should still abandon dead domains even with stall_ratio=0"
        );
    }

    #[test]
    fn test_interleave_sends_all_urls() {
        let seeds: Vec<SeedURL> = (0..100)
            .map(|i| SeedURL {
                url: format!("https://d{}.com/page{}", i % 5, i),
                domain: format!("d{}.com", i % 5),
            })
            .collect();
        let batches = group_by_domain(seeds);
        let result = interleave_by_domain(batches);
        assert_eq!(result.len(), 100, "all URLs must be delivered");
    }

    #[test]
    fn test_interleave_round_robin_order() {
        // a has 2 URLs, b has 1 URL → round-robin: a0, b0, a1
        let seeds = vec![
            SeedURL { url: "https://a.com/1".into(), domain: "a.com".into() },
            SeedURL { url: "https://a.com/2".into(), domain: "a.com".into() },
            SeedURL { url: "https://b.com/1".into(), domain: "b.com".into() },
        ];
        let batches = group_by_domain(seeds);
        let result = interleave_by_domain(batches);
        assert_eq!(result.len(), 3);
        // First URL from a, first from b, then second from a
        assert_eq!(result[0].domain, "a.com");
        assert_eq!(result[1].domain, "b.com");
        assert_eq!(result[2].domain, "a.com");
    }

    #[test]
    fn test_interleave_single_domain() {
        let seeds = vec![
            SeedURL { url: "https://only.com/1".into(), domain: "only.com".into() },
            SeedURL { url: "https://only.com/2".into(), domain: "only.com".into() },
        ];
        let batches = group_by_domain(seeds);
        let result = interleave_by_domain(batches);
        assert_eq!(result.len(), 2);
    }
}
