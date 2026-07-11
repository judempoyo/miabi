// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import "testing"

func TestPostgresConnDiscreteFields(t *testing.T) {
	d := DatabaseConfig{host: "db", user: "miabi", password: "secret", name: "miabi", port: 5432, sslMode: "require"}
	host, port, name, user, pass, ssl := d.PostgresConn()
	if host != "db" || port != 5432 || name != "miabi" || user != "miabi" || pass != "secret" || ssl != "require" {
		t.Fatalf("discrete fields not resolved: %s %d %s %s %s %s", host, port, name, user, pass, ssl)
	}
}

func TestPostgresConnURLOverrides(t *testing.T) {
	d := DatabaseConfig{
		host: "fallback", port: 5432, name: "fallback", user: "fallback", password: "fallback", sslMode: "disable",
		url: "postgres://u:p@pghost:6543/appdb?sslmode=require",
	}
	host, port, name, user, pass, ssl := d.PostgresConn()
	if host != "pghost" || port != 6543 || name != "appdb" || user != "u" || pass != "p" || ssl != "require" {
		t.Fatalf("url not parsed: %s %d %s %s %s %s", host, port, name, user, pass, ssl)
	}
}

func TestPostgresConnBadURLFallsBack(t *testing.T) {
	d := DatabaseConfig{host: "db", port: 5432, name: "miabi", user: "miabi", password: "x", sslMode: "disable", url: "::not-a-url::"}
	host, port, _, _, _, _ := d.PostgresConn()
	if host != "db" || port != 5432 {
		t.Fatalf("bad url should fall back to discrete fields, got host=%s port=%d", host, port)
	}
}
