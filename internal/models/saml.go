// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "time"

// SAMLConfig is a SAML 2.0 identity-provider connection for an organization. The
// table is migrated in every build but is only read/written by the enterprise
// SAML handler (gated on the sso_saml entitlement); it stays empty in Community.
type SAMLConfig struct {
	ID             uint  `json:"id" gorm:"primaryKey"`
	OrganizationID *uint `json:"organization_id" gorm:"index"`
	// Name is the globally unique handle.
	Name string `json:"name" gorm:"uniqueIndex;not null"`
	// DisplayName is the free-text label shown in the UI.
	DisplayName string `json:"display_name"`

	// IdP metadata: a URL to fetch, or inline XML. One of the two is required.
	IDPMetadataURL string `json:"idp_metadata_url"`
	IDPMetadataXML string `json:"idp_metadata_xml" gorm:"type:text"`

	// SPEntityID overrides the service-provider entity ID (defaults to the SP
	// metadata URL when blank).
	SPEntityID string `json:"sp_entity_id"`

	// Attribute names mapping the assertion onto the user. Blank falls back to
	// the standard "email" / "displayName" (and the NameID for email).
	AttrEmail string `json:"attr_email"`
	AttrName  string `json:"attr_name"`

	Enabled   bool      `json:"enabled" gorm:"default:true;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
