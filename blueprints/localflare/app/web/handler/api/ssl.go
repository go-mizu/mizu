package api

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// SSL handles SSL/TLS requests.
type SSL struct {
	store     store.SSLStore
	zoneStore store.ZoneStore
}

// NewSSL creates a new SSL handler.
func NewSSL(store store.SSLStore, zoneStore store.ZoneStore) *SSL {
	return &SSL{store: store, zoneStore: zoneStore}
}

// GetSettings retrieves SSL settings for a zone.
func (h *SSL) GetSettings(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	settings, err := h.store.GetSettings(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  settings,
	})
}

// UpdateSSLSettingsInput is the input for updating SSL settings.
type UpdateSSLSettingsInput struct {
	Mode                    string `json:"mode"`
	AlwaysHTTPS             bool   `json:"always_https"`
	MinTLSVersion           string `json:"min_tls_version"`
	OpportunisticEncryption bool   `json:"opportunistic_encryption"`
	TLS13                   bool   `json:"tls_1_3"`
	AutomaticHTTPSRewrites  bool   `json:"automatic_https_rewrites"`
}

// UpdateSettings updates SSL settings for a zone.
func (h *SSL) UpdateSettings(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	var input UpdateSSLSettingsInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	settings := &store.SSLSettings{
		ZoneID:                  zoneID,
		Mode:                    input.Mode,
		AlwaysHTTPS:             input.AlwaysHTTPS,
		MinTLSVersion:           input.MinTLSVersion,
		OpportunisticEncryption: input.OpportunisticEncryption,
		TLS13:                   input.TLS13,
		AutomaticHTTPSRewrites:  input.AutomaticHTTPSRewrites,
	}

	if err := h.store.UpdateSettings(c.Request().Context(), settings); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  settings,
	})
}

// ListCertificates lists all certificates for a zone.
func (h *SSL) ListCertificates(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	certs, err := h.store.ListCertificates(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  certs,
	})
}

// CreateCertificateInput is the input for creating a certificate.
type CreateCertificateInput struct {
	Type  string   `json:"type"`
	Hosts []string `json:"hosts"`
}

// CreateCertificate creates a new certificate.
func (h *SSL) CreateCertificate(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	zone, err := h.zoneStore.GetByID(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Zone not found"})
	}

	var input CreateCertificateInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if len(input.Hosts) == 0 {
		input.Hosts = []string{zone.Name, "*." + zone.Name}
	}

	if input.Type == "" {
		input.Type = "edge"
	}

	// Generate self-signed certificate
	certPEM, keyPEM, err := generateSelfSignedCert(input.Hosts)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "Failed to generate certificate"})
	}

	now := time.Now()
	cert := &store.Certificate{
		ID:          ulid.Make().String(),
		ZoneID:      zoneID,
		Type:        input.Type,
		Hosts:       input.Hosts,
		Issuer:      "Localflare CA",
		SerialNum:   ulid.Make().String(),
		Signature:   "SHA256-RSA",
		Status:      "active",
		ExpiresAt:   now.AddDate(1, 0, 0), // 1 year
		Certificate: certPEM,
		PrivateKey:  keyPEM,
		CreatedAt:   now,
	}

	if err := h.store.CreateCertificate(c.Request().Context(), cert); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Don't return private key in response
	cert.PrivateKey = "[REDACTED]"

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  cert,
	})
}

// DeleteCertificate deletes a certificate.
func (h *SSL) DeleteCertificate(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DeleteCertificate(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// CreateOriginCAInput is the input for creating an Origin CA certificate.
type CreateOriginCAInput struct {
	Hosts    []string `json:"hosts"`
	Validity int      `json:"validity"` // days
}

// CreateOriginCA creates an Origin CA certificate.
func (h *SSL) CreateOriginCA(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	zone, err := h.zoneStore.GetByID(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Zone not found"})
	}

	var input CreateOriginCAInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if len(input.Hosts) == 0 {
		input.Hosts = []string{zone.Name, "*." + zone.Name}
	}

	if input.Validity == 0 {
		input.Validity = 5475 // 15 years (Cloudflare default)
	}

	// Generate Origin CA certificate (longer validity)
	certPEM, keyPEM, err := generateOriginCACert(input.Hosts, input.Validity)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "Failed to generate certificate"})
	}

	now := time.Now()
	cert := &store.Certificate{
		ID:          ulid.Make().String(),
		ZoneID:      zoneID,
		Type:        "origin",
		Hosts:       input.Hosts,
		Issuer:      "Localflare Origin CA",
		SerialNum:   ulid.Make().String(),
		Signature:   "SHA256-RSA",
		Status:      "active",
		ExpiresAt:   now.AddDate(0, 0, input.Validity),
		Certificate: certPEM,
		PrivateKey:  keyPEM,
		CreatedAt:   now,
	}

	if err := h.store.CreateCertificate(c.Request().Context(), cert); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"id":          cert.ID,
			"certificate": cert.Certificate,
			"private_key": cert.PrivateKey,
			"expires_at":  cert.ExpiresAt,
		},
	})
}

func generateSelfSignedCert(hosts []string) (string, string, error) {
	return generateCert(hosts, 365)
}

func generateOriginCACert(hosts []string, validityDays int) (string, string, error) {
	return generateCert(hosts, validityDays)
}

func generateCert(hosts []string, validityDays int) (string, string, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", err
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Localflare"},
			CommonName:   hosts[0],
		},
		NotBefore:             now,
		NotAfter:              now.AddDate(0, 0, validityDays),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              hosts,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", "", err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return string(certPEM), string(keyPEM), nil
}
