package httpapi

import (
	"context"
	"net"
	"strconv"
	"strings"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/GDGBabcockUniversity/auth-v2/internal/httpapi/dto"
	"github.com/gofiber/fiber/v3"
)

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
	var body dto.LoginRequest
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
	return c.JSON(dto.LoginResponse{Success: true, User: dto.NewLoginUser(user), Tokens: dto.NewTokenPair(tokens)})
}
func (a API) refresh(c fiber.Ctx) error {
	var body dto.RefreshRequest
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
	return c.JSON(dto.TokensResponse{Success: true, Tokens: dto.NewTokenPair(tokens)})
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
	var body dto.LogoutRequest
	if len(c.Body()) > 0 {
		if err := parse(c, &body); err != nil {
			return err
		}
	}
	if err := a.Service.Logout(reqContext(c), current(c).ID, body.RefreshToken); err != nil {
		return err
	}
	return c.JSON(dto.LogoutResponse{Success: true, Message: "Logged out successfully"})
}
func (a API) me(c fiber.Ctx) error {
	return c.JSON(dto.ProfileResponse{Success: true, User: dto.NewProfileUser(current(c))})
}
func (a API) verify(c fiber.Ctx) error {
	return c.JSON(dto.VerifyResponse{Success: true, Valid: true, User: c.Locals("claims").(auth.Claims)})
}
func (a API) profile(c fiber.Ctx) error {
	var body dto.ProfileUpdateRequest
	if err := parse(c, &body); err != nil {
		return err
	}
	u, err := a.Service.UpdateProfile(reqContext(c), current(c).ID, map[string]any(body))
	if err != nil {
		return err
	}
	return c.JSON(dto.ProfileResponse{Success: true, User: dto.NewProfileUser(u)})
}
func (a API) requireAdmin(c fiber.Ctx) error {
	for _, r := range current(c).Roles {
		if r == "admin" {
			return c.Next()
		}
	}
	return c.Status(403).JSON(dto.ErrorResponse{Success: false, Error: "Insufficient permissions"})
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
	out := make([]dto.AdminUser, 0, len(users))
	for _, u := range users {
		out = append(out, dto.NewAdminUser(u))
	}
	return c.JSON(dto.AdminListResponse{Success: true, Users: out, Total: total, Page: page, Limit: limit})
}
func (a API) adminUpdate(c fiber.Ctx) error {
	var body dto.AdminUpdateRequest
	if err := parse(c, &body); err != nil {
		return err
	}
	u, err := a.Service.UpdateUser(reqContext(c), current(c).ID, c.Params("id"), map[string]any(body))
	if err != nil {
		return err
	}
	return c.JSON(dto.AdminUserResponse{Success: true, User: dto.NewAdminUser(u)})
}
