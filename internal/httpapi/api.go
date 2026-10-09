package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/url"
	"strings"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/GDGBabcockUniversity/auth-v2/internal/dto"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

type API struct {
	Auth            auth.Service
	Audience        string
	Log             *slog.Logger
	SecureCookies   bool
	GoogleReturnURL string
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
	g.Post("/exchange", a.exchangeGoogleHandoff)
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

const (
	googleStateCookie       = "google_oauth_state"
	googleReturnStateCookie = "google_return_state"
)

func (a API) googleStart(c fiber.Ctx) error {
	if a.Auth.Google == nil {
		return domain.ErrNotFound
	}
	state, e := oauthState()
	if e != nil {
		return e
	}
	returnState := c.Query("state")
	if returnState != "" {
		raw, decodeErr := base64.RawURLEncoding.DecodeString(returnState)
		if decodeErr != nil || len(raw) < 24 || len(raw) > 64 {
			return domain.FieldError{Field: "state", Message: "is invalid"}
		}
		c.Cookie(&fiber.Cookie{
			Name: googleReturnStateCookie, Value: returnState,
			Path: "/auth/google", MaxAge: 600, HTTPOnly: true,
			Secure: a.SecureCookies, SameSite: fiber.CookieSameSiteLaxMode,
		})
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
	returnState := c.Cookies(googleReturnStateCookie)
	c.ClearCookie(googleStateCookie)
	c.ClearCookie(googleReturnStateCookie)

	if c.Query("error") != "" || c.Query("code") == "" {
		return domain.ErrUnauthorized
	}
	if a.GoogleReturnURL != "" && returnState != "" {
		handoff, err := a.Auth.CreateGoogleHandoff(context.Background(), c.Query("code"))
		if err != nil {
			return err
		}
		redirect, err := url.Parse(a.GoogleReturnURL)
		if err != nil || !redirect.IsAbs() || (redirect.Scheme != "https" && redirect.Scheme != "http") || redirect.Host == "" {
			return domain.ErrValidation
		}
		query := redirect.Query()
		query.Set("code", handoff)
		query.Set("state", returnState)
		redirect.RawQuery = query.Encode()
		c.Set("Cache-Control", "no-store")
		c.Set("Referrer-Policy", "no-referrer")
		return c.Redirect().To(redirect.String())
	}
	s, e := a.Auth.LoginWithGoogleCode(context.Background(), c.Query("code"))
	if e != nil {
		return e
	}
	return c.JSON(sessionResponse(s))
}

func (a API) exchangeGoogleHandoff(c fiber.Ctx) error {
	var in dto.GoogleHandoffRequest
	if e := c.Bind().Body(&in); e != nil || in.Code == "" {
		return domain.FieldError{Field: "code", Message: "is required"}
	}
	s, e := a.Auth.ExchangeGoogleHandoff(context.Background(), in.Code)
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
