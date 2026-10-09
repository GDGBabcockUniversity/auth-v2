package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/GDGBabcockUniversity/auth-v2/internal/dto"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

type API struct {
	Auth          auth.Service
	Audience      string
	Log           *slog.Logger
	SecureCookies bool
}

func (a API) App() *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: a.errorHandler,
	})

	app.Use(requestid.New())
	app.Use(recover.New())

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy"})
	})

	g := app.Group("/auth")
	g.Post("/register", a.register)
	g.Post("/login", a.login)
	g.Get("/google", a.googleStart)
	g.Get("/google/callback", a.googleCallback)
	g.Post("/refresh", a.refresh)
	g.Get("/me", a.requireAuth, a.me)
	g.Get("/verify", a.requireAuth, func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true, "valid": true})
	})
	g.Post("/logout", a.requireAuth, func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	admin := app.Group("/admin", a.requireAuth, a.requireAdmin)
	admin.Get("/users", a.listUsers)
	admin.Get("/users/:id", a.getUser)
	admin.Patch("/users/:id/active", a.setUserActive)
	admin.Delete("/users/:id", a.deleteUser)

	return app
}

const googleStateCookie = "google_oauth_state"

func (a API) googleStart(c fiber.Ctx) error {
	if a.Auth.Google == nil {
		return domain.ErrNotFound
	}
	state, e := oauthState()
	if e != nil {
		return e
	}
	c.Cookie(&fiber.Cookie{
		Name:     googleStateCookie,
		Value:    state,
		Path:     "/auth/google",
		MaxAge:   600,
		HTTPOnly: true,
		Secure:   a.SecureCookies,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
	return c.Redirect().To(a.Auth.Google.AuthorizationURL(state))
}

func (a API) googleCallback(c fiber.Ctx) error {
	state := c.Query("state")
	if state == "" || state != c.Cookies(googleStateCookie) {
		return domain.ErrUnauthorized
	}
	c.ClearCookie(googleStateCookie)

	if c.Query("error") != "" || c.Query("code") == "" {
		return domain.ErrUnauthorized
	}
	s, e := a.Auth.LoginWithGoogleCode(context.Background(), c.Query("code"))
	if e != nil {
		return e
	}
	return c.JSON(sessionResponse(s))
}

func oauthState() (string, error) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (a API) register(c fiber.Ctx) error {
	var in dto.RegisterRequest
	if e := c.Bind().Body(&in); e != nil {
		return domain.FieldError{Field: "body", Message: "must be valid JSON"}
	}
	s, e := a.Auth.Register(context.Background(), in.Email, in.FullName, in.Password)
	if e != nil {
		return e
	}
	return c.Status(fiber.StatusCreated).JSON(sessionResponse(s))
}
func (a API) login(c fiber.Ctx) error {
	var in dto.LoginRequest
	if e := c.Bind().Body(&in); e != nil {
		return domain.FieldError{Field: "body", Message: "must be valid JSON"}
	}
	s, e := a.Auth.Login(context.Background(), in.Email, in.Password)
	if e != nil {
		return e
	}
	return c.JSON(sessionResponse(s))
}
func (a API) refresh(c fiber.Ctx) error {
	var in dto.RefreshRequest
	if e := c.Bind().Body(&in); e != nil {
		return domain.FieldError{Field: "body", Message: "must be valid JSON"}
	}
	if in.RefreshToken == "" {
		return domain.FieldError{Field: "refresh_token", Message: "is required"}
	}
	s, e := a.Auth.RefreshSession(context.Background(), in.RefreshToken)
	if e != nil {
		return e
	}
	return c.JSON(sessionResponse(s))
}
func (a API) requireAuth(c fiber.Ctx) error {
	p := strings.Fields(c.Get("Authorization"))
	if len(p) != 2 || !strings.EqualFold(p[0], "bearer") {
		return domain.ErrUnauthorized
	}
	claims, e := a.Auth.Access.Verify(p[1], a.Audience, "access")
	if e != nil {
		return e
	}
	c.Locals("claims", claims)
	return c.Next()
}
func (a API) me(c fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)
	u, e := a.Auth.UserRepo.FindByID(context.Background(), claims.Subject)
	if e != nil {
		return e
	}
	return c.JSON(fiber.Map{"success": true, "user": publicUser(u)})
}
func publicUser(u domain.User) dto.UserResponse {
	return dto.UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		FullName:      u.FullName,
		Roles:         []string(u.Roles),
		EmailVerified: u.EmailVerified,
	}
}
func sessionResponse(s auth.Session) dto.SessionResponse {
	return dto.SessionResponse{
		Success: true,
		User:    publicUser(s.User),
		Tokens: dto.TokenResponse{
			AccessToken:  s.AccessToken,
			RefreshToken: s.RefreshToken,
			TokenType:    s.TokenType,
			ExpiresIn:    s.ExpiresIn,
		},
	}
}
func (a API) errorHandler(c fiber.Ctx, err error) error {

	status := fiber.StatusInternalServerError
	msg := "Internal server error"
	var fiberErr *fiber.Error
	switch {
	case errors.As(err, &fiberErr):
		status, msg = fiberErr.Code, fiberErr.Message
	case errors.Is(err, domain.ErrValidation):
		status, msg = 400, err.Error()
	case errors.As(err, new(domain.FieldError)):
		status, msg = 400, err.Error()
	case errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrInvalidToken), errors.Is(err, domain.ErrExpiredToken), errors.Is(err, domain.ErrRefreshReuse), errors.Is(err, domain.ErrInactiveAccount):
		status, msg = 401, "Authentication failed"
	case errors.Is(err, domain.ErrForbidden):
		status, msg = 403, "Forbidden"
	case errors.Is(err, domain.ErrNotFound):
		status, msg = 404, "Not found"
	case errors.Is(err, domain.ErrConflict):
		status, msg = 409, "Resource already exists"
	}
	if a.Log != nil && status >= 500 {
		a.Log.Error("request failed", "error", err, "request_id", requestid.FromContext(c))
	}
	return c.Status(status).JSON(dto.ErrorResponse{
		Message: msg,
	})
}

func (a API) requireAdmin(c fiber.Ctx) error {
	claims, ok := c.Locals("claims").(*auth.Claims)
	if !ok {
		return domain.ErrUnauthorized
	}
	for _, r := range claims.Roles {
		if strings.EqualFold(r, "ADMIN") {
			return c.Next()
		}
	}
	return domain.ErrForbidden
}

func (a API) listUsers(c fiber.Ctx) error {
	var q dto.ListUsersQuery
	if e := c.Bind().Query(&q); e != nil {
		return domain.FieldError{Field: "query", Message: "invalid"}
	}
	users, total, e := a.Auth.ListUsers(context.Background(), q.Limit, q.Offset, q.Active)
	if e != nil {
		return e
	}
	out := make([]dto.AdminUserResponse, 0, len(users))
	for _, u := range users {
		out = append(out, adminUser(u))
	}
	return c.JSON(dto.ListUsersResponse{
		Success: true,
		Users:   out,
		Total:   total,
		Limit:   q.Limit,
		Offset:  q.Offset,
	})
}

func (a API) getUser(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.FieldError{Field: "id", Message: "is required"}
	}
	u, e := a.Auth.UserRepo.FindByID(context.Background(), id)
	if e != nil {
		return e
	}
	return c.JSON(fiber.Map{"success": true, "user": adminUser(u)})
}

func (a API) setUserActive(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.FieldError{Field: "id", Message: "is required"}
	}
	var in dto.SetActiveRequest
	if e := c.Bind().Body(&in); e != nil {
		return domain.FieldError{Field: "body", Message: "must be valid JSON"}
	}

	claims := c.Locals("claims").(*auth.Claims)
	if claims.Subject == id && !in.Active {
		return domain.FieldError{Field: "id", Message: "cannot deactivate your own account"}
	}

	u, e := a.Auth.SetUserActive(context.Background(), id, in.Active)
	if e != nil {
		return e
	}
	return c.JSON(fiber.Map{"success": true, "user": adminUser(u)})
}

func (a API) deleteUser(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.FieldError{Field: "id", Message: "is required"}
	}

	claims := c.Locals("claims").(*auth.Claims)
	if claims.Subject == id {
		return domain.FieldError{Field: "id", Message: "cannot delete your own account"}
	}

	if e := a.Auth.DeleteUser(context.Background(), id); e != nil {
		return e
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func adminUser(u domain.User) dto.AdminUserResponse {
	return dto.AdminUserResponse{
		ID:            u.ID,
		Email:         u.Email,
		FullName:      u.FullName,
		Roles:         []string(u.Roles),
		Active:        u.Active,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     u.UpdatedAt.UTC().Format(time.RFC3339),
	}
}