package httpapi

import (
	"errors"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/GDGBabcockUniversity/auth-v2/internal/httpapi/dto"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

type API struct {
	Service auth.Service
	Origins []string
}

func (a API) App() *fiber.App {
	app := fiber.New(fiber.Config{BodyLimit: 1 << 20, ErrorHandler: func(c fiber.Ctx, e error) error {
		status := fiber.StatusInternalServerError
		message := "Internal server error"
		switch {
		case errors.Is(e, domain.ErrUnauthorized):
			status = 401
			message = "Authentication required"
		case errors.Is(e, domain.ErrInactive):
			status = 403
			message = "Account is inactive"
		case errors.Is(e, domain.ErrConflict):
			status = 409
			message = "Account needs identity linking"
		case errors.Is(e, domain.ErrBadInput):
			status = 400
			message = "Invalid request"
		case errors.Is(e, domain.ErrNotFound):
			status = 404
			message = "User not found"
		}
		return c.Status(status).JSON(dto.ErrorResponse{Success: false, Error: message})
	}})
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{AllowOrigins: a.Origins, AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"}, AllowMethods: []string{"GET", "POST", "PUT", "OPTIONS"}}))
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(dto.HealthResponse{Status: "healthy"})
	})
	app.Post("/auth/login", a.login)
	app.Post("/auth/refresh", a.refresh)
	app.Post("/auth/logout", a.authorize, a.logout)
	app.Get("/auth/me", a.authorize, a.me)
	app.Put("/auth/profile", a.authorize, a.profile)
	app.Get("/auth/verify", a.authorize, a.verify)
	app.Get("/admin/users", a.authorize, a.requireAdmin, a.adminList)
	app.Put("/admin/users/:id", a.authorize, a.requireAdmin, a.adminUpdate)
	return app
}
