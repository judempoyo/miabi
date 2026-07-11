// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import "testing"

// prodConfig is a minimal non-dev Config with every guarded field set to a safe,
// non-default value. Each test mutates one field to assert its guard fires.
func prodConfig() *Config {
	return &Config{
		Env:           "production",
		JWTSecret:     "a-real-secret",
		EncryptionKey: "a-real-key",
		AdminPassword: "a-real-password",
		CORSOrigins:   "https://panel.example.com",
	}
}

func TestValidateAcceptsSafeNonDevConfig(t *testing.T) {
	if err := prodConfig().validate(); err != nil {
		t.Fatalf("safe non-dev config rejected: %v", err)
	}
}

func TestValidateAllowsInsecureDefaultsInDev(t *testing.T) {
	c := &Config{
		Env:           "dev",
		JWTSecret:     defaultJWTSecret,
		EncryptionKey: "",
		AdminPassword: defaultAdminPassword,
		CORSOrigins:   "*",
	}
	if err := c.validate(); err != nil {
		t.Fatalf("dev config should permit defaults, got: %v", err)
	}
}

func TestValidateRejectsInsecureDefaultsInNonDev(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"default jwt secret", func(c *Config) { c.JWTSecret = defaultJWTSecret }},
		{"empty jwt secret", func(c *Config) { c.JWTSecret = "" }},
		{"empty encryption key", func(c *Config) { c.EncryptionKey = "" }},
		{"whitespace encryption key", func(c *Config) { c.EncryptionKey = "   " }},
		{"default admin password", func(c *Config) { c.AdminPassword = defaultAdminPassword }},
		{"empty admin password", func(c *Config) { c.AdminPassword = "" }},
		{"wildcard cors", func(c *Config) { c.CORSOrigins = "*" }},
		{"wildcard cors in list", func(c *Config) { c.CORSOrigins = "https://ok.example.com, *" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := prodConfig()
			tt.mutate(c)
			if err := c.validate(); err == nil {
				t.Fatalf("expected validate() to reject %q in non-dev, got nil", tt.name)
			}
		})
	}
}

func TestHasWildcardOrigin(t *testing.T) {
	cases := map[string]bool{
		"*":                            true,
		" * ":                          true,
		"https://a.com, *":             true,
		"https://a.com":                false,
		"https://a.com, https://b.com": false,
		"":                             false,
	}
	for in, want := range cases {
		if got := hasWildcardOrigin(in); got != want {
			t.Errorf("hasWildcardOrigin(%q) = %v, want %v", in, got, want)
		}
	}
}
