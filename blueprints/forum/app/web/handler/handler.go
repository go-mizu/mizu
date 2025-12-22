// Package handler provides HTTP request handlers.
package handler

import (
	"html/template"

	"github.com/go-mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/blueprints/forum/feature/forums"
	"github.com/go-mizu/blueprints/forum/feature/posts"
	"github.com/go-mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/blueprints/forum/feature/votes"
	"github.com/go-mizu/mizu"
)

// Handler contains all HTTP handlers.
type Handler struct {
	templates *template.Template

	// Services
	accounts accounts.API
	forums   forums.API
	threads  threads.API
	posts    posts.API
	votes    votes.API
}

// New creates a new handler.
func New(
	templates *template.Template,
	accounts accounts.API,
	forums forums.API,
	threads threads.API,
	posts posts.API,
	votes votes.API,
) *Handler {
	return &Handler{
		templates: templates,
		accounts:  accounts,
		forums:    forums,
		threads:   threads,
		posts:     posts,
		votes:     votes,
	}
}

// Home renders the home page.
func (h *Handler) Home(c *mizu.Ctx) error {
	return c.HTML(200, `
		<html>
		<head><title>Forum</title></head>
		<body>
			<h1>Welcome to Forum</h1>
			<p>This is a production-ready forum platform built with Mizu.</p>
			<ul>
				<li><a href="/login">Login</a></li>
				<li><a href="/register">Register</a></li>
			</ul>
		</body>
		</html>
	`)
}

// LoginPage renders the login page.
func (h *Handler) LoginPage(c *mizu.Ctx) error {
	return c.HTML(200, `
		<html>
		<head><title>Login - Forum</title></head>
		<body>
			<h1>Login</h1>
			<form method="POST" action="/login">
				<div>
					<label>Username or Email:</label>
					<input type="text" name="username_or_email" required>
				</div>
				<div>
					<label>Password:</label>
					<input type="password" name="password" required>
				</div>
				<button type="submit">Login</button>
			</form>
			<p><a href="/register">Don't have an account? Register</a></p>
		</body>
		</html>
	`)
}

// Login handles login submission.
func (h *Handler) Login(c *mizu.Ctx) error {
	// TODO: Implement login logic
	return c.Redirect(302, "/")
}

// RegisterPage renders the registration page.
func (h *Handler) RegisterPage(c *mizu.Ctx) error {
	return c.HTML(200, `
		<html>
		<head><title>Register - Forum</title></head>
		<body>
			<h1>Register</h1>
			<form method="POST" action="/register">
				<div>
					<label>Username:</label>
					<input type="text" name="username" required>
				</div>
				<div>
					<label>Email:</label>
					<input type="email" name="email" required>
				</div>
				<div>
					<label>Password:</label>
					<input type="password" name="password" required>
				</div>
				<button type="submit">Register</button>
			</form>
			<p><a href="/login">Already have an account? Login</a></p>
		</body>
		</html>
	`)
}

// Register handles registration submission.
func (h *Handler) Register(c *mizu.Ctx) error {
	// TODO: Implement registration logic
	return c.Redirect(302, "/login")
}

// Logout handles logout.
func (h *Handler) Logout(c *mizu.Ctx) error {
	// TODO: Implement logout logic
	return c.Redirect(302, "/")
}

// ForumPage renders a forum page.
func (h *Handler) ForumPage(c *mizu.Ctx) error {
	slug := c.Param("slug")
	return c.HTML(200, "<h1>Forum: "+slug+"</h1>")
}

// ThreadPage renders a thread page.
func (h *Handler) ThreadPage(c *mizu.Ctx) error {
	id := c.Param("id")
	return c.HTML(200, "<h1>Thread: "+id+"</h1>")
}

// ProfilePage renders a user profile page.
func (h *Handler) ProfilePage(c *mizu.Ctx) error {
	username := c.Param("username")
	return c.HTML(200, "<h1>Profile: "+username+"</h1>")
}

// API handlers (stubs)
func (h *Handler) APIRegister(c *mizu.Ctx) error       { return c.JSON(200, map[string]string{"message": "API register"}) }
func (h *Handler) APILogin(c *mizu.Ctx) error          { return c.JSON(200, map[string]string{"message": "API login"}) }
func (h *Handler) APILogout(c *mizu.Ctx) error         { return c.JSON(200, map[string]string{"message": "API logout"}) }
func (h *Handler) APIListForums(c *mizu.Ctx) error     { return c.JSON(200, map[string]string{"message": "API list forums"}) }
func (h *Handler) APICreateForum(c *mizu.Ctx) error    { return c.JSON(200, map[string]string{"message": "API create forum"}) }
func (h *Handler) APIGetForum(c *mizu.Ctx) error       { return c.JSON(200, map[string]string{"message": "API get forum"}) }
func (h *Handler) APIListThreads(c *mizu.Ctx) error    { return c.JSON(200, map[string]string{"message": "API list threads"}) }
func (h *Handler) APICreateThread(c *mizu.Ctx) error   { return c.JSON(200, map[string]string{"message": "API create thread"}) }
func (h *Handler) APIGetThread(c *mizu.Ctx) error      { return c.JSON(200, map[string]string{"message": "API get thread"}) }
func (h *Handler) APIListPosts(c *mizu.Ctx) error      { return c.JSON(200, map[string]string{"message": "API list posts"}) }
func (h *Handler) APICreatePost(c *mizu.Ctx) error     { return c.JSON(200, map[string]string{"message": "API create post"}) }
func (h *Handler) APIVoteThread(c *mizu.Ctx) error     { return c.JSON(200, map[string]string{"message": "API vote thread"}) }
func (h *Handler) APIVotePost(c *mizu.Ctx) error       { return c.JSON(200, map[string]string{"message": "API vote post"}) }
