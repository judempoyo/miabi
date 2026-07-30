// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package registryserver

import (
	"testing"

	"github.com/miabi-io/miabi/internal/config"
	"github.com/miabi-io/miabi/internal/models"
)

// baseDomain is a settings reader returning a fixed external base domain.
type baseDomain string

func (b baseDomain) String(_, def string) string {
	if b == "" {
		return def
	}
	return string(b)
}

// Enablement follows MIABI_REGISTRY_ENABLED and nothing else — in particular not
// the stored row, which is what the admin UI used to write.
func TestEnabledIsEnvOnly(t *testing.T) {
	cases := []struct {
		name   string
		env    bool
		stored bool
		want   bool
	}{
		{"env on", true, false, true},
		{"env off beats a stored on", false, true, false},
		{"env off", false, false, false},
		{"env on with stored on", true, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := &models.RegistrySettings{Enabled: tc.stored}
			svc := &Service{cfg: config.RegistryConfig{Enabled: tc.env}}
			svc.applyEnvConfig(st)
			if st.Enabled != tc.want {
				t.Errorf("Enabled = %v, want %v", st.Enabled, tc.want)
			}
		})
	}
}

func TestHostResolution(t *testing.T) {
	cases := []struct {
		name       string
		envHost    string
		storedHost string
		base       string
		wantHost   string
		wantSource string
	}{
		{"env wins", "registry.env.test", "registry.old.test", "example.com", "registry.env.test", "env"},
		{
			// An install upgraded from when the host was UI-editable must keep
			// answering on the name its existing images already reference.
			"stored value survives when env is unset",
			"", "registry.old.test", "example.com", "registry.old.test", "stored",
		},
		{"base domain when nothing is set", "", "", "example.com", "registry.example.com", "base_domain"},
		{"nothing at all", "", "", "", "", "unset"},
		// A host that matches no image reference must resolve to "" — distribution
		// then reports itself unavailable instead of running with an inert check.
		{"invalid env host is dropped", "https://registry.env.test", "", "example.com", "registry.example.com", "base_domain"},
		{"invalid stored host is dropped", "", "reg istry", "example.com", "registry.example.com", "base_domain"},
		{"invalid base domain yields no host", "", "", "not a domain", "", "unset"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &Service{cfg: config.RegistryConfig{Host: tc.envHost}, settings: baseDomain(tc.base)}
			st := &models.RegistrySettings{Host: tc.storedHost}
			svc.applyEnvConfig(st)
			if got := svc.HostFor(st); got != tc.wantHost {
				t.Errorf("HostFor = %q, want %q", got, tc.wantHost)
			}
			if got := svc.HostSource(st); got != tc.wantSource {
				t.Errorf("HostSource = %q, want %q", got, tc.wantSource)
			}
		})
	}
}

// Save must not be a back door onto the two locked fields: whatever the row held,
// what comes back is what the environment says.
func TestSaveCannotChangeEnablement(t *testing.T) {
	svc := &Service{cfg: config.RegistryConfig{Enabled: false, Host: "registry.env.test"}, settings: baseDomain("")}
	st := &models.RegistrySettings{Enabled: true, Host: "registry.attacker.test"}
	svc.applyEnvConfig(st)
	if st.Enabled {
		t.Error("a stored enabled=true survived an env that says disabled")
	}
	if st.Host != "registry.env.test" {
		t.Errorf("Host = %q, want the env value to win", st.Host)
	}
}
