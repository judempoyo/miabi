// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package blob

import (
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/aws/smithy-go"
)

// apiErr fakes the shape the SDK reports a service error in.
type apiErr struct{ code string }

func (e apiErr) Error() string                 { return e.code }
func (e apiErr) ErrorCode() string             { return e.code }
func (e apiErr) ErrorMessage() string          { return e.code }
func (e apiErr) ErrorFault() smithy.ErrorFault { return smithy.FaultServer }

// The point of the probe is that a failure names the setting to fix. A 403 is
// not an answer; "the secret key does not match the access key" is.
func TestExplainNamesTheSettingToFix(t *testing.T) {
	cfg := Config{Bucket: "acme-backups", Region: "eu-central-1"}
	cases := []struct{ code, want string }{
		{"NoSuchBucket", "does not exist"},
		{"InvalidAccessKeyId", "access key is not recognized"},
		{"SignatureDoesNotMatch", "secret key does not match"},
		{"AccessDenied", "not allowed to write"},
		{"PermanentRedirect", "not in region"},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			got := explain(apiErr{code: tc.code}, cfg)
			if got == nil || !strings.Contains(got.Error(), tc.want) {
				t.Fatalf("explain(%s) = %v, want it to mention %q", tc.code, got, tc.want)
			}
		})
	}
}

// A self-hosted store addressed virtual-host style fails as a DNS error, which
// tells an operator nothing. It is also the single most common way a working
// MinIO looks broken, so the hint is worth more than the error.
func TestExplainSuggestsPathStyleOnDNSFailure(t *testing.T) {
	dnsErr := &net.DNSError{Err: "no such host", Name: "acme-backups.minio.internal"}
	cfg := Config{Bucket: "acme-backups", Endpoint: "minio.internal:9000"}

	got := explain(dnsErr, cfg)
	if got == nil || !strings.Contains(got.Error(), "path-style") {
		t.Fatalf("explain(dns) = %v, want a path-style hint", got)
	}
	if !strings.Contains(got.Error(), "minio.internal:9000") {
		t.Fatalf("the hint does not name the endpoint tried: %v", got)
	}

	// With path-style already on, the DNS failure is a genuine networking problem
	// and must not be explained away as a setting.
	cfg.ForcePathStyle = true
	got = explain(dnsErr, cfg)
	if strings.Contains(got.Error(), "path-style") {
		t.Fatalf("path-style was suggested when it is already enabled: %v", got)
	}
	if !errors.Is(got, dnsErr) {
		t.Fatalf("the underlying error was swallowed: %v", got)
	}
}

// An error the probe has no advice for is passed through unchanged rather than
// wrapped in a guess.
func TestExplainPassesThroughWhatItCannotImprove(t *testing.T) {
	base := errors.New("connection reset by peer")
	if got := explain(base, Config{}); !errors.Is(got, base) {
		t.Fatalf("explain rewrote an error it does not understand: %v", got)
	}
	if explain(nil, Config{}) != nil {
		t.Fatal("explain invented an error from nil")
	}
}
