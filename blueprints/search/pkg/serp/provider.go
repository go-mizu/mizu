package serp

// Provider is a SERP API backend that can execute searches.
type Provider interface {
	Search(apiKey, query string) (*SearchResult, error)
	Name() string
}

var providers = map[string]Provider{
	"serper":    &SerperProvider{},
	"zenserp":   &ZenserpProvider{},
	"searchapi": &SearchAPIProvider{},
	"serpstack": &SerpStackProvider{},
	"serply":    &SerplyProvider{},
}

// AddProvider registers a provider (called from sub-package init()).
func AddProvider(name string, p Provider) {
	providers[name] = p
}

// AllProviders returns provider instances keyed by name.
func AllProviders() map[string]Provider {
	out := make(map[string]Provider, len(providers))
	for k, v := range providers {
		out[k] = v
	}
	return out
}
