package api

import (
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/table/feature/records"
)

// Record handles record endpoints.
type Record struct {
	records   *records.Service
	getUserID func(*mizu.Ctx) string
}

// NewRecord creates a new record handler.
func NewRecord(records *records.Service, getUserID func(*mizu.Ctx) string) *Record {
	return &Record{records: records, getUserID: getUserID}
}

// CreateRecordRequest is the request body for creating a record.
type CreateRecordRequest struct {
	TableID string                 `json:"table_id"`
	Fields  map[string]interface{} `json:"fields"`
}

// List returns records for a table.
func (h *Record) List(c *mizu.Ctx) error {
	tableID := c.Query("table_id")
	if tableID == "" {
		return BadRequest(c, "table_id is required")
	}

	opts := records.ListOpts{
		Limit:  100,
		Offset: 0,
	}

	if limit := c.Query("limit"); limit != "" {
		if n, err := strconv.Atoi(limit); err == nil && n > 0 {
			opts.Limit = n
		}
	}

	if cursor := c.Query("cursor"); cursor != "" {
		if offset, err := strconv.Atoi(cursor); err == nil {
			opts.Offset = offset
		}
	}

	list, err := h.records.List(c.Context(), tableID, opts)
	if err != nil {
		return InternalError(c, "failed to list records")
	}

	// Convert to frontend format
	result := make([]map[string]any, 0, len(list.Records))
	for _, rec := range list.Records {
		result = append(result, map[string]any{
			"id":         rec.ID,
			"table_id":   rec.TableID,
			"position":   rec.Position,
			"values":     rec.Cells,
			"createdBy":  rec.CreatedBy,
			"createdAt":  rec.CreatedAt,
			"updatedBy":  rec.UpdatedBy,
			"updatedAt":  rec.UpdatedAt,
		})
	}

	hasMore := opts.Offset+len(list.Records) < list.Total
	var nextCursor *string
	if hasMore {
		cursor := strconv.Itoa(opts.Offset + len(list.Records))
		nextCursor = &cursor
	}

	return OK(c, map[string]any{
		"records":     result,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

// Create creates a new record.
func (h *Record) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var req CreateRecordRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	record, err := h.records.Create(c.Context(), req.TableID, req.Fields, userID)
	if err != nil {
		return InternalError(c, "failed to create record")
	}

	// Convert to frontend format
	result := map[string]any{
		"id":         record.ID,
		"table_id":   record.TableID,
		"position":   record.Position,
		"values":     record.Cells,
		"createdBy":  record.CreatedBy,
		"createdAt":  record.CreatedAt,
		"updatedBy":  record.UpdatedBy,
		"updatedAt":  record.UpdatedAt,
	}

	return Created(c, map[string]any{"record": result})
}

// Get returns a record by ID.
func (h *Record) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	record, err := h.records.GetByID(c.Context(), id)
	if err != nil {
		return NotFound(c, "record not found")
	}

	result := map[string]any{
		"id":         record.ID,
		"table_id":   record.TableID,
		"position":   record.Position,
		"values":     record.Cells,
		"createdBy":  record.CreatedBy,
		"createdAt":  record.CreatedAt,
		"updatedBy":  record.UpdatedBy,
		"updatedAt":  record.UpdatedAt,
	}

	return OK(c, map[string]any{"record": result})
}

// UpdateRecordRequest is the request body for updating a record.
type UpdateRecordRequest struct {
	Fields map[string]interface{} `json:"fields"`
}

// Update updates a record.
func (h *Record) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	var req UpdateRecordRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	record, err := h.records.Update(c.Context(), id, req.Fields, userID)
	if err != nil {
		if err == records.ErrNotFound {
			return NotFound(c, "record not found")
		}
		return InternalError(c, "failed to update record")
	}

	result := map[string]any{
		"id":         record.ID,
		"table_id":   record.TableID,
		"position":   record.Position,
		"values":     record.Cells,
		"createdBy":  record.CreatedBy,
		"createdAt":  record.CreatedAt,
		"updatedBy":  record.UpdatedBy,
		"updatedAt":  record.UpdatedAt,
	}

	return OK(c, map[string]any{"record": result})
}

// Delete deletes a record.
func (h *Record) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.records.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete record")
	}

	return NoContent(c)
}

// BatchCreateRequest is the request body for batch creating records.
type BatchCreateRequest struct {
	TableID string                   `json:"table_id"`
	Records []map[string]interface{} `json:"records"`
}

// BatchCreate creates multiple records.
func (h *Record) BatchCreate(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var req BatchCreateRequest
	if err := c.BindJSON(&req, 10<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	// Convert records format
	recordsData := make([]map[string]interface{}, 0, len(req.Records))
	for _, r := range req.Records {
		if fields, ok := r["fields"].(map[string]interface{}); ok {
			recordsData = append(recordsData, fields)
		} else {
			recordsData = append(recordsData, r)
		}
	}

	result, err := h.records.CreateBatch(c.Context(), req.TableID, recordsData, userID)
	if err != nil {
		return InternalError(c, "failed to create records")
	}

	// Convert to frontend format
	records := make([]map[string]any, 0, len(result))
	for _, rec := range result {
		records = append(records, map[string]any{
			"id":         rec.ID,
			"table_id":   rec.TableID,
			"position":   rec.Position,
			"values":     rec.Cells,
			"createdBy":  rec.CreatedBy,
			"createdAt":  rec.CreatedAt,
			"updatedBy":  rec.UpdatedBy,
			"updatedAt":  rec.UpdatedAt,
		})
	}

	return Created(c, map[string]any{"records": records})
}

// BatchUpdateRequest is the request body for batch updating records.
type BatchUpdateRequest struct {
	Records []struct {
		ID     string                 `json:"id"`
		Fields map[string]interface{} `json:"fields"`
	} `json:"records"`
}

// BatchUpdate updates multiple records.
func (h *Record) BatchUpdate(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var req BatchUpdateRequest
	if err := c.BindJSON(&req, 10<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	updates := make([]records.RecordUpdate, 0, len(req.Records))
	for _, r := range req.Records {
		updates = append(updates, records.RecordUpdate{
			ID:    r.ID,
			Cells: r.Fields,
		})
	}

	result, err := h.records.UpdateBatch(c.Context(), updates, userID)
	if err != nil {
		return InternalError(c, "failed to update records")
	}

	// Convert to frontend format
	recordsList := make([]map[string]any, 0, len(result))
	for _, rec := range result {
		recordsList = append(recordsList, map[string]any{
			"id":         rec.ID,
			"table_id":   rec.TableID,
			"position":   rec.Position,
			"values":     rec.Cells,
			"createdBy":  rec.CreatedBy,
			"createdAt":  rec.CreatedAt,
			"updatedBy":  rec.UpdatedBy,
			"updatedAt":  rec.UpdatedAt,
		})
	}

	return OK(c, map[string]any{"records": recordsList})
}

// BatchDeleteRequest is the request body for batch deleting records.
type BatchDeleteRequest struct {
	IDs []string `json:"ids"`
}

// BatchDelete deletes multiple records.
func (h *Record) BatchDelete(c *mizu.Ctx) error {
	var req BatchDeleteRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.records.DeleteBatch(c.Context(), req.IDs); err != nil {
		return InternalError(c, "failed to delete records")
	}

	return NoContent(c)
}
