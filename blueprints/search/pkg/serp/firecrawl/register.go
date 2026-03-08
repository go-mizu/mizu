package firecrawl

import (
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/serp"
)

func init() {
	serp.RegisterRegistrar("firecrawl", &registrar{})
}

type registrar struct{}

// Register on firecrawl.dev — has reCAPTCHA Enterprise, manual signup required.
func (r *registrar) Register(email, password string, verbose bool) (string, error) {
	return "", fmt.Errorf("firecrawl signup requires reCAPTCHA — use 'serp add-key firecrawl <key>' after manual signup at https://www.firecrawl.dev/signin?view=signup")
}

func (r *registrar) VerifyAndGetKey(email, password, emailBody string, verbose bool) (string, error) {
	return "", fmt.Errorf("firecrawl: manual signup required")
}
