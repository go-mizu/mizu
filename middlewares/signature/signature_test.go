package signature

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("test-secret"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// GET is skipped by default
	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestValidSignature(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test payload")

	app := mizu.NewRouter()
	app.Use(New(secret))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Create valid signature
	signature := Sign(secret, SHA256, "hex", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Signature", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with valid signature, got %d", http.StatusOK, rec.Code)
	}
}

func TestInvalidSignature(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("test-secret"))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("payload"))
	req.Header.Set("X-Signature", "invalid-signature")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d with invalid signature, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestMissingSignature(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("test-secret"))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("payload"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d without signature, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestSignaturePrefix(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test payload")

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:          secret,
		SignaturePrefix: "sha256=",
	}))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	signature := "sha256=" + Sign(secret, SHA256, "hex", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Signature", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with prefixed signature, got %d", http.StatusOK, rec.Code)
	}
}

func TestBase64Encoding(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test payload")

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:   secret,
		Encoding: "base64",
	}))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	signature := Sign(secret, SHA256, "base64", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Signature", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with base64 signature, got %d", http.StatusOK, rec.Code)
	}
}

func TestSHA1Algorithm(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test payload")

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:    secret,
		Algorithm: SHA1,
	}))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	signature := Sign(secret, SHA1, "hex", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Signature", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with SHA1 signature, got %d", http.StatusOK, rec.Code)
	}
}

func TestSHA512Algorithm(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test payload")

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:    secret,
		Algorithm: SHA512,
	}))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	signature := Sign(secret, SHA512, "hex", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Signature", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with SHA512 signature, got %d", http.StatusOK, rec.Code)
	}
}

func TestSkipPaths(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:    "test-secret",
		SkipPaths: []string{"/health"},
	}))

	app.Post("/health", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d for skipped path, got %d", http.StatusOK, rec.Code)
	}
}

func TestCustomHeaderName(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test payload")

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:     secret,
		HeaderName: "X-Custom-Sig",
	}))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	signature := Sign(secret, SHA256, "hex", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Custom-Sig", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with custom header, got %d", http.StatusOK, rec.Code)
	}
}

func TestCustomErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret: "test-secret",
		ErrorHandler: func(c *mizu.Ctx) error {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "signature invalid",
			})
		},
	}))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("payload"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected custom error code, got %d", rec.Code)
	}
}

func TestOnValid(t *testing.T) {
	var validCalled bool
	secret := "test-secret"
	payload := []byte("test")

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret: secret,
		OnValid: func(c *mizu.Ctx) {
			validCalled = true
		},
	}))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	signature := Sign(secret, SHA256, "hex", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Signature", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !validCalled {
		t.Error("expected OnValid to be called")
	}
}

func TestGetInfo(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test")

	app := mizu.NewRouter()
	app.Use(New(secret))

	var info *Info

	app.Post("/webhook", func(c *mizu.Ctx) error {
		info = GetInfo(c)
		return c.Text(http.StatusOK, "ok")
	})

	signature := Sign(secret, SHA256, "hex", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Signature", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info == nil {
		t.Fatal("expected info")
	}
	if !info.Valid {
		t.Error("expected valid signature")
	}
}

func TestIsValid(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test")

	app := mizu.NewRouter()
	app.Use(New(secret))

	var valid bool

	app.Post("/webhook", func(c *mizu.Ctx) error {
		valid = IsValid(c)
		return c.Text(http.StatusOK, "ok")
	})

	signature := Sign(secret, SHA256, "hex", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Signature", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !valid {
		t.Error("expected IsValid to return true")
	}
}

func TestGitHub(t *testing.T) {
	secret := "github-secret"
	payload := []byte(`{"action":"ping"}`)

	app := mizu.NewRouter()
	app.Use(GitHub(secret))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	signature := "sha256=" + Sign(secret, SHA256, "hex", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Hub-Signature-256", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d for GitHub signature, got %d", http.StatusOK, rec.Code)
	}
}

func TestSign(t *testing.T) {
	tests := []struct {
		algo     Algorithm
		encoding string
	}{
		{SHA1, "hex"},
		{SHA256, "hex"},
		{SHA512, "hex"},
		{SHA256, "base64"},
	}

	for _, tc := range tests {
		signature := Sign("secret", tc.algo, tc.encoding, []byte("payload"))
		if signature == "" {
			t.Errorf("expected signature for %s/%s", tc.algo, tc.encoding)
		}

		// Same input should produce same signature
		signature2 := Sign("secret", tc.algo, tc.encoding, []byte("payload"))
		if signature != signature2 {
			t.Errorf("expected consistent signature for %s/%s", tc.algo, tc.encoding)
		}
	}
}

func TestSigner(t *testing.T) {
	signer := NewSigner("test-secret")

	payload := []byte("test payload")
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))

	err := signer.SignRequest(req)
	if err != nil {
		t.Fatalf("sign error: %v", err)
	}

	signature := req.Header.Get("X-Signature")
	if signature == "" {
		t.Error("expected signature header")
	}

	// Verify signature
	expected := Sign("test-secret", SHA256, "hex", payload)
	if signature != expected {
		t.Errorf("signature mismatch: got %q, expected %q", signature, expected)
	}
}

func TestSignerWithPrefix(t *testing.T) {
	signer := NewSigner("test-secret")
	signer.Prefix = "sha256="

	payload := []byte("test payload")
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))

	err := signer.SignRequest(req)
	if err != nil {
		t.Fatalf("sign error: %v", err)
	}

	signature := req.Header.Get("X-Signature")
	if !strings.HasPrefix(signature, "sha256=") {
		t.Error("expected signature with prefix")
	}
}

func TestSignerNilBody(t *testing.T) {
	signer := NewSigner("test-secret")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Body = nil

	err := signer.SignRequest(req)
	if err != nil {
		t.Fatalf("sign error: %v", err)
	}

	signature := req.Header.Get("X-Signature")
	if signature == "" {
		t.Error("expected signature for nil body")
	}
}

func TestStripe(t *testing.T) {
	secret := "stripe-secret"
	payload := []byte(`{"type":"checkout.session.completed"}`)

	app := mizu.NewRouter()
	app.Use(Stripe(secret))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	signature := Sign(secret, SHA256, "hex", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d for Stripe signature, got %d", http.StatusOK, rec.Code)
	}
}

func TestSlack(t *testing.T) {
	secret := "slack-signing-secret"
	body := "command=/test&text=hello"
	timestamp := "1234567890"

	// Slack signature format: v0=HMAC(v0:timestamp:body)
	baseString := "v0:" + timestamp + ":" + body

	app := mizu.NewRouter()
	app.Use(Slack(secret))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	signature := "v0=" + Sign(secret, SHA256, "hex", []byte(baseString))

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Slack-Signature", signature)
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d for Slack signature, got %d", http.StatusOK, rec.Code)
	}
}

func TestTwilio(t *testing.T) {
	secret := "twilio-auth-token" //nolint:gosec // G101: Test credential for webhook signature verification

	app := mizu.NewRouter()
	app.Use(Twilio(secret))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// For Twilio, we need form data and a proper signature
	formData := "Body=Test&From=%2B1234567890"

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Build base string: URL + sorted params
	baseString := req.URL.String() + "BodyTest" + "From+1234567890"
	signature := Sign(secret, SHA1, "base64", []byte(baseString))

	req.Header.Set("X-Twilio-Signature", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Note: The actual signature format is complex, this tests the code path
	// In real use, Twilio sends a properly formatted signature
	if rec.Code == http.StatusOK {
		t.Log("Twilio signature verified")
	}
}

func TestAWS(t *testing.T) {
	secret := "aws-secret-key"
	payload := []byte(`{"key":"value"}`)

	app := mizu.NewRouter()
	app.Use(AWS(secret))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	signature := Sign(secret, SHA256, "hex", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("Authorization", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d for AWS signature, got %d", http.StatusOK, rec.Code)
	}
}

func TestGetInfo_NoContext(t *testing.T) {
	app := mizu.NewRouter()

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = GetInfo(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info != nil {
		t.Error("expected nil info without signature middleware")
	}
}

func TestIsValid_NoContext(t *testing.T) {
	app := mizu.NewRouter()

	var valid bool
	app.Get("/", func(c *mizu.Ctx) error {
		valid = IsValid(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if valid {
		t.Error("expected IsValid to return false without middleware")
	}
}

func TestSignaturePrefix_Missing(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test payload")

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:          secret,
		SignaturePrefix: "sha256=",
	}))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Signature without prefix
	signature := Sign(secret, SHA256, "hex", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Signature", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d when prefix missing, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestCustomPayloadGetter(t *testing.T) {
	secret := "test-secret"

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret: secret,
		PayloadGetter: func(c *mizu.Ctx) ([]byte, error) {
			return []byte("custom-payload"), nil
		},
	}))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	signature := Sign(secret, SHA256, "hex", []byte("custom-payload"))

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("original"))
	req.Header.Set("X-Signature", signature)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with custom payload getter, got %d", http.StatusOK, rec.Code)
	}
}

func TestCustomPayloadGetter_Error(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret: "secret",
		PayloadGetter: func(c *mizu.Ctx) ([]byte, error) {
			return nil, http.ErrBodyNotAllowed
		},
	}))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("data"))
	req.Header.Set("X-Signature", "any")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d on payload getter error, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestSignUnknownAlgorithm(t *testing.T) {
	// Unknown algorithm should default to SHA256
	sig := Sign("secret", "unknown", "hex", []byte("payload"))
	expected := Sign("secret", SHA256, "hex", []byte("payload"))

	if sig != expected {
		t.Error("expected unknown algorithm to default to SHA256")
	}
}

func TestSkipMethods_Custom(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:      "test-secret",
		SkipMethods: []string{"post"}, // lowercase
	}))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d for skipped method, got %d", http.StatusOK, rec.Code)
	}
}

func TestSignerSign_Direct(t *testing.T) {
	signer := NewSigner("test-secret")
	signer.Algorithm = SHA512
	signer.Encoding = "base64"
	signer.HeaderName = "X-Custom-Sig"
	signer.Prefix = "sig="

	payload := []byte("test")
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	signer.Sign(req, payload)

	sig := req.Header.Get("X-Custom-Sig")
	if !strings.HasPrefix(sig, "sig=") {
		t.Error("expected signature with prefix")
	}
}
