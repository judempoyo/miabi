// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package acme

import (
	"crypto/ecdsa"
	"testing"

	"github.com/go-acme/lego/v4/registration"
)

func TestAccountKeyRoundTrip(t *testing.T) {
	key, err := GenerateAccountKey()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	pemStr, err := EncodeKey(key)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := DecodeKey(pemStr)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	a, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatal("generated key is not ECDSA")
	}
	b, ok := got.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatal("decoded key is not ECDSA")
	}
	if a.D.Cmp(b.D) != 0 {
		t.Error("round-tripped account key differs from the original")
	}
}

func TestRegistrationRoundTrip(t *testing.T) {
	reg := &registration.Resource{URI: "https://acme.example/acct/1"}
	s, err := EncodeRegistration(reg)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := DecodeRegistration(s)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got == nil || got.URI != reg.URI {
		t.Errorf("got %+v, want URI %q", got, reg.URI)
	}
	// Empty string decodes to nil (no account yet).
	if n, _ := DecodeRegistration(""); n != nil {
		t.Errorf("empty registration should decode to nil, got %+v", n)
	}
}
