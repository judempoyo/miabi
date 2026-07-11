//go:build enterprise

// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: LicenseRef-Miabi-Enterprise

package ldap

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestAuthenticate_Integration binds against a real directory when one is
// configured via env, so a CI job running an OpenLDAP/Samba-AD container (or any
// reachable directory) exercises the full dial→search→bind→group path. It skips
// when MIABI_LDAP_IT_HOST is unset, so the default `go test` stays hermetic.
//
// Example (OpenLDAP osixia container seeded with a user):
//
//	MIABI_LDAP_IT_HOST=localhost MIABI_LDAP_IT_PORT=389 \
//	MIABI_LDAP_IT_TLS=none \
//	MIABI_LDAP_IT_BINDDN='cn=admin,dc=example,dc=org' MIABI_LDAP_IT_BINDPW=admin \
//	MIABI_LDAP_IT_BASEDN='dc=example,dc=org' MIABI_LDAP_IT_FILTER='(uid=%s)' \
//	MIABI_LDAP_IT_USER=alice MIABI_LDAP_IT_PASS=secret \
//	go test -tags enterprise ./internal/enterprise/sso/ldap/ -run Integration -v
func TestAuthenticate_Integration(t *testing.T) {
	host := os.Getenv("MIABI_LDAP_IT_HOST")
	if host == "" {
		t.Skip("set MIABI_LDAP_IT_HOST to run the LDAP integration test")
	}
	port, _ := strconv.Atoi(getenv("MIABI_LDAP_IT_PORT", "389"))
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.LDAPConfig{}); err != nil {
		t.Fatal(err)
	}
	cfg := &models.LDAPConfig{
		Name:            "it",
		Host:            host,
		Port:            port,
		TLSMode:         getenv("MIABI_LDAP_IT_TLS", "none"),
		InsecureSkipTLS: os.Getenv("MIABI_LDAP_IT_INSECURE") == "1",
		BindDN:          os.Getenv("MIABI_LDAP_IT_BINDDN"),
		BindPasswordEnc: os.Getenv("MIABI_LDAP_IT_BINDPW"), // Decrypt defaults to passthrough
		UserBaseDN:      os.Getenv("MIABI_LDAP_IT_BASEDN"),
		UserFilter:      getenv("MIABI_LDAP_IT_FILTER", "(uid=%s)"),
		Enabled:         true,
	}
	if err := db.Create(cfg).Error; err != nil {
		t.Fatal(err)
	}
	p, err := New(Deps{DB: db})
	if err != nil {
		t.Fatal(err)
	}

	user := getenv("MIABI_LDAP_IT_USER", "alice")
	// Wrong password must fail.
	if _, err := p.Authenticate(context.Background(), user, "definitely-wrong"); err == nil {
		t.Error("expected a bind failure for a wrong password")
	}
	// Correct password must resolve an identity.
	ident, err := p.Authenticate(context.Background(), user, os.Getenv("MIABI_LDAP_IT_PASS"))
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if ident.Username == "" {
		t.Error("expected a resolved username")
	}
	t.Logf("resolved identity: user=%q email=%q groups=%d", ident.Username, ident.Email, len(ident.Groups))
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
