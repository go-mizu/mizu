package handler

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/issues"
	"github.com/go-mizu/blueprints/kanban/feature/projects"
	"github.com/go-mizu/blueprints/kanban/feature/workspaces"
)

// Issue handles issue endpoints.
type Issue struct {
	issues     issues.API
	projects   projects.API
	workspaces workspaces.API
	getUserID  func(*mizu.Ctx) string
}

// NewIssue creates a new issue handler.
func NewIssue(issues issues.API, projects projects.API, workspaces workspaces.API, getUserID func(*mizu.Ctx) string) *Issue {
	return &Issue{issues: issues, projects: projects, workspaces: workspaces, getUserID: getUserID}
}

// List returns all issues in a project.
func (h *Issue) List(c *mizu.Ctx) error {
	projectID := c.Param("projectID")

	filter := &issues.Filter{}
	if status := c.Query("status"); status != "" {
		filter.Status = status
	}
	if priority := c.Query("priority"); priority != "" {
		filter.Priority = priority
	}
	if issueType := c.Query("type"); issueType != "" {
		filter.Type = issueType
	}
	if assignee := c.Query("assignee"); assignee != "" {
		filter.AssigneeID = assignee
	}
	if sprint := c.Query("sprint"); sprint != "" {
		filter.SprintID = sprint
	}

	list, err := h.issues.ListByProject(c.Context(), projectID, filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to list issues"))
	}

	return c.JSON(http.StatusOK, list)
}

// Create creates a new issue.
func (h *Issue) Create(c *mizu.Ctx) error {
	projectID := c.Param("projectID")
	userID := h.getUserID(c)

	var in issues.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	issue, err := h.issues.Create(c.Context(), projectID, userID, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to create issue"))
	}

	return c.JSON(http.StatusCreated, issue)
}

// Get returns an issue by key.
func (h *Issue) Get(c *mizu.Ctx) error {
	key := c.Param("key")

	issue, err := h.issues.GetByKey(c.Context(), key)
	if err != nil {
		if err == issues.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("issue not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get issue"))
	}

	return c.JSON(http.StatusOK, issue)
}

// Update updates an issue.
func (h *Issue) Update(c *mizu.Ctx) error {
	key := c.Param("key")

	issue, err := h.issues.GetByKey(c.Context(), key)
	if err != nil {
		if err == issues.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("issue not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get issue"))
	}

	var in issues.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	updated, err := h.issues.Update(c.Context(), issue.ID, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to update issue"))
	}

	return c.JSON(http.StatusOK, updated)
}

// Delete deletes an issue.
func (h *Issue) Delete(c *mizu.Ctx) error {
	key := c.Param("key")

	issue, err := h.issues.GetByKey(c.Context(), key)
	if err != nil {
		if err == issues.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("issue not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get issue"))
	}

	if err := h.issues.Delete(c.Context(), issue.ID); err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to delete issue"))
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "issue deleted"})
}

// Move moves an issue to a new status/position.
func (h *Issue) Move(c *mizu.Ctx) error {
	key := c.Param("key")

	issue, err := h.issues.GetByKey(c.Context(), key)
	if err != nil {
		if err == issues.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("issue not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get issue"))
	}

	var in issues.MoveIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	updated, err := h.issues.Move(c.Context(), issue.ID, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to move issue"))
	}

	return c.JSON(http.StatusOK, updated)
}

// AddAssignee adds an assignee to an issue.
func (h *Issue) AddAssignee(c *mizu.Ctx) error {
	key := c.Param("key")

	issue, err := h.issues.GetByKey(c.Context(), key)
	if err != nil {
		if err == issues.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("issue not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get issue"))
	}

	var in struct {
		UserID string `json:"user_id"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	if err := h.issues.AddAssignee(c.Context(), issue.ID, in.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to add assignee"))
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "assignee added"})
}

// RemoveAssignee removes an assignee from an issue.
func (h *Issue) RemoveAssignee(c *mizu.Ctx) error {
	key := c.Param("key")
	userID := c.Param("userID")

	issue, err := h.issues.GetByKey(c.Context(), key)
	if err != nil {
		if err == issues.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("issue not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get issue"))
	}

	if err := h.issues.RemoveAssignee(c.Context(), issue.ID, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to remove assignee"))
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "assignee removed"})
}

// AddLabel adds a label to an issue.
func (h *Issue) AddLabel(c *mizu.Ctx) error {
	key := c.Param("key")

	issue, err := h.issues.GetByKey(c.Context(), key)
	if err != nil {
		if err == issues.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("issue not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get issue"))
	}

	var in struct {
		LabelID string `json:"label_id"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	if err := h.issues.AddLabel(c.Context(), issue.ID, in.LabelID); err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to add label"))
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "label added"})
}

// RemoveLabel removes a label from an issue.
func (h *Issue) RemoveLabel(c *mizu.Ctx) error {
	key := c.Param("key")
	labelID := c.Param("labelID")

	issue, err := h.issues.GetByKey(c.Context(), key)
	if err != nil {
		if err == issues.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("issue not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get issue"))
	}

	if err := h.issues.RemoveLabel(c.Context(), issue.ID, labelID); err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to remove label"))
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "label removed"})
}
