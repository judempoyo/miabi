// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package mwcatalog is the single source of truth for the curated Goma
// middleware types Miabi exposes as security policies. Each supported Goma type
// has one declarative Descriptor that drives validation, secret handling and the
// UI form. Adding a type later is one descriptor; an uncatalogued type is the
// "advanced" escape hatch — it is passed through to Goma without schema checks.
//
// Field schemas mirror the Goma rule structs in
// github.com/jkaninda/goma-gateway internal/types.go.
package mwcatalog

// Category groups middleware types for the UI.
type Category string

const (
	CategoryAccess        Category = "access"
	CategorySecurity      Category = "security"
	CategoryTraffic       Category = "traffic"
	CategoryTransform     Category = "transform"
	CategoryObservability Category = "observability"
)

// Field types understood by validation and the form renderer.
const (
	FieldString   = "string"
	FieldInt      = "int"
	FieldBool     = "bool"
	FieldStrings  = "string[]"
	FieldDuration = "duration" // a Go duration string, e.g. "10m"
	FieldEnum     = "enum"     // one of Options
	FieldUsers    = "users"    // basicAuth users: [{username, password}]
	FieldObject   = "object"   // free-form map, passed through
)

// Field is one key of a middleware's rule.
type Field struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`
	Required bool     `json:"required,omitempty"`
	Secret   bool     `json:"secret,omitempty"` // encrypted at rest, redacted in responses
	Default  any      `json:"default,omitempty"`
	Options  []string `json:"options,omitempty"` // for enum
	Help     string   `json:"help,omitempty"`
}

// Descriptor declares one curated Goma middleware type.
type Descriptor struct {
	Type        string   `json:"type"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Category    Category `json:"category"`
	Fields      []Field  `json:"fields"`
}

// secretFields returns the descriptor's fields marked Secret.
func (d Descriptor) secretFields() []Field {
	var out []Field
	for _, f := range d.Fields {
		if f.Secret {
			out = append(out, f)
		}
	}
	return out
}

// registry is the ordered set of supported descriptors. Order is the display
// order in the UI catalog.
var registry = []Descriptor{
	{
		Type:        "basicAuth",
		DisplayName: "Basic authentication",
		Description: "Require a username and password (HTTP Basic) to reach the route.",
		Category:    CategoryAccess,
		Fields: []Field{
			{Key: "users", Label: "Users", Type: FieldUsers, Required: true, Secret: true, Help: "One or more username/password pairs."},
			{Key: "realm", Label: "Realm", Type: FieldString, Default: "Restricted", Help: "Shown by the browser's auth prompt."},
			{Key: "forwardUsername", Label: "Forward username to backend", Type: FieldBool},
		},
	},
	{
		Type:        "jwtAuth",
		DisplayName: "JWT authentication",
		Description: "Require a valid JSON Web Token. Verify with a shared secret (HS*), a public key, or a JWKS endpoint.",
		Category:    CategoryAccess,
		Fields: []Field{
			{Key: "secret", Label: "Signing secret", Type: FieldString, Secret: true, Help: "Shared secret for HMAC algorithms (HS256/384/512)."},
			{Key: "publicKey", Label: "Public key", Type: FieldString, Help: "PEM public key for asymmetric algorithms (RS*/ES*)."},
			{Key: "jwksUrl", Label: "JWKS URL", Type: FieldString, Help: "Endpoint serving the signing keys."},
			{Key: "jwksFile", Label: "JWKS file", Type: FieldString, Help: "Path to a local JWKS file."},
			{Key: "algorithms", Label: "Algorithms", Type: FieldStrings, Help: "Accepted signing algorithms, e.g. RS256, ES256. Defaults to a safe set for the configured key type."},
			{Key: "issuer", Label: "Issuer", Type: FieldString, Help: "Required iss claim."},
			{Key: "audience", Label: "Audience", Type: FieldString, Help: "Required aud claim."},
			{Key: "claimsExpression", Label: "Claims expression", Type: FieldString, Help: "Expression the token claims must satisfy."},
			{Key: "forwardAuthorization", Label: "Forward Authorization header", Type: FieldBool},
			{Key: "forwardHeaders", Label: "Forward claim headers", Type: FieldObject, Help: "Advanced: map of header name → claim path."},
		},
	},
	{
		Type:        "forwardAuth",
		DisplayName: "Forward authentication",
		Description: "Delegate authentication to an external service, like Authelia or oauth2-proxy.",
		Category:    CategoryAccess,
		Fields: []Field{
			{Key: "authUrl", Label: "Auth URL", Type: FieldString, Required: true, Help: "Service that authorizes each request (2xx = allow)."},
			{Key: "authSignIn", Label: "Sign-in URL", Type: FieldString, Help: "Where to redirect unauthenticated users."},
			{Key: "forwardHostHeaders", Label: "Forward host headers", Type: FieldBool},
			{Key: "insecureSkipVerify", Label: "Skip TLS verification", Type: FieldBool, Help: "Don't verify the auth service's TLS certificate."},
			{Key: "authRequestHeaders", Label: "Auth request headers", Type: FieldStrings, Help: "Headers copied from the request to the auth service."},
			{Key: "authResponseHeaders", Label: "Auth response headers", Type: FieldStrings, Help: "Headers copied from the auth response to the backend."},
			{Key: "authResponseHeadersAsParams", Label: "Auth response headers as params", Type: FieldStrings},
			{Key: "addAuthCookiesToResponse", Label: "Add auth cookies to response", Type: FieldStrings},
		},
	},
	{
		Type:        "ldapAuth",
		DisplayName: "LDAP authentication",
		Description: "Authenticate users against an LDAP / Active Directory directory.",
		Category:    CategoryAccess,
		Fields: []Field{
			{Key: "url", Label: "Server URL", Type: FieldString, Required: true, Help: "e.g. ldap://ldap.example.com:389."},
			{Key: "baseDN", Label: "Base DN", Type: FieldString, Required: true, Help: "e.g. ou=users,dc=example,dc=com."},
			{Key: "bindDN", Label: "Bind DN", Type: FieldString, Required: true, Help: "DN used to bind for user lookups."},
			{Key: "bindPass", Label: "Bind password", Type: FieldString, Required: true, Secret: true},
			{Key: "userFilter", Label: "User filter", Type: FieldString, Required: true, Help: "e.g. (uid=%s)."},
			{Key: "realm", Label: "Realm", Type: FieldString, Help: "Shown by the browser's auth prompt."},
			{Key: "forwardUsername", Label: "Forward username to backend", Type: FieldBool},
			{Key: "startTLS", Label: "StartTLS", Type: FieldBool, Help: "Upgrade the connection to TLS."},
			{Key: "insecureSkipVerify", Label: "Skip TLS verification", Type: FieldBool},
			{Key: "connPool", Label: "Connection pool", Type: FieldObject, Help: "Advanced: {size, burst, ttl}."},
		},
	},
	{
		Type:        "access",
		DisplayName: "Block access",
		Description: "Deny requests to the matched paths with a fixed status code.",
		Category:    CategorySecurity,
		Fields: []Field{
			{Key: "statusCode", Label: "Status code", Type: FieldInt, Default: 403, Help: "HTTP status returned for blocked requests (default 403)."},
		},
	},
	{
		Type:        "accessPolicy",
		DisplayName: "IP access policy",
		Description: "Allow or deny requests by client IP / CIDR range.",
		Category:    CategorySecurity,
		Fields: []Field{
			{Key: "action", Label: "Action", Type: FieldEnum, Required: true, Options: []string{"ALLOW", "DENY"}, Default: "ALLOW"},
			{Key: "sourceRanges", Label: "Source ranges", Type: FieldStrings, Required: true, Help: "IPs or CIDRs, e.g. 10.0.0.0/8, 203.0.113.5."},
		},
	},
	{
		Type:        "bodyLimit",
		DisplayName: "Request body limit",
		Description: "Reject requests whose body exceeds a size limit.",
		Category:    CategorySecurity,
		Fields: []Field{
			{Key: "limit", Label: "Limit", Type: FieldString, Required: true, Help: "Max body size, e.g. 10MB, 512KB."},
		},
	},
	{
		Type:        "rateLimit",
		DisplayName: "Rate limit",
		Description: "Throttle requests per client over a time unit.",
		Category:    CategoryTraffic,
		Fields: []Field{
			{Key: "unit", Label: "Per", Type: FieldEnum, Required: true, Options: []string{"second", "minute", "hour"}, Default: "minute"},
			{Key: "requestsPerUnit", Label: "Requests per unit", Type: FieldInt, Required: true, Default: 100},
			{Key: "burst", Label: "Burst", Type: FieldInt, Help: "Extra requests allowed in a short spike."},
			{Key: "banAfter", Label: "Ban after", Type: FieldInt, Help: "Ban a client after this many rejected requests."},
			{Key: "banDuration", Label: "Ban duration", Type: FieldDuration, Default: "10m", Help: "How long a banned client stays blocked, e.g. 10m."},
			{Key: "keyStrategy", Label: "Key strategy", Type: FieldObject, Help: "Advanced: how clients are identified (source/name)."},
		},
	},
	{
		Type:        "redirectScheme",
		DisplayName: "Force scheme (HTTPS)",
		Description: "Redirect requests to a different scheme — typically http→https.",
		Category:    CategoryTransform,
		Fields: []Field{
			{Key: "scheme", Label: "Scheme", Type: FieldEnum, Required: true, Options: []string{"https", "http"}, Default: "https"},
			{Key: "port", Label: "Port", Type: FieldInt, Help: "Optional target port (e.g. 443)."},
			{Key: "permanent", Label: "Permanent (301)", Type: FieldBool, Help: "Use 301 instead of 302."},
		},
	},
}

var byType = func() map[string]Descriptor {
	m := make(map[string]Descriptor, len(registry))
	for _, d := range registry {
		m[d.Type] = d
	}
	return m
}()

// Get returns the descriptor for a Goma middleware type, if catalogued.
func Get(t string) (Descriptor, bool) {
	d, ok := byType[t]
	return d, ok
}

// All returns every catalogued descriptor in display order.
func All() []Descriptor {
	out := make([]Descriptor, len(registry))
	copy(out, registry)
	return out
}
