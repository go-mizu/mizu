package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/activities"
	"github.com/go-mizu/blueprints/kanban/feature/columns"
	"github.com/go-mizu/blueprints/kanban/feature/cycles"
	"github.com/go-mizu/blueprints/kanban/feature/issues"
	"github.com/go-mizu/blueprints/kanban/feature/projects"
)

// Issue handles issue endpoints.
type Issue struct {
	issues     issues.API
	projects   projects.API
	columns    columns.API
	cycles     cycles.API
	activities activities.API
	getUserID  func(*mizu.Ctx) string
}

// NewIssue creates a new issue handler.
func NewIssue(issues issues.API, projects projects.API, columns columns.API, cycles cycles.API, activities activities.API, getUserID func(*mizu.Ctx) string) *Issue {
	return &Issue{issues: issues, projects: projects, columns: columns, cycles: cycles, activities: activities, getUserID: getUserID}
}

// List returns all issues in a project.
func (h *Issue) List(c *mizu.Ctx) error {
	projectID := c.Param("projectID")

	list, err := h.issues.ListByProject(c.Context(), projectID)
	if err != nil {
		return InternalError(c, "failed to list issues")
	}

	return OK(c, list)
}

// ListByColumn returns all issues in a column.
func (h *Issue) ListByColumn(c *mizu.Ctx) error {
	columnID := c.Param("columnID")

	list, err := h.issues.ListByColumn(c.Context(), columnID)
	if err != nil {
		return InternalError(c, "failed to list issues")
	}

	return OK(c, list)
}

// ListByCycle returns all issues in a cycle.
func (h *Issue) ListByCycle(c *mizu.Ctx) error {
	cycleID := c.Param("cycleID")

	list, err := h.issues.ListByCycle(c.Context(), cycleID)
	if err != nil {
		return InternalError(c, "failed to list issues")
	}

	return OK(c, list)
}

// Create creates a new issue.
func (h *Issue) Create(c *mizu.Ctx) error {
	projectID := c.Param("projectID")
	userID := h.getUserID(c)

	var in issues.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		c.Logger().Error("failed to bind JSON", "error", err)
		return BadRequest(c, "invalid request body")
	}

	// Validate project exists
	project, err := h.projects.GetByID(c.Context(), projectID)
	if err != nil || project == nil {
		c.Logger().Error("project not found", "projectID", projectID, "error", err)
		return NotFound(c, "project not found")
	}

	issue, err := h.issues.Create(c.Context(), projectID, userID, &in)
	if err != nil {
		c.Logger().Error("failed to create issue", "projectID", projectID, "userID", userID, "title", in.Title, "error", err)
		return InternalError(c, "failed to create issue: "+err.Error())
	}

	// Log activity
	h.activities.Create(c.Context(), issue.ID, userID, &activities.CreateIn{
		Action:   activities.ActionIssueCreated,
		NewValue: issue.Title,
	})

	return Created(c, issue)
}

// Get returns an issue by key.
func (h *Issue) Get(c *mizu.Ctx) error {
	key := c.Param("key")

	issue, err := h.issues.GetByKey(c.Context(), key)
	if err != nil {
		if err == issues.ErrNotFound {
			return NotFound(c, "issue not found")
		}
		return InternalError(c, "failed to get issue")
	}

	return OK(c, issue)
}

// Update updates an issue.
func (h *Issue) Update(c *mizu.Ctx) error {
	key := c.Param("key")
	userID := h.getUserID(c)

	issue, err := h.issues.GetByKey(c.Context(), key)
	if err != nil {
		if err == issues.ErrNotFound {
			return NotFound(c, "issue not found")
		}
		return InternalError(c, "failed to get issue")
	}

	var in issues.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	// Track changes for activity logging
	oldPriority := issue.Priority

	updated, err := h.issues.Update(c.Context(), issue.ID, &in)
	if err != nil {
		return InternalError(c, "failed to update issue")
	}

	// Log activities for changes
	ctx := c.Context()

	// Priority change
	if in.Priority != nil && *in.Priority != oldPriority {
		priorityNames := map[int]string{0: "None", 1: "Urgent", 2: "High", 3: "Medium", 4: "Low"}
		h.activities.Create(ctx, issue.ID, userID, &activities.CreateIn{
			Action:   activities.ActionPriorityChanged,
			OldValue: priorityNames[oldPriority],
			NewValue: priorityNames[*in.Priority],
		})
	}

	// Title change
	if in.Title != nil && *in.Title != issue.Title {
		h.activities.Create(ctx, issue.ID, userID, &activities.CreateIn{
			Action:   activities.ActionTitleChanged,
			OldValue: issue.Title,
			NewValue: *in.Title,
		})
	}

	// Description change
	if in.Description != nil && *in.Description != issue.Description {
		h.activities.Create(ctx, issue.ID, userID, &activities.CreateIn{
			Action: activities.ActionDescChanged,
		})
	}

	// Due date change
	if in.DueDate != nil {
		h.activities.Create(ctx, issue.ID, userID, &activities.CreateIn{
			Action:   activities.ActionDueDateSet,
			NewValue: in.DueDate.Format("Jan 2, 2006"),
		})
	}

	// Start date change
	if in.StartDate != nil {
		h.activities.Create(ctx, issue.ID, userID, &activities.CreateIn{
			Action:   activities.ActionStartDateSet,
			NewValue: in.StartDate.Format("Jan 2, 2006"),
		})
	}

	return OK(c, updated)
}

// Delete deletes an issue.
func (h *Issue) Delete(c *mizu.Ctx) error {
	key := c.Param("key")

	issue, err := h.issues.GetByKey(c.Context(), key)
	if err != nil {
		if err == issues.ErrNotFound {
			return NotFound(c, "issue not found")
		}
		return InternalError(c, "failed to get issue")
	}

	if err := h.issues.Delete(c.Context(), issue.ID); err != nil {
		return InternalError(c, "failed to delete issue")
	}

	return OK(c, map[string]string{"message": "issue deleted"})
}

// Move moves an issue to a new column/position.
func (h *Issue) Move(c *mizu.Ctx) error {
	key := c.Param("key")
	userID := h.getUserID(c)

	issue, err := h.issues.GetByKey(c.Context(), key)
	if err != nil {
		if err == issues.ErrNotFound {
			return NotFound(c, "issue not found")
		}
		c.Logger().Error("failed to get issue for move", "key", key, "error", err)
		return InternalError(c, "failed to get issue")
	}

	var in issues.MoveIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		c.Logger().Error("failed to bind move JSON", "key", key, "error", err)
		return BadRequest(c, "invalid request body")
	}

	if in.ColumnID == "" {
		return BadRequest(c, "column_id is required")
	}

	// Get old column name for activity
	oldColumnName := ""
	if oldCol, err := h.columns.GetByID(c.Context(), issue.ColumnID); err == nil && oldCol != nil {
		oldColumnName = oldCol.Name
	}

	updated, err := h.issues.Move(c.Context(), issue.ID, &in)
	if err != nil {
		c.Logger().Error("failed to move issue", "key", key, "issueID", issue.ID, "columnID", in.ColumnID, "position", in.Position, "error", err)
		return InternalError(c, "failed to move issue")
	}

	// Log status change activity if column changed
	if in.ColumnID != issue.ColumnID {
		newColumnName := ""
		if newCol, err := h.columns.GetByID(c.Context(), in.ColumnID); err == nil && newCol != nil {
			newColumnName = newCol.Name
		}
		h.activities.Create(c.Context(), issue.ID, userID, &activities.CreateIn{
			Action:   activities.ActionStatusChanged,
			OldValue: oldColumnName,
			NewValue: newColumnName,
		})
	}

	return OK(c, updated)
}

// AttachCycle attaches an issue to a cycle.
func (h *Issue) AttachCycle(c *mizu.Ctx) error {
	key := c.Param("key")
	userID := h.getUserID(c)

	issue, err := h.issues.GetByKey(c.Context(), key)
	if err != nil {
		if err == issues.ErrNotFound {
			return NotFound(c, "issue not found")
		}
		return InternalError(c, "failed to get issue")
	}

	var in struct {
		CycleID string `json:"cycle_id"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.issues.AttachCycle(c.Context(), issue.ID, in.CycleID); err != nil {
		return InternalError(c, "failed to attach cycle")
	}

	// Log activity
	cycleName := ""
	if cycle, err := h.cycles.GetByID(c.Context(), in.CycleID); err == nil && cycle != nil {
		cycleName = cycle.Name
	}
	h.activities.Create(c.Context(), issue.ID, userID, &activities.CreateIn{
		Action:   activities.ActionCycleAttached,
		NewValue: cycleName,
	})

	return OK(c, map[string]string{"message": "cycle attached"})
}

// DetachCycle detaches an issue from its cycle.
func (h *Issue) DetachCycle(c *mizu.Ctx) error {
	key := c.Param("key")
	userID := h.getUserID(c)

	issue, err := h.issues.GetByKey(c.Context(), key)
	if err != nil {
		if err == issues.ErrNotFound {
			return NotFound(c, "issue not found")
		}
		return InternalError(c, "failed to get issue")
	}

	// Get old cycle name for activity
	oldCycleName := ""
	if issue.CycleID != "" {
		if cycle, err := h.cycles.GetByID(c.Context(), issue.CycleID); err == nil && cycle != nil {
			oldCycleName = cycle.Name
		}
	}

	if err := h.issues.DetachCycle(c.Context(), issue.ID); err != nil {
		return InternalError(c, "failed to detach cycle")
	}

	// Log activity
	h.activities.Create(c.Context(), issue.ID, userID, &activities.CreateIn{
		Action:   activities.ActionCycleDetached,
		OldValue: oldCycleName,
	})

	return OK(c, map[string]string{"message": "cycle detached"})
}

// Search searches for issues.
func (h *Issue) Search(c *mizu.Ctx) error {
	projectID := c.Param("projectID")
	query := c.Query("q")

	if query == "" {
		return BadRequest(c, "query parameter 'q' is required")
	}

	limit := 20 // default
	list, err := h.issues.Search(c.Context(), projectID, query, limit)
	if err != nil {
		return InternalError(c, "failed to search issues")
	}

	return OK(c, list)
}
