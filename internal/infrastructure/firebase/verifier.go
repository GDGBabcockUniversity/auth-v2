package firebase

import (
	"context"
	"strings"

	firebase "firebase.google.com/go/v4"
	fireauth "firebase.google.com/go/v4/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/config"
	"google.golang.org/api/option"
)

type Verifier struct {
	Client *fireauth.Client
}

var _ auth.Verifier = Verifier{}

func New(ctx context.Context, c config.Config) (Verifier, error) {
	opts := []option.ClientOption{}
	opts = append(opts, option.WithCredentialsJSON([]byte(c.FirebaseServiceAccount)))
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: c.FirebaseProjectID}, opts...)
	if err != nil {
		return Verifier{}, err
	}
	client, err := app.Auth(ctx)
	return Verifier{Client: client}, err
}

func (v Verifier) Verify(ctx context.Context, raw string) (auth.Identity, error) {
	token, err := v.Client.VerifyIDToken(ctx, raw)
	if err != nil {
		return auth.Identity{}, err
	}
	get := func(key string) string {
		value, _ := token.Claims[key].(string)
		return strings.TrimSpace(value)
	}
	return auth.Identity{
		UID:   token.UID,
		Email: strings.ToLower(get("email")),
		Name:  get("name"), Picture: get("picture"),
		EmailVerified: token.Claims["email_verified"] == true,
	}, nil
}
