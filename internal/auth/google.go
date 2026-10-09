package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
)

type GoogleProvider struct {
	config   oauth2.Config
	clientID string
}

func NewGoogleProvider(clientID, clientSecret, redirectURL string) *GoogleProvider {
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil
	}
	return &GoogleProvider{
		clientID: clientID,
		config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     google.Endpoint,
			Scopes:       []string{"openid", "email", "profile"},
		},
	}
}

func (p *GoogleProvider) AuthorizationURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *GoogleProvider) IdentityFromCode(ctx context.Context, code string) (domain.GoogleIdentity, error) {

	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return domain.GoogleIdentity{}, fmt.Errorf("exchange Google authorization code: %w", err)
	}
	raw, ok := token.Extra("id_token").(string)

	if !ok || raw == "" {
		return domain.GoogleIdentity{}, fmt.Errorf("google response did not include an ID token")
	}

	payload, err := idtoken.Validate(ctx, raw, p.clientID)

	if err != nil {
		return domain.GoogleIdentity{}, fmt.Errorf("validate Google ID token: %w", err)
	}

	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	verified, _ := payload.Claims["email_verified"].(bool)
	picture, _ := payload.Claims["picture"].(string)

	if payload.Subject == "" || !verified || strings.TrimSpace(email) == "" {
		return domain.GoogleIdentity{}, fmt.Errorf("google account must have a verified email")
	}

	return domain.GoogleIdentity{Subject: payload.Subject, Email: email, Name: name, Picture: picture}, nil
}
