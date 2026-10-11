package httpapi

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
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
		case errors.Is(e, auth.ErrUnauthorized):
			status = 401
			message = "Authentication required"
		case errors.Is(e, auth.ErrInactive):
			status = 403
			message = "Account is inactive"
		case errors.Is(e, auth.ErrConflict):
			status = 409
			message = "Account needs identity linking"
		case errors.Is(e, auth.ErrBadInput):
			status = 400
			message = "Invalid request"
		case errors.Is(e, domain.ErrNotFound):
			status = 404
			message = "User not found"
		}
		return c.Status(status).JSON(fiber.Map{"success": false, "error": message})
	}})
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{AllowOrigins: a.Origins, AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"}, AllowMethods: []string{"GET", "POST", "PUT", "OPTIONS"}}))
	app.Get("/health", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"status": "healthy"}) })
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
func parse(c fiber.Ctx, dst any) error {
	if err := c.Bind().Body(dst); err != nil {
		return auth.ErrBadInput
	}
	return nil
}
func reqContext(c fiber.Ctx) context.Context { return c.Context() }
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func (a API) login(c fiber.Ctx) error {
	var body struct {
		FirebaseToken string `json:"firebase_token"`
	}
	if err := parse(c, &body); err != nil {
		return err
	}
	if body.FirebaseToken == "" {
		return auth.ErrBadInput
	}
	ip := net.ParseIP(c.IP())
	var addr *string
	if ip != nil {
		v := ip.String()
		addr = &v
	}
	user, tokens, err := a.Service.Login(reqContext(c), body.FirebaseToken, addr, optional(c.Get("User-Agent")))
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"success": true, "user": fiber.Map{"id": user.ID, "email": user.Email, "full_name": user.FullName, "avatar_url": user.AvatarURL, "roles": user.Roles, "teams": user.Teams}, "tokens": tokens})
}
func (a API) refresh(c fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := parse(c, &body); err != nil {
		return err
	}
	if body.RefreshToken == "" {
		return auth.ErrBadInput
	}
	tokens, err := a.Service.Refresh(reqContext(c), body.RefreshToken)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"success": true, "tokens": tokens})
}
func (a API) authorize(c fiber.Ctx) error {
	h := strings.Fields(c.Get("Authorization"))
	if len(h) != 2 || !strings.EqualFold(h[0], "Bearer") {
		return auth.ErrUnauthorized
	}
	user, claims, err := a.Service.AccessUser(reqContext(c), h[1])
	if err != nil {
		return err
	}
	c.Locals("user", user)
	claims.Email = user.Email
	claims.FullName = user.FullName
	claims.Roles = []string(user.Roles)
	claims.Teams = []string(user.Teams)
	c.Locals("claims", claims)
	return c.Next()
}
func current(c fiber.Ctx) domain.User { return c.Locals("user").(domain.User) }
func (a API) logout(c fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if len(c.Body()) > 0 {
		if err := parse(c, &body); err != nil {
			return err
		}
	}
	if err := a.Service.Logout(reqContext(c), current(c).ID, body.RefreshToken); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"success": true, "message": "Logged out successfully"})
}
func profileUser(u domain.User) fiber.Map {
	return fiber.Map{"id": u.ID, "firebase_uid": u.FirebaseUID, "email": u.Email, "full_name": u.FullName, "whatsapp_number": u.WhatsAppNumber, "avatar_url": u.AvatarURL, "email_verified": u.EmailVerified, "gender": u.Gender, "birthday": u.Birthday, "student_status": u.StudentStatus, "matric_no": u.MatricNo, "department": u.Department, "faculty": u.Faculty, "primary_track": u.PrimaryTrack, "secondary_track": u.SecondaryTrack, "primary_skill_level": u.PrimarySkillLevel, "secondary_skill_level": u.SecondarySkillLevel, "teams": u.Teams, "roles": u.Roles, "tos_agreed": u.TOSAgreed, "tos_agreed_at": u.TOSAgreedAt, "tos_version": u.TOSVersion, "created_at": u.CreatedAt, "updated_at": u.UpdatedAt, "last_login_at": u.LastLoginAt, "is_active": u.IsActive}
}
func (a API) me(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "user": profileUser(current(c))})
}
func (a API) verify(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "valid": true, "user": c.Locals("claims")})
}
func (a API) profile(c fiber.Ctx) error {
	var body map[string]any
	if err := parse(c, &body); err != nil {
		return err
	}
	u, err := a.Service.UpdateProfile(reqContext(c), current(c).ID, body)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"success": true, "user": profileUser(u)})
}
func (a API) requireAdmin(c fiber.Ctx) error {
	for _, r := range current(c).Roles {
		if r == "admin" {
			return c.Next()
		}
	}
	return c.Status(403).JSON(fiber.Map{"success": false, "error": "Insufficient permissions"})
}
func (a API) adminList(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	users, total, err := a.Service.ListUsers(reqContext(c), c.Query("search"), c.Query("role"), page, limit)
	if err != nil {
		return err
	}
	out := make([]fiber.Map, 0, len(users))
	for _, u := range users {
		out = append(out, adminUser(u))
	}
	return c.JSON(fiber.Map{"success": true, "users": out, "total": total, "page": page, "limit": limit})
}
func adminUser(u domain.User) fiber.Map {
	return fiber.Map{"id": u.ID, "email": u.Email, "full_name": u.FullName, "avatar_url": u.AvatarURL, "roles": u.Roles, "teams": u.Teams, "is_active": u.IsActive, "created_at": u.CreatedAt, "last_login_at": u.LastLoginAt}
}
func (a API) adminUpdate(c fiber.Ctx) error {
	var body map[string]any
	if err := parse(c, &body); err != nil {
		return err
	}
	u, err := a.Service.UpdateUser(reqContext(c), current(c).ID, c.Params("id"), body)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"success": true, "user": adminUser(u)})
}
