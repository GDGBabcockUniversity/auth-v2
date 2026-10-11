package httpapi

import (
	"net"
	"strconv"
	"strings"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/GDGBabcockUniversity/auth-v2/internal/httpapi/dto"
	"github.com/gofiber/fiber/v3"
)

func (a API) login(c fiber.Ctx) error {
	var body dto.LoginRequest
	if err := c.Bind().Body(&body); err != nil {
		return domain.ErrBadInput
	}
	if body.FirebaseToken == "" {
		return domain.ErrBadInput
	}
	ip := net.ParseIP(c.IP())
	var addr *string
	if ip != nil {
		v := ip.String()
		addr = &v
	}
	var agent *string
	if value := c.Get("User-Agent"); value != "" {
		agent = &value
	}
	user, tokens, err := a.Service.Login(c.Context(), body.FirebaseToken, addr, agent)
	if err != nil {
		return err
	}
	return c.JSON(dto.LoginResponse{
		Success: true,
		User: dto.LoginUser{
			ID: user.ID, Email: user.Email, FullName: user.FullName,
			AvatarURL: user.AvatarURL, Roles: user.Roles, Teams: user.Teams,
		},
		Tokens: dto.TokenPair{
			AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken,
			TokenType: tokens.TokenType, ExpiresIn: tokens.ExpiresIn,
		},
	})
}
func (a API) refresh(c fiber.Ctx) error {
	var body dto.RefreshRequest
	if err := c.Bind().Body(&body); err != nil {
		return domain.ErrBadInput
	}
	if body.RefreshToken == "" {
		return domain.ErrBadInput
	}
	tokens, err := a.Service.Refresh(c.Context(), body.RefreshToken)
	if err != nil {
		return err
	}
	return c.JSON(dto.TokensResponse{Success: true, Tokens: dto.TokenPair{
		AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken,
		TokenType: tokens.TokenType, ExpiresIn: tokens.ExpiresIn,
	}})
}
func (a API) authorize(c fiber.Ctx) error {
	h := strings.Fields(c.Get("Authorization"))
	if len(h) != 2 || !strings.EqualFold(h[0], "Bearer") {
		return domain.ErrUnauthorized
	}
	user, claims, err := a.Service.AccessUser(c.Context(), h[1])
	if err != nil {
		return err
	}
	c.Locals("user", user)
	claims.Roles = []string(user.Roles)
	claims.Teams = []string(user.Teams)
	c.Locals("claims", claims)
	return c.Next()
}
func (a API) logout(c fiber.Ctx) error {
	var body dto.LogoutRequest
	if len(c.Body()) > 0 {
		if err := c.Bind().Body(&body); err != nil {
			return domain.ErrBadInput
		}
	}
	user := c.Locals("user").(domain.User)
	if err := a.Service.Logout(c.Context(), user.ID, body.RefreshToken); err != nil {
		return err
	}
	return c.JSON(dto.LogoutResponse{Success: true, Message: "Logged out successfully"})
}
func (a API) me(c fiber.Ctx) error {
	u := c.Locals("user").(domain.User)
	return c.JSON(dto.ProfileResponse{Success: true, User: dto.ProfileUser{
		ID: u.ID, FirebaseUID: u.FirebaseUID, Email: u.Email, FullName: u.FullName,
		WhatsAppNumber: u.WhatsAppNumber, AvatarURL: u.AvatarURL, EmailVerified: u.EmailVerified,
		Gender: u.Gender, Birthday: u.Birthday, StudentStatus: u.StudentStatus,
		MatricNo: u.MatricNo, Department: u.Department, Faculty: u.Faculty,
		PrimaryTrack: u.PrimaryTrack, SecondaryTrack: u.SecondaryTrack,
		PrimarySkillLevel: u.PrimarySkillLevel, SecondarySkillLevel: u.SecondarySkillLevel,
		Teams: u.Teams, Roles: u.Roles, TOSAgreed: u.TOSAgreed,
		TOSAgreedAt: u.TOSAgreedAt, TOSVersion: u.TOSVersion, CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt, LastLoginAt: u.LastLoginAt, IsActive: u.IsActive,
	}})
}
func (a API) verify(c fiber.Ctx) error {
	return c.JSON(dto.VerifyResponse{Success: true, Valid: true, User: c.Locals("claims").(auth.Claims)})
}
func (a API) profile(c fiber.Ctx) error {
	var body dto.ProfileUpdateRequest
	if err := c.Bind().Body(&body); err != nil {
		return domain.ErrBadInput
	}
	user := c.Locals("user").(domain.User)
	u, err := a.Service.UpdateProfile(c.Context(), user.ID, map[string]any(body))
	if err != nil {
		return err
	}
	return c.JSON(dto.ProfileResponse{Success: true, User: dto.ProfileUser{
		ID: u.ID, FirebaseUID: u.FirebaseUID, Email: u.Email, FullName: u.FullName,
		WhatsAppNumber: u.WhatsAppNumber, AvatarURL: u.AvatarURL, EmailVerified: u.EmailVerified,
		Gender: u.Gender, Birthday: u.Birthday, StudentStatus: u.StudentStatus,
		MatricNo: u.MatricNo, Department: u.Department, Faculty: u.Faculty,
		PrimaryTrack: u.PrimaryTrack, SecondaryTrack: u.SecondaryTrack,
		PrimarySkillLevel: u.PrimarySkillLevel, SecondarySkillLevel: u.SecondarySkillLevel,
		Teams: u.Teams, Roles: u.Roles, TOSAgreed: u.TOSAgreed,
		TOSAgreedAt: u.TOSAgreedAt, TOSVersion: u.TOSVersion, CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt, LastLoginAt: u.LastLoginAt, IsActive: u.IsActive,
	}})
}
func (a API) requireAdmin(c fiber.Ctx) error {
	user := c.Locals("user").(domain.User)
	for _, r := range user.Roles {
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
	users, total, err := a.Service.ListUsers(c.Context(), c.Query("search"), c.Query("role"), page, limit)
	if err != nil {
		return err
	}
	out := make([]dto.AdminUser, 0, len(users))
	for _, u := range users {
		out = append(out, dto.AdminUser{
			ID: u.ID, Email: u.Email, FullName: u.FullName, AvatarURL: u.AvatarURL,
			Roles: u.Roles, Teams: u.Teams, IsActive: u.IsActive,
			CreatedAt: u.CreatedAt, LastLoginAt: u.LastLoginAt,
		})
	}
	return c.JSON(dto.AdminListResponse{Success: true, Users: out, Total: total, Page: page, Limit: limit})
}
func (a API) adminUpdate(c fiber.Ctx) error {
	var body dto.AdminUpdateRequest
	if err := c.Bind().Body(&body); err != nil {
		return domain.ErrBadInput
	}
	user := c.Locals("user").(domain.User)
	u, err := a.Service.UpdateUser(c.Context(), user.ID, c.Params("id"), map[string]any(body))
	if err != nil {
		return err
	}
	return c.JSON(dto.AdminUserResponse{Success: true, User: dto.AdminUser{
		ID: u.ID, Email: u.Email, FullName: u.FullName, AvatarURL: u.AvatarURL,
		Roles: u.Roles, Teams: u.Teams, IsActive: u.IsActive,
		CreatedAt: u.CreatedAt, LastLoginAt: u.LastLoginAt,
	}})
}
