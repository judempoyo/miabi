// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package certificate

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/miabi-io/miabi/internal/models"
)

func TestHostMatches(t *testing.T) {
	cases := []struct {
		sans []string
		host string
		want bool
	}{
		{[]string{"app.example.com"}, "app.example.com", true},
		{[]string{"app.example.com"}, "other.example.com", false},
		{[]string{"*.example.com"}, "app.example.com", true},
		{[]string{"*.example.com"}, "a.b.example.com", false}, // only one label
		{[]string{"*.example.com"}, "example.com", false},     // bare apex not covered
		{[]string{"*.example.com"}, "APP.example.com", true},  // case-insensitive (caller lowercases)
		{[]string{"a.com", "*.b.com"}, "x.b.com", true},
		{nil, "app.example.com", false},
		{[]string{"app.example.com"}, "", false},
	}
	for _, c := range cases {
		if got := hostMatches(c.sans, c.host); got != c.want {
			t.Errorf("hostMatches(%v, %q) = %v, want %v", c.sans, c.host, got, c.want)
		}
	}
}

// fakeDomains implements DomainLister for the import-gate tests.
type fakeDomains struct{ list []models.Domain }

func (f fakeDomains) ListByWorkspace(uint) ([]models.Domain, error) { return f.list, nil }

func TestValidateAgainstDomains(t *testing.T) {
	meta := &certMeta{commonName: "example.com", dnsNames: []string{"example.com", "*.example.com"}}

	// No domain registered → blocked.
	s := &Service{domains: fakeDomains{}}
	if err := s.validateAgainstDomains(1, meta); !errors.Is(err, ErrNoDomains) {
		t.Errorf("no domains: got %v, want ErrNoDomains", err)
	}

	// Cert names all covered by a registered domain → ok.
	s = &Service{domains: fakeDomains{list: []models.Domain{{Name: "example.com"}}}}
	if err := s.validateAgainstDomains(1, meta); err != nil {
		t.Errorf("covered cert should pass, got %v", err)
	}

	// A SAN for a domain the workspace does not control → blocked.
	other := &certMeta{commonName: "example.com", dnsNames: []string{"example.com", "evil.org"}}
	if err := s.validateAgainstDomains(1, other); !errors.Is(err, ErrDomainMismatch) {
		t.Errorf("uncovered SAN: got %v, want ErrDomainMismatch", err)
	}

	// Nil lister (unwired/tests) skips the checks.
	if err := (&Service{}).validateAgainstDomains(1, other); err != nil {
		t.Errorf("nil domain lister should skip, got %v", err)
	}
}

func TestNameUnderDomain(t *testing.T) {
	domains := []models.Domain{{Name: "Example.com"}}
	for _, c := range []struct {
		name string
		want bool
	}{
		{"example.com", true},
		{"app.example.com", true},
		{"*.example.com", true},
		{"example.org", false},
		{"notexample.com", false},
	} {
		if got := nameUnderDomain(c.name, domains); got != c.want {
			t.Errorf("nameUnderDomain(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestParse(t *testing.T) {
	certPEM, keyPEM := genCert(t, "example.com", []string{"example.com", "*.example.com"})

	meta, err := parse(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("parse valid cert: %v", err)
	}
	if meta.commonName != "example.com" {
		t.Errorf("commonName = %q, want example.com", meta.commonName)
	}
	if len(meta.dnsNames) != 2 {
		t.Errorf("dnsNames = %v, want 2 SANs", meta.dnsNames)
	}
	if meta.notAfter.Before(time.Now()) {
		t.Errorf("notAfter %v is in the past", meta.notAfter)
	}

	// Missing material.
	if _, err := parse("", keyPEM); !errors.Is(err, ErrPEMRequired) {
		t.Errorf("parse empty cert: got %v, want ErrPEMRequired", err)
	}

	// Mismatched key (a second, unrelated key).
	_, otherKey := genCert(t, "other.com", nil)
	if _, err := parse(certPEM, otherKey); !errors.Is(err, ErrInvalidPEM) {
		t.Errorf("parse mismatched key: got %v, want ErrInvalidPEM", err)
	}
}

// genCert returns a self-signed cert + key PEM for testing.
func genCert(t *testing.T, cn string, sans []string) (certPEM, keyPEM string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		DNSNames:     sans,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
	return certPEM, keyPEM
}
