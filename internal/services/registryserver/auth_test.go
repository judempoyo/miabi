// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package registryserver

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/miabi-io/miabi/internal/config"
	"github.com/miabi-io/miabi/internal/models"
)

type fakeKeys struct {
	key *models.APIKey
	err error
}

func (f fakeKeys) Verify(string) (*models.APIKey, error) { return f.key, f.err }

type fakeWS struct {
	byID    map[uint]*models.Workspace
	byName  map[string]*models.Workspace
	members map[string]*models.WorkspaceMember // "<wsID>:<userID>"
}

func (f fakeWS) FindByID(id uint) (*models.Workspace, error) {
	if w, ok := f.byID[id]; ok {
		return w, nil
	}
	return nil, errors.New("not found")
}

func (f fakeWS) FindByName(name string) (*models.Workspace, error) {
	if w, ok := f.byName[name]; ok {
		return w, nil
	}
	return nil, errors.New("not found")
}

func (f fakeWS) FindMember(workspaceID, userID uint) (*models.WorkspaceMember, error) {
	if m, ok := f.members[fmt.Sprintf("%d:%d", workspaceID, userID)]; ok {
		return m, nil
	}
	return nil, errors.New("not a member")
}

// fixture: acme (id 7), other (id 8); user 42 is a developer in acme, user 99 a
// viewer in acme. Nobody is a member of other.
func wsFixture() fakeWS {
	acme := &models.Workspace{ID: 7, Name: "acme"}
	other := &models.Workspace{ID: 8, Name: "other"}
	return fakeWS{
		byID:   map[uint]*models.Workspace{7: acme, 8: other},
		byName: map[string]*models.Workspace{"acme": acme, "other": other},
		members: map[string]*models.WorkspaceMember{
			"7:42": {WorkspaceID: 7, UserID: 42, Role: models.WorkspaceRoleDeveloper},
			"7:99": {WorkspaceID: 7, UserID: 99, Role: models.WorkspaceRoleViewer},
		},
	}
}

// wsToken builds a service whose token is scoped to workspace acme (id 7).
func wsToken(scopes []string) *Service {
	id := uint(7)
	return &Service{keys: fakeKeys{key: &models.APIKey{WorkspaceID: &id, UserID: 1, Scopes: scopes}}, ws: wsFixture()}
}

// userToken builds a service whose token is account-wide, owned by userID.
func userToken(userID uint, scopes []string) *Service {
	return &Service{keys: fakeKeys{key: &models.APIKey{UserID: userID, Scopes: scopes}}, ws: wsFixture()}
}

func basic(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

func TestAuthorize(t *testing.T) {
	rw := []string{models.ScopeRead, models.ScopeWrite}

	cases := []struct {
		name       string
		svc        *Service
		in         AuthInput
		wantStatus int
		challenge  bool
	}{
		{"no credentials challenges", wsToken(rw), AuthInput{Method: "GET", URI: "/v2/"}, http.StatusUnauthorized, true},
		{"malformed basic challenges", wsToken(rw), AuthInput{Authorization: "Basic !!!", Method: "GET", URI: "/v2/"}, http.StatusUnauthorized, true},
		{
			"invalid token challenges",
			&Service{keys: fakeKeys{err: errors.New("nope")}, ws: wsFixture()},
			AuthInput{Authorization: basic("acme", "bad"), Method: "GET", URI: "/v2/acme/web/manifests/1"},
			http.StatusUnauthorized, true,
		},
		{"base v2 allowed for any valid token", wsToken(rw), AuthInput{Authorization: basic("x", "t"), Method: "GET", URI: "/v2/"}, http.StatusOK, false},
		{"catalog forbidden", wsToken(rw), AuthInput{Authorization: basic("x", "t"), Method: "GET", URI: "/v2/_catalog"}, http.StatusForbidden, false},
		{"unknown namespace forbidden", wsToken(rw), AuthInput{Authorization: basic("x", "t"), Method: "GET", URI: "/v2/ghost/web/manifests/1"}, http.StatusForbidden, false},

		// --- workspace-scoped token (form #1) ---
		{"ws token pulls own ns", wsToken([]string{models.ScopeRead}), AuthInput{Authorization: basic("acme", "t"), Method: "GET", URI: "/v2/acme/web/manifests/1"}, http.StatusOK, false},
		{"ws token pushes own ns", wsToken(rw), AuthInput{Authorization: basic("acme", "t"), Method: "PUT", URI: "/v2/acme/web/blobs/uploads/x"}, http.StatusOK, false},
		{"ws token rejected on other ns", wsToken(rw), AuthInput{Authorization: basic("acme", "t"), Method: "GET", URI: "/v2/other/web/manifests/1"}, http.StatusForbidden, false},
		{"ws token replay via ws_7 id form", wsToken(rw), AuthInput{Authorization: basic("acme", "t"), Method: "PUT", URI: "/v2/ws_7/web/blobs/uploads/x?_state=y"}, http.StatusOK, false},
		{"ws token rejected on ws_8 id form", wsToken(rw), AuthInput{Authorization: basic("acme", "t"), Method: "GET", URI: "/v2/ws_8/web/manifests/1"}, http.StatusForbidden, false},
		{"read-only ws token cannot push", wsToken([]string{models.ScopeRead}), AuthInput{Authorization: basic("acme", "t"), Method: "POST", URI: "/v2/acme/web/blobs/uploads/"}, http.StatusForbidden, false},

		// --- account-wide user token (form #2) ---
		{"user (developer) pulls a member workspace", userToken(42, rw), AuthInput{Authorization: basic("jane", "t"), Method: "GET", URI: "/v2/acme/web/manifests/1"}, http.StatusOK, false},
		{"user (developer) pushes a member workspace", userToken(42, rw), AuthInput{Authorization: basic("jane", "t"), Method: "PUT", URI: "/v2/acme/web/manifests/1"}, http.StatusOK, false},
		{"user (viewer) pulls a member workspace", userToken(99, []string{models.ScopeRead}), AuthInput{Authorization: basic("vic", "t"), Method: "GET", URI: "/v2/acme/web/manifests/1"}, http.StatusOK, false},
		{"user (viewer) cannot push", userToken(99, rw), AuthInput{Authorization: basic("vic", "t"), Method: "PUT", URI: "/v2/acme/web/manifests/1"}, http.StatusForbidden, false},
		{"user non-member forbidden", userToken(42, rw), AuthInput{Authorization: basic("jane", "t"), Method: "GET", URI: "/v2/other/web/manifests/1"}, http.StatusForbidden, false},
		{"user with read-only scope cannot push", userToken(42, []string{models.ScopeRead}), AuthInput{Authorization: basic("jane", "t"), Method: "PUT", URI: "/v2/acme/web/manifests/1"}, http.StatusForbidden, false},
		{"user via ws_7 id form pushes", userToken(42, rw), AuthInput{Authorization: basic("jane", "t"), Method: "PUT", URI: "/v2/ws_7/web/blobs/uploads/x"}, http.StatusOK, false},

		// --- dedicated registry scopes ---
		{"registry_write pushes", wsToken([]string{models.ScopeRegistryWrite}), AuthInput{Authorization: basic("acme", "t"), Method: "PUT", URI: "/v2/acme/web/manifests/1"}, http.StatusOK, false},
		{"registry_read pulls", wsToken([]string{models.ScopeRegistryRead}), AuthInput{Authorization: basic("acme", "t"), Method: "GET", URI: "/v2/acme/web/manifests/1"}, http.StatusOK, false},
		{"registry_read cannot push", wsToken([]string{models.ScopeRegistryRead}), AuthInput{Authorization: basic("acme", "t"), Method: "PUT", URI: "/v2/acme/web/manifests/1"}, http.StatusForbidden, false},
		{"user registry_write pushes a member workspace", userToken(42, []string{models.ScopeRegistryWrite}), AuthInput{Authorization: basic("jane", "t"), Method: "PUT", URI: "/v2/acme/web/manifests/1"}, http.StatusOK, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.svc.Authorize(tc.in)
			if got.Status != tc.wantStatus {
				t.Fatalf("status = %d (%s), want %d", got.Status, got.Reason, tc.wantStatus)
			}
			if got.Challenge != tc.challenge {
				t.Errorf("challenge = %v, want %v", got.Challenge, tc.challenge)
			}
			// On an allow for a real repo, the rewrite target must point at acme/ws_7.
			if tc.wantStatus == http.StatusOK && got.WorkspaceID != 0 {
				if got.Workspace != "acme" || got.Namespace != "ws_7" || got.WorkspaceID != 7 {
					t.Errorf("target = (%q,%q,%d), want (acme,ws_7,7)", got.Workspace, got.Namespace, got.WorkspaceID)
				}
			}
		})
	}
}

func TestAuthorizePlatformToken(t *testing.T) {
	// The platform token (build/deploy worker) authorizes any namespace; keys are
	// never consulted (Verify would fail here, proving the short-circuit).
	svc := &Service{cfg: config.RegistryConfig{PlatformToken: "plat-secret"}, ws: wsFixture(), keys: fakeKeys{err: errors.New("nope")}}

	cases := []struct {
		name       string
		in         AuthInput
		wantStatus int
		wantNs     string
	}{
		{"push to acme", AuthInput{Authorization: basic("_miabi", "plat-secret"), Method: "PUT", URI: "/v2/acme/app-5/manifests/9"}, http.StatusOK, "ws_7"},
		{"push to other (any namespace)", AuthInput{Authorization: basic("_miabi", "plat-secret"), Method: "PUT", URI: "/v2/other/app-1/blobs/uploads/"}, http.StatusOK, "ws_8"},
		{"pull via ws_8 id form", AuthInput{Authorization: basic("_miabi", "plat-secret"), Method: "GET", URI: "/v2/ws_8/app-1/manifests/9"}, http.StatusOK, "ws_8"},
		{"base v2", AuthInput{Authorization: basic("_miabi", "plat-secret"), Method: "GET", URI: "/v2/"}, http.StatusOK, ""},
		{"catalog forbidden", AuthInput{Authorization: basic("_miabi", "plat-secret"), Method: "GET", URI: "/v2/_catalog"}, http.StatusForbidden, ""},
		{"unknown namespace", AuthInput{Authorization: basic("_miabi", "plat-secret"), Method: "PUT", URI: "/v2/ghost/app-1/manifests/9"}, http.StatusForbidden, ""},
		{"wrong token falls through to key path", AuthInput{Authorization: basic("_miabi", "wrong"), Method: "GET", URI: "/v2/acme/app-1/manifests/9"}, http.StatusUnauthorized, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := svc.Authorize(tc.in)
			if got.Status != tc.wantStatus {
				t.Fatalf("status = %d (%s), want %d", got.Status, got.Reason, tc.wantStatus)
			}
			if tc.wantNs != "" && got.Namespace != tc.wantNs {
				t.Errorf("namespace = %q, want %q", got.Namespace, tc.wantNs)
			}
		})
	}
}

func TestParseRepo(t *testing.T) {
	cases := []struct {
		uri     string
		repo    string
		base    bool
		catalog bool
	}{
		{"/v2/", "", true, false},
		{"/v2/_catalog", "", false, true},
		{"/v2/acme/web/manifests/1.0", "acme/web", false, false},
		{"/v2/acme/web/blobs/uploads/uuid", "acme/web", false, false},
		{"/v2/acme/team/web/tags/list", "acme/team/web", false, false},
		{"/v2/acme/web/manifests/sha256:abc?foo=bar", "acme/web", false, false},
		{"https://registry.example.com/v2/acme/web/manifests/1", "acme/web", false, false},
	}
	for _, tc := range cases {
		repo, base, catalog := parseRepo(tc.uri)
		if repo != tc.repo || base != tc.base || catalog != tc.catalog {
			t.Errorf("parseRepo(%q) = (%q,%v,%v), want (%q,%v,%v)", tc.uri, repo, base, catalog, tc.repo, tc.base, tc.catalog)
		}
	}
}

func TestFirstSegmentAndIDNamespace(t *testing.T) {
	if firstSegment("acme/team/web") != "acme" {
		t.Errorf("firstSegment wrong")
	}
	if id, ok := parseIDNamespace("ws_42"); !ok || id != 42 {
		t.Errorf("parseIDNamespace(ws_42) = (%d,%v), want (42,true)", id, ok)
	}
	for _, ns := range []string{"acme", "ws_", "ws_x", "ws-5", "wsabc"} {
		if _, ok := parseIDNamespace(ns); ok {
			t.Errorf("parseIDNamespace(%q) should be false", ns)
		}
	}
}
