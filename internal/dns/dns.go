// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package dns abstracts a managed DNS host behind one small Provider interface,
// backed by libdns modules (Cloudflare, Route 53, DigitalOcean). It mirrors the
// blob.Store pattern: adding a host later is a new case in Build, not a new
// client. Miabi uses it to manage only the records it owns (ownership TXT, app
// A/AAAA) — never a user's other records.
package dns

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/libdns/cloudflare"
	"github.com/libdns/digitalocean"
	"github.com/libdns/libdns"
	"github.com/libdns/route53"
	"github.com/miabi-io/miabi/internal/models"
)

// Record is Miabi's provider-agnostic view of a DNS record. Name is a FQDN (the
// adapter relativizes it to the zone); Value is the record data (TXT text, A/AAAA
// IP, CNAME target).
type Record struct {
	Type  string        `json:"type"`  // TXT | A | AAAA | CNAME
	Name  string        `json:"name"`  // FQDN, e.g. _miabi-challenge.example.com
	Value string        `json:"value"` // record data
	TTL   time.Duration `json:"-"`     // 0 = provider default
}

// Credentials is the union of fields across provider types, parsed from the
// decrypted JSON blob. Only the fields relevant to a given Type are set.
type Credentials struct {
	APIToken        string `json:"api_token,omitempty"`         // cloudflare, digitalocean
	AccessKeyID     string `json:"access_key_id,omitempty"`     // route53
	SecretAccessKey string `json:"secret_access_key,omitempty"` // route53
	Region          string `json:"region,omitempty"`            // route53
}

// Provider is Miabi's view of a DNS host. Implementations are idempotent and
// safe for concurrent use (libdns guarantees the latter).
type Provider interface {
	// GetRecords lists the records in a zone (used by test + conflict checks).
	GetRecords(ctx context.Context, zone string) ([]Record, error)
	// SetRecord upserts a record (creates or replaces the RRset for name+type).
	SetRecord(ctx context.Context, zone string, rec Record) error
	// DeleteRecord removes a record.
	DeleteRecord(ctx context.Context, zone string, rec Record) error
	// Test validates the credentials against a zone (a successful GetRecords).
	Test(ctx context.Context, zone string) error
}

// zoneClient is the libdns surface the adapter needs; every wired module
// satisfies it.
type zoneClient interface {
	libdns.RecordGetter
	libdns.RecordSetter
	libdns.RecordDeleter
}

// Build returns a Provider for a connection type + its (already-decrypted)
// credentials. Unknown types return an error.
func Build(providerType string, creds Credentials) (Provider, error) {
	switch providerType {
	case models.DNSProviderCloudflare:
		if creds.APIToken == "" {
			return nil, fmt.Errorf("cloudflare: api_token is required")
		}
		return &adapter{z: &cloudflare.Provider{APIToken: creds.APIToken}}, nil
	case models.DNSProviderDigitalOcean:
		if creds.APIToken == "" {
			return nil, fmt.Errorf("digitalocean: api_token is required")
		}
		return &adapter{z: &digitalocean.Provider{APIToken: creds.APIToken}}, nil
	case models.DNSProviderRoute53:
		if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
			return nil, fmt.Errorf("route53: access_key_id and secret_access_key are required")
		}
		return &adapter{z: &route53.Provider{
			AccessKeyId: creds.AccessKeyID, SecretAccessKey: creds.SecretAccessKey, Region: creds.Region,
		}}, nil
	default:
		return nil, fmt.Errorf("unknown DNS provider type %q", providerType)
	}
}

// adapter maps the Miabi Provider interface onto a libdns zoneClient.
type adapter struct{ z zoneClient }

// canonicalZone gives the zone a trailing dot, libdns's canonical FQDN form.
func canonicalZone(zone string) string {
	zone = strings.TrimSpace(zone)
	if zone == "" {
		return "."
	}
	if !strings.HasSuffix(zone, ".") {
		zone += "."
	}
	return zone
}

func (a *adapter) GetRecords(ctx context.Context, zone string) ([]Record, error) {
	recs, err := a.z.GetRecords(ctx, canonicalZone(zone))
	if err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(recs))
	for _, r := range recs {
		rr := r.RR()
		out = append(out, Record{Type: rr.Type, Name: rr.Name, Value: rr.Data, TTL: rr.TTL})
	}
	return out, nil
}

func (a *adapter) SetRecord(ctx context.Context, zone string, rec Record) error {
	cz := canonicalZone(zone)
	_, err := a.z.SetRecords(ctx, cz, []libdns.Record{a.toRR(cz, rec)})
	return err
}

func (a *adapter) DeleteRecord(ctx context.Context, zone string, rec Record) error {
	cz := canonicalZone(zone)
	_, err := a.z.DeleteRecords(ctx, cz, []libdns.Record{a.toRR(cz, rec)})
	return err
}

func (a *adapter) Test(ctx context.Context, zone string) error {
	_, err := a.z.GetRecords(ctx, canonicalZone(zone))
	return err
}

// toRR builds a libdns record relative to the zone. RR implements libdns.Record;
// providers accept it for set/delete (only the specific RR-types are required on
// the *return* path, which we don't use here).
func (a *adapter) toRR(zone string, rec Record) libdns.RR {
	return libdns.RR{
		Type: rec.Type,
		Name: libdns.RelativeName(strings.TrimSuffix(rec.Name, "."), zone),
		Data: rec.Value,
		TTL:  rec.TTL,
	}
}
