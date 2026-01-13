package api

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// DNS handles DNS record requests.
type DNS struct {
	store     store.DNSStore
	zoneStore store.ZoneStore
}

// NewDNS creates a new DNS handler.
func NewDNS(store store.DNSStore, zoneStore store.ZoneStore) *DNS {
	return &DNS{store: store, zoneStore: zoneStore}
}

// List lists all DNS records for a zone.
func (h *DNS) List(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	records, err := h.store.ListByZone(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  records,
	})
}

// Get retrieves a DNS record by ID.
func (h *DNS) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	record, err := h.store.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Record not found"})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  record,
	})
}

// CreateDNSInput is the input for creating a DNS record.
type CreateDNSInput struct {
	Type     string   `json:"type"`
	Name     string   `json:"name"`
	Content  string   `json:"content"`
	TTL      int      `json:"ttl"`
	Priority int      `json:"priority"`
	Proxied  bool     `json:"proxied"`
	Comment  string   `json:"comment"`
	Tags     []string `json:"tags"`
}

// Create creates a new DNS record.
func (h *DNS) Create(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	var input CreateDNSInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Type == "" || input.Name == "" || input.Content == "" {
		return c.JSON(400, map[string]string{"error": "Type, name, and content are required"})
	}

	now := time.Now()
	record := &store.DNSRecord{
		ID:        ulid.Make().String(),
		ZoneID:    zoneID,
		Type:      input.Type,
		Name:      input.Name,
		Content:   input.Content,
		TTL:       input.TTL,
		Priority:  input.Priority,
		Proxied:   input.Proxied,
		Comment:   input.Comment,
		Tags:      input.Tags,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if record.TTL == 0 {
		record.TTL = 300
	}

	if err := h.store.Create(c.Request().Context(), record); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  record,
	})
}

// Update updates a DNS record.
func (h *DNS) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	record, err := h.store.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Record not found"})
	}

	var input CreateDNSInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Type != "" {
		record.Type = input.Type
	}
	if input.Name != "" {
		record.Name = input.Name
	}
	if input.Content != "" {
		record.Content = input.Content
	}
	if input.TTL != 0 {
		record.TTL = input.TTL
	}
	record.Priority = input.Priority
	record.Proxied = input.Proxied
	record.Comment = input.Comment
	if input.Tags != nil {
		record.Tags = input.Tags
	}
	record.UpdatedAt = time.Now()

	if err := h.store.Update(c.Request().Context(), record); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  record,
	})
}

// Delete deletes a DNS record.
func (h *DNS) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// Import imports DNS records from BIND format.
func (h *DNS) Import(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	zone, err := h.zoneStore.GetByID(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Zone not found"})
	}

	body, _ := io.ReadAll(c.Request().Body)
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	var imported int

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}

		name := parts[0]
		ttl := 300
		recordType := ""
		content := ""

		// Parse TTL if present
		idx := 1
		if _, err := fmt.Sscanf(parts[1], "%d", &ttl); err == nil {
			idx = 2
		}

		// Skip class (IN)
		if idx < len(parts) && parts[idx] == "IN" {
			idx++
		}

		if idx < len(parts) {
			recordType = parts[idx]
			idx++
		}

		if idx < len(parts) {
			content = strings.Join(parts[idx:], " ")
		}

		if recordType == "" || content == "" {
			continue
		}

		// Normalize name
		if name == "@" || name == zone.Name+"." {
			name = "@"
		} else {
			name = strings.TrimSuffix(name, "."+zone.Name+".")
			name = strings.TrimSuffix(name, ".")
		}

		now := time.Now()
		record := &store.DNSRecord{
			ID:        ulid.Make().String(),
			ZoneID:    zoneID,
			Type:      recordType,
			Name:      name,
			Content:   content,
			TTL:       ttl,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := h.store.Create(c.Request().Context(), record); err == nil {
			imported++
		}
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"records_imported": imported,
		},
	})
}

// Export exports DNS records in BIND format.
func (h *DNS) Export(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	zone, err := h.zoneStore.GetByID(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Zone not found"})
	}

	records, err := h.store.ListByZone(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("; Zone file for %s\n", zone.Name))
	builder.WriteString(fmt.Sprintf("; Exported at %s\n\n", time.Now().Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("$ORIGIN %s.\n", zone.Name))
	builder.WriteString("$TTL 300\n\n")

	for _, record := range records {
		name := record.Name
		if name == "@" {
			name = zone.Name + "."
		} else {
			name = name + "." + zone.Name + "."
		}

		if record.Type == "MX" && record.Priority > 0 {
			builder.WriteString(fmt.Sprintf("%-30s %d IN %-6s %d %s\n",
				name, record.TTL, record.Type, record.Priority, record.Content))
		} else {
			builder.WriteString(fmt.Sprintf("%-30s %d IN %-6s %s\n",
				name, record.TTL, record.Type, record.Content))
		}
	}

	c.Writer().Header().Set("Content-Type", "text/plain")
	c.Writer().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zone", zone.Name))
	c.Writer().Write([]byte(builder.String()))
	return nil
}
