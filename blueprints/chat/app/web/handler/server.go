package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/chat/feature/channels"
	"github.com/go-mizu/blueprints/chat/feature/members"
	"github.com/go-mizu/blueprints/chat/feature/roles"
	"github.com/go-mizu/blueprints/chat/feature/servers"
)

// Server handles server endpoints.
type Server struct {
	servers  servers.API
	channels channels.API
	members  members.API
	roles    roles.API
	getUserID func(*mizu.Ctx) string
}

// NewServer creates a new Server handler.
func NewServer(
	servers servers.API,
	channels channels.API,
	members members.API,
	roles roles.API,
	getUserID func(*mizu.Ctx) string,
) *Server {
	return &Server{
		servers:   servers,
		channels:  channels,
		members:   members,
		roles:     roles,
		getUserID: getUserID,
	}
}

// List lists servers the user is a member of.
func (h *Server) List(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	srvs, err := h.servers.ListByUser(c.Request().Context(), userID, 100, 0)
	if err != nil {
		return InternalError(c, "Failed to list servers")
	}

	return Success(c, srvs)
}

// Create creates a new server.
func (h *Server) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	var in servers.CreateIn
	if err := c.Bind(&in); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	if in.Name == "" {
		return BadRequest(c, "Server name is required")
	}

	ctx := c.Request().Context()

	// Create server
	srv, err := h.servers.Create(ctx, userID, &in)
	if err != nil {
		return InternalError(c, "Failed to create server")
	}

	// Create default role
	h.roles.(*roles.Service).CreateDefaultRole(ctx, srv.ID)

	// Create default channel
	ch, err := h.channels.Create(ctx, &channels.CreateIn{
		ServerID: srv.ID,
		Type:     channels.TypeText,
		Name:     "general",
	})
	if err == nil {
		h.servers.(*servers.Service).SetDefaultChannel(ctx, srv.ID, ch.ID)
	}

	// Add owner as member
	h.members.Join(ctx, srv.ID, userID)

	return Created(c, srv)
}

// Get returns a server.
func (h *Server) Get(c *mizu.Ctx) error {
	serverID := c.Param("id")

	srv, err := h.servers.GetByID(c.Request().Context(), serverID)
	if err != nil {
		return NotFound(c, "Server not found")
	}

	return Success(c, srv)
}

// Update updates a server.
func (h *Server) Update(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	serverID := c.Param("id")

	// Check ownership
	srv, err := h.servers.GetByID(c.Request().Context(), serverID)
	if err != nil {
		return NotFound(c, "Server not found")
	}
	if srv.OwnerID != userID {
		return Forbidden(c, "Only the owner can update the server")
	}

	var in servers.UpdateIn
	if err := c.Bind(&in); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	srv, err = h.servers.Update(c.Request().Context(), serverID, &in)
	if err != nil {
		return InternalError(c, "Failed to update server")
	}

	return Success(c, srv)
}

// Delete deletes a server.
func (h *Server) Delete(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	serverID := c.Param("id")

	// Check ownership
	srv, err := h.servers.GetByID(c.Request().Context(), serverID)
	if err != nil {
		return NotFound(c, "Server not found")
	}
	if srv.OwnerID != userID {
		return Forbidden(c, "Only the owner can delete the server")
	}

	if err := h.servers.Delete(c.Request().Context(), serverID); err != nil {
		return InternalError(c, "Failed to delete server")
	}

	return NoContent(c)
}

// ListChannels lists channels in a server.
func (h *Server) ListChannels(c *mizu.Ctx) error {
	serverID := c.Param("id")

	chs, err := h.channels.ListByServer(c.Request().Context(), serverID)
	if err != nil {
		return InternalError(c, "Failed to list channels")
	}

	return Success(c, chs)
}

// ListMembers lists members in a server.
func (h *Server) ListMembers(c *mizu.Ctx) error {
	serverID := c.Param("id")

	mems, err := h.members.List(c.Request().Context(), serverID, 100, 0)
	if err != nil {
		return InternalError(c, "Failed to list members")
	}

	return Success(c, mems)
}

// Join joins a server.
func (h *Server) Join(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	serverID := c.Param("id")
	ctx := c.Request().Context()

	member, err := h.members.Join(ctx, serverID, userID)
	if err != nil {
		if err == members.ErrBanned {
			return Forbidden(c, "You are banned from this server")
		}
		return InternalError(c, "Failed to join server")
	}

	// Increment member count
	h.servers.(*servers.Service).IncrementMemberCount(ctx, serverID)

	return Created(c, member)
}

// Leave leaves a server.
func (h *Server) Leave(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	serverID := c.Param("id")
	ctx := c.Request().Context()

	// Check if owner
	srv, err := h.servers.GetByID(ctx, serverID)
	if err != nil {
		return NotFound(c, "Server not found")
	}
	if srv.OwnerID == userID {
		return BadRequest(c, "Owner cannot leave the server. Transfer ownership or delete the server.")
	}

	if err := h.members.Leave(ctx, serverID, userID); err != nil {
		return InternalError(c, "Failed to leave server")
	}

	// Decrement member count
	h.servers.(*servers.Service).DecrementMemberCount(ctx, serverID)

	return NoContent(c)
}

// JoinByInvite joins a server by invite code.
func (h *Server) JoinByInvite(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	code := c.Param("code")
	ctx := c.Request().Context()

	srv, err := h.servers.GetByInviteCode(ctx, code)
	if err != nil {
		return NotFound(c, "Invalid invite code")
	}

	member, err := h.members.Join(ctx, srv.ID, userID)
	if err != nil {
		if err == members.ErrBanned {
			return Forbidden(c, "You are banned from this server")
		}
		return InternalError(c, "Failed to join server")
	}

	h.servers.(*servers.Service).IncrementMemberCount(ctx, srv.ID)

	return Created(c, map[string]any{
		"server": srv,
		"member": member,
	})
}

// ListRoles lists roles in a server.
func (h *Server) ListRoles(c *mizu.Ctx) error {
	serverID := c.Param("id")

	rs, err := h.roles.ListByServer(c.Request().Context(), serverID)
	if err != nil {
		return InternalError(c, "Failed to list roles")
	}

	return Success(c, rs)
}

// ListPublic lists public servers.
func (h *Server) ListPublic(c *mizu.Ctx) error {
	srvs, err := h.servers.ListPublic(c.Request().Context(), 50, 0)
	if err != nil {
		return InternalError(c, "Failed to list servers")
	}
	return Success(c, srvs)
}
