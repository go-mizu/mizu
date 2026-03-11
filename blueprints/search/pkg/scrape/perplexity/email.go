package perplexity

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

// TempEmailClient provides disposable email addresses for registration.
type TempEmailClient interface {
	Email() string
	WaitForMessage(ctx context.Context, matchSubject string, timeout time.Duration) (string, error)
}

// ProviderTier determines the order in which providers are tried.
// Lower values are preferred (private > session > public).
type ProviderTier int

const (
	TierPrivate ProviderTier = 1 // Private inbox, auth required (mail.tm, mail.gw, tempmail.lol)
	TierSession ProviderTier = 2 // Session-based inbox (guerrillamail, dropmail)
	TierPublic  ProviderTier = 3 // Public inbox, no auth (tempmail.plus, inboxkitten)
)

// EmailProvider describes a temp email provider.
type EmailProvider struct {
	Name    string
	Tier    ProviderTier
	NewFunc func(ctx context.Context) (TempEmailClient, error)
}

// allProviders is the registry of all available temp email providers.
var allProviders = []EmailProvider{
	// Tier 1: Private (JWT/token auth, private inbox)
	{Name: "mail.tm", Tier: TierPrivate, NewFunc: func(ctx context.Context) (TempEmailClient, error) { return NewMailTMClient(ctx) }},
	{Name: "mail.gw", Tier: TierPrivate, NewFunc: func(ctx context.Context) (TempEmailClient, error) { return NewMailGWClient(ctx) }},
	{Name: "tempmail.lol", Tier: TierPrivate, NewFunc: func(ctx context.Context) (TempEmailClient, error) { return NewTempMailLolClient(ctx) }},

	// Tier 2: Session (session-based, semi-private)
	{Name: "guerrillamail", Tier: TierSession, NewFunc: func(ctx context.Context) (TempEmailClient, error) { return NewGuerrillaClient(ctx) }},
	{Name: "dropmail", Tier: TierSession, NewFunc: func(ctx context.Context) (TempEmailClient, error) { return NewDropMailClient(ctx) }},

	// Tier 3: Public (no auth, anyone can read inbox)
	{Name: "tempmail.plus", Tier: TierPublic, NewFunc: func(ctx context.Context) (TempEmailClient, error) { return NewTempMailPlusClient(ctx) }},
	{Name: "inboxkitten", Tier: TierPublic, NewFunc: func(ctx context.Context) (TempEmailClient, error) { return NewInboxKittenClient(ctx) }},
}

// NewTempEmailClient creates a disposable email client.
// Shuffles providers within each tier and tries them in tier order (private > session > public).
// Returns the first successful provider.
func NewTempEmailClient(ctx context.Context) (TempEmailClient, error) {
	ordered := shuffledByTier(allProviders)

	var errs []string
	for _, p := range ordered {
		client, err := p.NewFunc(ctx)
		if err == nil {
			fmt.Printf("Using email provider: %s (%s)\n", p.Name, client.Email())
			return client, nil
		}
		fmt.Printf("%s failed: %v\n", p.Name, err)
		errs = append(errs, fmt.Sprintf("%s: %v", p.Name, err))
	}

	return nil, fmt.Errorf("all %d email providers failed:\n  %s", len(ordered), strings.Join(errs, "\n  "))
}

// shuffledByTier groups providers by tier, shuffles within each tier, then concatenates.
func shuffledByTier(providers []EmailProvider) []EmailProvider {
	tiers := map[ProviderTier][]EmailProvider{}
	for _, p := range providers {
		tiers[p.Tier] = append(tiers[p.Tier], p)
	}

	var result []EmailProvider
	for _, tier := range []ProviderTier{TierPrivate, TierSession, TierPublic} {
		group := tiers[tier]
		rand.Shuffle(len(group), func(i, j int) { group[i], group[j] = group[j], group[i] })
		result = append(result, group...)
	}
	return result
}

// randomString generates a random lowercase alphanumeric string of given length.
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.IntN(len(letters))]
	}
	return string(b)
}
