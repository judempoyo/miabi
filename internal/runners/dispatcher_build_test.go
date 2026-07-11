// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package runners

import (
	"strings"
	"testing"
	"time"

	"github.com/miabi-io/runner/proto"
)

func TestBuildOnlyJobSpec(t *testing.T) {
	in := BuildInputs{
		DeploymentID:     42,
		DeploymentNumber: 3,
		WorkspaceID:      7,
		WorkspaceName:    "acme",
		AppID:            5,
		AppName:          "web",
		SourceURL:        "https://git.example.com/acme/web.git",
		Commit:           "abc123",
		Repository:       "reg.example.com/ws_7/app-5",
		Registry:         "reg.example.com",
		Build:            &proto.BuildConfig{Method: "buildpack", Builder: "paketobuildpacks/builder-jammy-base"},
	}
	spec, secrets := buildOnlyJobSpec(in, nil, time.Time{})

	// RunID carries the per-app deployment number (the image tag), not the id.
	if spec.RunID != 3 {
		t.Errorf("RunID = %d, want the deployment number 3", spec.RunID)
	}
	if len(spec.Steps) != 1 || spec.Steps[0].Uses != "build" {
		t.Fatalf("want exactly one build step, got %+v", spec.Steps)
	}
	if spec.Steps[0].Build == nil || spec.Steps[0].Build.Method != "buildpack" {
		t.Errorf("build config not carried onto the step: %+v", spec.Steps[0].Build)
	}
	env := strings.Join(spec.Env, "\n")
	for _, want := range []string{
		"MIABI_IMAGE_REPOSITORY=reg.example.com/ws_7/app-5",
		"MIABI_APP_ID=5",
		"MIABI_COMMIT=abc123",
		"MIABI_REGISTRY=reg.example.com",
	} {
		if !strings.Contains(env, want) {
			t.Errorf("env missing %q; got %v", want, spec.Env)
		}
	}
	if len(secrets) != 0 {
		t.Errorf("no creds -> no secrets to mask, got %v", secrets)
	}
}
