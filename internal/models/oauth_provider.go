// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "time"

// OAuthProviderType is the kind of identity provider.
type OAuthProviderType string

const (
	OAuthProviderGoogle OAuthProviderType = "google" // Google OAuth 2.0 (well-known discovery)
	OAuthProviderOIDC   OAuthProviderType = "oidc"   // generic OpenID Connect
)

// OAuthProvider is a platform-level SSO identity provider. Secrets are stored
// encrypted at rest (see internal/services/crypto) and never serialized.
type OAuthProvider struct {
	ID uint `json:"id" gorm:"primaryKey"`
	// Name is the globally unique handle used in callback routes.
	Name string `json:"name" gorm:"uniqueIndex;not null"`
	// DisplayName is the free-text label shown on login buttons.
	DisplayName string            `json:"display_name"`
	Type        OAuthProviderType `json:"type" gorm:"not null"`

	ClientID        string `json:"-" gorm:"not null"`
	ClientSecretEnc string `json:"-" gorm:"column:client_secret;not null"`

	// OIDC endpoints. For "google" these are derived from the well-known config.
	Issuer      string `json:"issuer"`
	AuthURL     string `json:"auth_url"`
	TokenURL    string `json:"token_url"`
	UserInfoURL string `json:"userinfo_url"`

	Scopes string `json:"scopes" gorm:"default:'openid email profile'"`

	Enabled      bool `json:"enabled" gorm:"default:true;not null"`
	Hidden       bool `json:"hidden" gorm:"default:false;not null"`       // hide from login buttons
	AutoRegister bool `json:"auto_register" gorm:"default:true;not null"` // auto-create users on first login

	// AllowedDomains is a CSV of email domains permitted to sign in / register
	// through this provider. Empty means any domain is allowed.
	AllowedDomains string `json:"allowed_domains"`

	// OrganizationID is the realm this provider belongs to (nullable → default org).
	OrganizationID *uint `json:"organization_id" gorm:"index"`

	// EmailClaim / NameClaim map a generic OIDC userinfo claim onto the user's
	// email / display name. Empty falls back to the standard "email" / "name".
	EmailClaim string `json:"email_claim"`
	NameClaim  string `json:"name_claim"`

	// DefaultWorkspaceID + DefaultRole auto-join a newly registered SSO user to a
	// workspace with a role on first login. Nil = no auto-join (user is invited
	// separately).
	DefaultWorkspaceID *uint         `json:"default_workspace_id" gorm:"index"`
	DefaultRole        WorkspaceRole `json:"default_role"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
