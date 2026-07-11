// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package twofactor

import (
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestGenerateReturnsSecretAndOtpauthURL(t *testing.T) {
	secret, url, err := Generate("Miabi", "user@example.com")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if secret == "" {
		t.Fatal("expected a non-empty secret")
	}
	if !strings.HasPrefix(url, "otpauth://totp/") {
		t.Fatalf("expected otpauth URL, got %q", url)
	}
	if !strings.Contains(url, "issuer=Miabi") {
		t.Fatalf("expected issuer in URL, got %q", url)
	}
}

func TestValidate(t *testing.T) {
	secret, _, err := Generate("Miabi", "user@example.com")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	if !Validate(secret, code) {
		t.Error("expected the current code to validate")
	}
	if Validate(secret, "000000") {
		t.Error("expected an obviously wrong code to fail")
	}
	if Validate("", code) {
		t.Error("expected validation against an empty secret to fail")
	}
}

func TestQRDataURI(t *testing.T) {
	_, url, err := Generate("Miabi", "user@example.com")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	uri, err := QRDataURI(url, 200)
	if err != nil {
		t.Fatalf("QRDataURI: %v", err)
	}
	if !strings.HasPrefix(uri, "data:image/png;base64,") {
		t.Fatal("expected a PNG data URI")
	}
}
