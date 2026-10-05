package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/GDGBabcockUniversity/auth-v2/internal/dto"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

type API struct {
	Auth     auth.Service
	Audience string
	Log      *slog.Logger
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
	g.Post("/refresh", a.refresh)
	g.Get("/me", a.requireAuth, a.me)
	g.Get("/verify", a.requireAuth, func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true, "valid": true})
	})
	g.Post("/logout", a.requireAuth, func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	return app
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
	code, msg := "internal_error", "Internal server error"
	switch {
	case errors.Is(err, domain.ErrValidation):
		status, code, msg = 400, "validation_error", err.Error()
	case errors.As(err, new(domain.FieldError)):
		status, code, msg = 400, "validation_error", err.Error()
	case errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrInvalidToken), errors.Is(err, domain.ErrExpiredToken), errors.Is(err, domain.ErrRefreshReuse), errors.Is(err, domain.ErrInactiveAccount):
		status, code, msg = 401, "unauthorized", "Authentication failed"
	case errors.Is(err, domain.ErrForbidden):
		status, code, msg = 403, "forbidden", "Forbidden"
	case errors.Is(err, domain.ErrNotFound):
		status, code, msg = 404, "not_found", "Not found"
	case errors.Is(err, domain.ErrConflict):
		status, code, msg = 409, "conflict", "Resource already exists"
	}
	if a.Log != nil && status >= 500 {
		a.Log.Error("request failed", "error", err, "request_id", requestid.FromContext(c))
	}
	return c.Status(status).JSON(dto.ErrorResponse{
		Success:   false,
		Error:     msg,
		Code:      code,
		RequestID: requestid.FromContext(c),
	})
}
