// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import "testing"

const validPipeline = `
apiVersion: miabi.io/v1
kind: Pipeline
metadata: { name: web }
on:
  push: { branches: [main] }
  manual: true
steps:
  - name: build
    image: gcr.io/kaniko-project/executor
    run: "--dockerfile=Dockerfile --destination=$IMAGE"
  - name: test
    image: node:20
    run: "npm test"
  - name: deploy
    uses: deploy
    app: web
`

func TestParseSpecValid(t *testing.T) {
	s, err := ParseSpec([]byte(validPipeline))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(s.Steps) != 3 {
		t.Fatalf("want 3 steps, got %d", len(s.Steps))
	}
	if s.Steps[2].Uses != UsesDeploy {
		t.Errorf("deploy step not recognized: %q", s.Steps[2].Uses)
	}
	if !s.On.FiresOnBranch("main") || s.On.FiresOnBranch("dev") {
		t.Error("push branch matching is wrong")
	}
}

func TestParseSpecContinueOnError(t *testing.T) {
	const y = `
apiVersion: miabi.io/v1
kind: Pipeline
metadata: { name: web }
steps:
  - name: scan
    image: aquasec/trivy:latest
    continue-on-error: true
    run: "trivy image $MIABI_IMAGE"
  - name: deploy
    uses: deploy
`
	s, err := ParseSpec([]byte(y))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !s.Steps[0].ContinueOnError {
		t.Error("continue-on-error not parsed on the scan step")
	}
	if s.Steps[1].ContinueOnError {
		t.Error("continue-on-error should default to false when omitted")
	}
}

func TestParseSpecAcceptsBuildStep(t *testing.T) {
	const y = `
apiVersion: miabi.io/v1
kind: Pipeline
metadata: { name: web }
steps:
  - name: build
    uses: build
    dockerfile: docker/Dockerfile
  - name: deploy
    uses: deploy
`
	s, err := ParseSpec([]byte(y))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if s.Steps[0].Uses != UsesBuild {
		t.Errorf("build step not recognized: %q", s.Steps[0].Uses)
	}
	if s.Steps[0].Dockerfile != "docker/Dockerfile" {
		t.Errorf("dockerfile not parsed: %q", s.Steps[0].Dockerfile)
	}
}

func TestParseSpecRejectsDockerfileOnNonBuildStep(t *testing.T) {
	const y = `
apiVersion: miabi.io/v1
kind: Pipeline
metadata: { name: x }
steps:
  - name: test
    image: node:20
    run: "npm test"
    dockerfile: Dockerfile
`
	if _, err := ParseSpec([]byte(y)); err == nil {
		t.Fatal("expected error: dockerfile is only valid on a build step")
	}
}

func TestParseSpecRejectsUnknownBuiltin(t *testing.T) {
	const y = `
apiVersion: miabi.io/v1
kind: Pipeline
metadata: { name: x }
steps:
  - name: nope
    uses: teleport
`
	if _, err := ParseSpec([]byte(y)); err == nil {
		t.Fatal("expected error for unknown built-in step")
	}
}

func TestParseSpecRequiresImageOrUses(t *testing.T) {
	const y = `
apiVersion: miabi.io/v1
kind: Pipeline
metadata: { name: x }
steps:
  - name: empty
    run: "echo hi"
`
	if _, err := ParseSpec([]byte(y)); err == nil {
		t.Fatal("expected error for step with neither image nor uses")
	}
}

func TestParseSpecRejectsDuplicateStepNames(t *testing.T) {
	const y = `
apiVersion: miabi.io/v1
kind: Pipeline
metadata: { name: x }
steps:
  - name: a
    image: busybox
  - name: a
    image: busybox
`
	if _, err := ParseSpec([]byte(y)); err == nil {
		t.Fatal("expected error for duplicate step names")
	}
}
