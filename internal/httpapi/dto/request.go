package dto

type LoginRequest struct {
	FirebaseToken string `json:"firebase_token" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

type ProfileUpdateRequest struct {
	FullName            *string `json:"full_name,omitempty" example:"Jane Doe"`
	WhatsAppNumber      *string `json:"whatsapp_number,omitempty" example:"+2348012345678"`
	AvatarURL           *string `json:"avatar_url,omitempty"`
	Gender              *string `json:"gender,omitempty"`
	Birthday            *string `json:"birthday,omitempty" example:"2000-01-15"`
	StudentStatus       *string `json:"student_status,omitempty"`
	MatricNo            *string `json:"matric_no,omitempty"`
	Department          *string `json:"department,omitempty" example:"Computer Science"`
	Faculty             *string `json:"faculty,omitempty"`
	PrimaryTrack        *string `json:"primary_track,omitempty"`
	SecondaryTrack      *string `json:"secondary_track,omitempty"`
	PrimarySkillLevel   *string `json:"primary_skill_level,omitempty"`
	SecondarySkillLevel *string `json:"secondary_skill_level,omitempty"`
	TOSAgreed           *bool   `json:"tos_agreed,omitempty" example:"true"`
	TOSAgreedAt         *string `json:"tos_agreed_at,omitempty"`
	TOSVersion          *string `json:"tos_version,omitempty" example:"1.0"`
}

type AdminUpdateRequest struct {
	Roles    []string `json:"roles,omitempty" example:"user,admin"`
	Teams    []string `json:"teams,omitempty" example:"core"`
	IsActive *bool    `json:"is_active,omitempty" example:"true"`

	// Profile fields an admin may correct on someone else's record.
	FullName            *string `json:"full_name,omitempty" example:"Jane Doe"`
	WhatsAppNumber      *string `json:"whatsapp_number,omitempty" example:"+2348012345678"`
	AvatarURL           *string `json:"avatar_url,omitempty"`
	Gender              *string `json:"gender,omitempty"`
	Birthday            *string `json:"birthday,omitempty" example:"2000-01-15"`
	StudentStatus       *string `json:"student_status,omitempty"`
	MatricNo            *string `json:"matric_no,omitempty" example:"20/1234"`
	Department          *string `json:"department,omitempty" example:"Computer Science"`
	Faculty             *string `json:"faculty,omitempty"`
	PrimaryTrack        *string `json:"primary_track,omitempty"`
	SecondaryTrack      *string `json:"secondary_track,omitempty"`
	PrimarySkillLevel   *string `json:"primary_skill_level,omitempty"`
	SecondarySkillLevel *string `json:"secondary_skill_level,omitempty"`
}

type AdminCreateRequest struct {
	Email    string   `json:"email" example:"jane.doe@babcock.edu.ng"`
	FullName string   `json:"full_name,omitempty" example:"Jane Doe"`
	Roles    []string `json:"roles,omitempty" example:"user"`
	Teams    []string `json:"teams,omitempty" example:"core"`
}
