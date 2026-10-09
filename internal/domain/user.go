package domain

import (
	"time"

	"github.com/lib/pq"
)

type User struct {
	ID             string `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Email          string `gorm:"uniqueIndex;not null"`
	FullName       string
	PasswordHash   string
 	GoogleSubject  *string        `gorm:"uniqueIndex"` //yo bro i switched this to pointer inorder for sql to treat a null value in multiple table entries indifferently 
	Roles          pq.StringArray `gorm:"type:text[];not null"`
	Active         bool           `gorm:"column:is_active;default:true"`
	EmailVerified  bool           `gorm:"default:false"`
	ProfilePicture *string
	CreatedAt      time.Time `gorm:"default:now()"`
	UpdatedAt      time.Time
}

func (User) TableName() string {
	return "users"
}

type RefreshToken struct {
	ID        string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    string    `gorm:"type:uuid;not null;index"`
	User      User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"` 
	//yh bro, implemented this inorder for when a user is deleted their refresh token
	//is dropped from the Db as well
	TokenHash string    `gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"not null"`
	RevokedAt *time.Time
	CreatedAt time.Time
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
