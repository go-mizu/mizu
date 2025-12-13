// Package captcha provides CAPTCHA verification middleware for the Mizu web framework.
//
// This middleware verifies CAPTCHA tokens to protect forms and APIs from automated bots.
// It supports multiple providers including Google reCAPTCHA (v2/v3), hCaptcha, and
// Cloudflare Turnstile.
//
// # Supported Providers
//
// The middleware supports the following CAPTCHA providers:
//
//   - Google reCAPTCHA v2 (checkbox-based)
//   - Google reCAPTCHA v3 (invisible, score-based)
//   - hCaptcha (checkbox-based, privacy-focused)
//   - Cloudflare Turnstile (invisible)
//   - Custom verification functions
//
// # Basic Usage
//
// Using the default reCAPTCHA v2 provider:
//
//	app := mizu.New()
//	app.Use(captcha.ReCaptchaV2("your-secret-key"))
//
// Using reCAPTCHA v3 with a custom score threshold:
//
//	app.Use(captcha.ReCaptchaV3("your-secret-key", 0.7))
//
// Using hCaptcha:
//
//	app.Use(captcha.HCaptcha("your-secret-key"))
//
// Using Cloudflare Turnstile:
//
//	app.Use(captcha.Turnstile("your-secret-key"))
//
// # Advanced Configuration
//
// The middleware can be configured with various options using WithOptions:
//
//	app.Use(captcha.WithOptions(captcha.Options{
//	    Provider:    captcha.ProviderRecaptchaV3,
//	    Secret:      os.Getenv("RECAPTCHA_SECRET"),
//	    MinScore:    0.7,
//	    TokenLookup: "header:X-Captcha-Token",
//	    SkipPaths:   []string{"/api/webhook", "/api/health"},
//	    Timeout:     15 * time.Second,
//	    ErrorHandler: func(c *mizu.Ctx, err error) error {
//	        return c.JSON(400, map[string]string{"error": err.Error()})
//	    },
//	}))
//
// # Token Extraction
//
// The middleware supports extracting tokens from multiple sources:
//
//   - Form data: "form:field-name" (default: "form:g-recaptcha-response")
//   - HTTP headers: "header:Header-Name"
//   - Query parameters: "query:param-name"
//
// Example with custom token location:
//
//	app.Use(captcha.WithOptions(captcha.Options{
//	    Provider:    captcha.ProviderRecaptchaV3,
//	    Secret:      secret,
//	    TokenLookup: "header:X-Captcha-Token",
//	}))
//
// # Custom Verification
//
// You can provide a custom verification function:
//
//	app.Use(captcha.Custom(
//	    func(token string, c *mizu.Ctx) (bool, error) {
//	        // Custom verification logic
//	        return myVerifier.Verify(token, c.ClientIP())
//	    },
//	    "form:my-captcha-token",
//	))
//
// # Request Filtering
//
// The middleware automatically skips verification for:
//
//   - Safe HTTP methods (GET, HEAD, OPTIONS)
//   - Paths configured in SkipPaths option
//
// # Verification Flow
//
// For each request, the middleware:
//
//  1. Checks if the request method is safe (GET, HEAD, OPTIONS) - if so, skips verification
//  2. Checks if the request path is in SkipPaths - if so, skips verification
//  3. Extracts the CAPTCHA token from the configured location
//  4. Verifies the token using either a custom verifier or the provider's API
//  5. For v3 providers (reCAPTCHA v3, Turnstile), validates the score against MinScore
//  6. Allows the request to proceed if verification succeeds, otherwise returns an error
//
// # Client IP Detection
//
// The middleware detects the client IP address for enhanced verification:
//
//  1. Checks X-Forwarded-For header (uses first IP in comma-separated list)
//  2. Checks X-Real-IP header
//  3. Falls back to RemoteAddr from the request
//
// The detected IP is sent to the provider's verification API when available.
//
// # Error Handling
//
// The middleware defines three error types:
//
//   - ErrMissingToken: Token not found in the request
//   - ErrInvalidToken: Token verification failed or score too low
//   - ErrVerifyFailed: HTTP request to provider API failed
//
// You can provide a custom error handler to customize error responses:
//
//	app.Use(captcha.WithOptions(captcha.Options{
//	    Provider: captcha.ProviderRecaptchaV2,
//	    Secret:   secret,
//	    ErrorHandler: func(c *mizu.Ctx, err error) error {
//	        if err == captcha.ErrMissingToken {
//	            return c.JSON(400, map[string]string{
//	                "error": "Please complete the CAPTCHA",
//	            })
//	        }
//	        return c.JSON(403, map[string]string{
//	            "error": "CAPTCHA verification failed",
//	        })
//	    },
//	}))
//
// # Provider API Integration
//
// The middleware integrates with provider APIs at the following endpoints:
//
//   - reCAPTCHA v2/v3: https://www.google.com/recaptcha/api/siteverify
//   - hCaptcha: https://hcaptcha.com/siteverify
//   - Turnstile: https://challenges.cloudflare.com/turnstile/v0/siteverify
//
// Each request to the provider includes:
//
//   - secret: Your secret key
//   - response: The CAPTCHA token
//   - remoteip: The client's IP address (when available)
//
// # Security Considerations
//
// When using this middleware:
//
//   - Never expose your secret key in frontend code or version control
//   - Set appropriate score thresholds for v3 providers based on your traffic patterns
//   - Configure reasonable timeouts to prevent slow verification requests
//   - Monitor verification failures to tune settings
//   - Consider having a manual review process for edge cases
//
// # Route-Specific Protection
//
// You can apply CAPTCHA verification to specific routes instead of globally:
//
//	registerCaptcha := captcha.ReCaptchaV2(secret)
//	loginCaptcha := captcha.ReCaptchaV3(secret, 0.5)
//
//	app.Post("/register", registerHandler, registerCaptcha)
//	app.Post("/login", loginHandler, loginCaptcha)
package captcha
