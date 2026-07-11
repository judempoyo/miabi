// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package mwcatalog

import (
	"errors"
	"testing"

	"github.com/miabi-io/miabi/internal/services/crypto"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		mwType  string
		rule    map[string]any
		wantErr bool
	}{
		{"basicAuth valid", "basicAuth", map[string]any{"users": []any{map[string]any{"username": "u", "password": "p"}}}, false},
		{"basicAuth no users", "basicAuth", map[string]any{"realm": "x"}, true},
		{"basicAuth empty users", "basicAuth", map[string]any{"users": []any{}}, true},
		{"basicAuth user missing password", "basicAuth", map[string]any{"users": []any{map[string]any{"username": "u"}}}, true},
		{"rateLimit valid", "rateLimit", map[string]any{"unit": "minute", "requestsPerUnit": float64(100)}, false},
		{"rateLimit missing required", "rateLimit", map[string]any{"unit": "minute"}, true},
		{"rateLimit bad enum", "rateLimit", map[string]any{"unit": "decade", "requestsPerUnit": float64(1)}, true},
		{"rateLimit non-int", "rateLimit", map[string]any{"unit": "minute", "requestsPerUnit": "lots"}, true},
		{"rateLimit unknown key", "rateLimit", map[string]any{"unit": "minute", "requestsPerUnit": float64(1), "bogus": 1}, true},
		{"accessPolicy valid", "accessPolicy", map[string]any{"action": "DENY", "sourceRanges": []any{"10.0.0.0/8"}}, false},
		{"accessPolicy bad ranges type", "accessPolicy", map[string]any{"action": "DENY", "sourceRanges": "10.0.0.0/8"}, true},
		{"jwtAuth valid (algorithms)", "jwtAuth", map[string]any{"secret": "s", "algorithms": []any{"HS256"}}, false},
		{"jwtAuth algorithms wrong type", "jwtAuth", map[string]any{"secret": "s", "algorithms": "HS256"}, true},
		{"jwtAuth removed alg field", "jwtAuth", map[string]any{"secret": "s", "alg": "HS256"}, true},
		{"jwtAuth unknown key", "jwtAuth", map[string]any{"bogus": 1}, true},
		{"forwardAuth valid", "forwardAuth", map[string]any{"authUrl": "https://auth.example.com"}, false},
		{"forwardAuth missing required", "forwardAuth", map[string]any{"authSignIn": "https://x"}, true},
		{"forwardAuth rejects deprecated field", "forwardAuth", map[string]any{"authUrl": "https://a", "enableHostForwarding": true}, true},
		{"ldapAuth valid", "ldapAuth", map[string]any{"url": "ldap://x:389", "baseDN": "dc=x", "bindDN": "cn=a", "bindPass": "p", "userFilter": "(uid=%s)"}, false},
		{"ldapAuth missing bindPass", "ldapAuth", map[string]any{"url": "ldap://x:389", "baseDN": "dc=x", "bindDN": "cn=a", "userFilter": "(uid=%s)"}, true},
		{"redirectScheme valid", "redirectScheme", map[string]any{"scheme": "https", "permanent": true}, false},
		{"bodyLimit valid", "bodyLimit", map[string]any{"limit": "10MB"}, false},
		{"access valid", "access", map[string]any{"statusCode": float64(403)}, false},
		{"uncatalogued passes through", "customThing", map[string]any{"anything": "goes"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.mwType, tc.rule)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Validate(%s) = nil, want error", tc.mwType)
				}
				if !errors.Is(err, ErrInvalidRule) {
					t.Fatalf("Validate(%s) error = %v, want ErrInvalidRule", tc.mwType, err)
				}
			} else if err != nil {
				t.Fatalf("Validate(%s) = %v, want nil", tc.mwType, err)
			}
		})
	}
}

func TestSecretRoundTripAndRedaction(t *testing.T) {
	crypto.Init("test-master-key-for-mwcatalog") // enables encryption

	rule := map[string]any{
		"realm": "Restricted",
		"users": []any{map[string]any{"username": "jude", "password": "s3cret"}},
	}

	enc, err := EncryptSecrets("basicAuth", 5, rule)
	if err != nil {
		t.Fatal(err)
	}
	encPw := enc["users"].([]any)[0].(map[string]any)["password"].(string)
	if encPw == "s3cret" || !crypto.LooksEncrypted(encPw) {
		t.Fatalf("password not encrypted at rest: %q", encPw)
	}
	// Original rule must be untouched (transforms copy).
	if rule["users"].([]any)[0].(map[string]any)["password"] != "s3cret" {
		t.Fatal("EncryptSecrets mutated the input rule")
	}

	// Redaction hides the (encrypted) secret entirely.
	red := Redact("basicAuth", enc)
	if got := red["users"].([]any)[0].(map[string]any)["password"]; got != RedactedSentinel {
		t.Fatalf("redacted password = %v, want %q", got, RedactedSentinel)
	}

	// Decrypt-at-render restores the plaintext.
	dec, err := DecryptSecrets("basicAuth", enc)
	if err != nil {
		t.Fatal(err)
	}
	if got := dec["users"].([]any)[0].(map[string]any)["password"]; got != "s3cret" {
		t.Fatalf("decrypted password = %v, want s3cret", got)
	}
}

func TestTopLevelSecretField(t *testing.T) {
	crypto.Init("test-master-key-for-mwcatalog")
	rule := map[string]any{"url": "ldap://x:389", "bindPass": "topsecret"}

	enc, err := EncryptSecrets("ldapAuth", 7, rule)
	if err != nil {
		t.Fatal(err)
	}
	if bp := enc["bindPass"].(string); bp == "topsecret" || !crypto.LooksEncrypted(bp) {
		t.Fatalf("bindPass not encrypted: %q", bp)
	}
	if Redact("ldapAuth", enc)["bindPass"] != RedactedSentinel {
		t.Fatal("bindPass not redacted")
	}
	dec, _ := DecryptSecrets("ldapAuth", enc)
	if dec["bindPass"] != "topsecret" {
		t.Fatalf("bindPass decrypt = %v, want topsecret", dec["bindPass"])
	}
	// Non-secret field is left intact.
	if dec["url"] != "ldap://x:389" {
		t.Fatal("non-secret field altered")
	}
}

func TestMergeKeptSecrets(t *testing.T) {
	crypto.Init("test-master-key-for-mwcatalog")
	existing, _ := EncryptSecrets("basicAuth", 5, map[string]any{
		"users": []any{map[string]any{"username": "jude", "password": "keepme"}},
	})
	// Client edited the realm but left the password redacted.
	incoming := map[string]any{
		"realm": "New",
		"users": []any{map[string]any{"username": "jude", "password": RedactedSentinel}},
	}
	merged := MergeKeptSecrets("basicAuth", incoming, existing)
	mergedPw := merged["users"].([]any)[0].(map[string]any)["password"].(string)
	existingPw := existing["users"].([]any)[0].(map[string]any)["password"].(string)
	if mergedPw != existingPw {
		t.Fatalf("merged password = %q, want kept %q", mergedPw, existingPw)
	}
	// And it still decrypts to the original plaintext.
	dec, _ := DecryptSecrets("basicAuth", merged)
	if got := dec["users"].([]any)[0].(map[string]any)["password"]; got != "keepme" {
		t.Fatalf("kept password decrypts to %v, want keepme", got)
	}
}
