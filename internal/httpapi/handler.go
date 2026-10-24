package httpapi

import (
	"net"
	"slices"
	"strconv"
	"strings"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/GDGBabcockUniversity/auth-v2/internal/httpapi/dto"
	"github.com/gofiber/fiber/v3"
)

func profileUpdates(body dto.ProfileUpdateRequest) map[string]any {
	m := map[string]any{}
	if body.FullName != nil {
		m["full_name"] = *body.FullName
	}
	if body.WhatsAppNumber != nil {
		m["whatsapp_number"] = *body.WhatsAppNumber
	}
	if body.AvatarURL != nil {
		m["avatar_url"] = *body.AvatarURL
	}
	if body.Gender != nil {
		m["gender"] = *body.Gender
	}
	if body.Birthday != nil {
		m["birthday"] = *body.Birthday
	}
	if body.StudentStatus != nil {
		m["student_status"] = *body.StudentStatus
	}
	if body.MatricNo != nil {
		m["matric_no"] = *body.MatricNo
	}
	if body.Department != nil {
		m["department"] = *body.Department
	}
	if body.Faculty != nil {
		m["faculty"] = *body.Faculty
	}
	if body.PrimaryTrack != nil {
		m["primary_track"] = *body.PrimaryTrack
	}
	if body.SecondaryTrack != nil {
		m["secondary_track"] = *body.SecondaryTrack
	}
	if body.PrimarySkillLevel != nil {
		m["primary_skill_level"] = *body.PrimarySkillLevel
	}
	if body.SecondarySkillLevel != nil {
		m["secondary_skill_level"] = *body.SecondarySkillLevel
	}
	if body.TOSAgreed != nil {
		m["tos_agreed"] = *body.TOSAgreed
	}
	if body.TOSAgreedAt != nil {
		m["tos_agreed_at"] = *body.TOSAgreedAt
	}
	if body.TOSVersion != nil {
		m["tos_version"] = *body.TOSVersion
	}
	return m
}

func adminUpdates(body dto.AdminUpdateRequest) map[string]any {
	m := map[string]any{}
	if body.Roles != nil {
		roles := make([]any, len(body.Roles))
		for i, r := range body.Roles {
			roles[i] = r
		}
		m["roles"] = roles
	}
	if body.Teams != nil {
		teams := make([]any, len(body.Teams))
		for i, t := range body.Teams {
			teams[i] = t
		}
		m["teams"] = teams
	}
	if body.IsActive != nil {
		m["is_active"] = *body.IsActive
	}
	return m
}

// HealthCheck godoc
// @Summary      Health check
// @Tags         Health
// @Produce      json
// @Success      200  {object}  dto.HealthResponse
// @Router       /health [get]
func (a API) health(c fiber.Ctx) error {
	return c.JSON(dto.HealthResponse{Status: "healthy"})
}

// Login godoc
// @Summary      Login with Firebase ID token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.LoginRequest  true  "Firebase ID token"
// @Success      200   {object}  dto.LoginResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      403   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Router       /auth/login [post]
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

// Refresh godoc
// @Summary      Refresh access token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.RefreshRequest  true  "Refresh token"
// @Success      200   {object}  dto.TokensResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Router       /auth/refresh [post]
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

// Logout godoc
// @Summary      Logout
// @Tags         Auth
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.LogoutRequest  false  "Optional refresh token to revoke"
// @Success      200   {object}  dto.LogoutResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Router       /auth/logout [post]
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

// Me godoc
// @Summary      Current user profile
// @Tags         Auth
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  dto.ProfileResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Router       /auth/me [get]
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

// UpdateProfile godoc
// @Summary      Update current profile
// @Tags         Auth
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.ProfileUpdateRequest  true  "Fields to update"
// @Success      200   {object}  dto.ProfileResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      403   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Router       /auth/profile [put]
func (a API) profile(c fiber.Ctx) error {
	var body dto.ProfileUpdateRequest
	if err := c.Bind().Body(&body); err != nil {
		return domain.ErrBadInput
	}
	user := c.Locals("user").(domain.User)
	u, err := a.Service.UpdateProfile(c.Context(), user.ID, profileUpdates(body))
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

	if slices.Contains(user.Roles, "admin") {
		return c.Next()
	}
	return c.Status(403).JSON(dto.ErrorResponse{Success: false, Error: "Insufficient permissions"})
}

// AdminListUsers godoc
// @Summary      List users (admin)
// @Tags         Admin
// @Security     BearerAuth
// @Produce      json
// @Param        page    query     int     false  "Page (default 1)"
// @Param        limit   query     int     false  "Page size (default 20, max 100)"
// @Param        search  query     string  false  "Search email/name"
// @Param        role    query     string  false  "Filter by role"
// @Success      200     {object}  dto.AdminListResponse
// @Failure      401     {object}  dto.ErrorResponse
// @Failure      403     {object}  dto.ErrorResponse
// @Router       /admin/users [get]
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

// AdminUpdateUser godoc
// @Summary      Update user (admin)
// @Tags         Admin
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string                 true  "User UUID"
// @Param        body  body      dto.AdminUpdateRequest  true  "roles, teams, is_active"
// @Success      200   {object}  dto.AdminUserResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      403   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Router       /admin/users/{id} [put]
func (a API) adminUpdate(c fiber.Ctx) error {
	var body dto.AdminUpdateRequest
	if err := c.Bind().Body(&body); err != nil {
		return domain.ErrBadInput
	}
	user := c.Locals("user").(domain.User)
	u, err := a.Service.UpdateUser(c.Context(), user.ID, c.Params("id"), adminUpdates(body))
	if err != nil {
		return err
	}
	return c.JSON(dto.AdminUserResponse{Success: true, User: dto.AdminUser{
		ID: u.ID, Email: u.Email, FullName: u.FullName, AvatarURL: u.AvatarURL,
		Roles: u.Roles, Teams: u.Teams, IsActive: u.IsActive,
		CreatedAt: u.CreatedAt, LastLoginAt: u.LastLoginAt,
	}})
}