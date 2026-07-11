// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package pipeline implements Miabi's CI/CD: pipeline-as-code
// (kind: Pipeline) versioned with the app, PipelineRun lifecycle, and the
// internal runner that executes steps in isolated containers. It is the
// imperative, push-based half of the GitOps & CI/CD model — a pipeline turns a
// commit into an image and a Release.
package pipeline

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	apiVersion   = "miabi.io/v1"
	kindPipeline = "Pipeline"
	// UsesBuild is the built-in step that builds the shared workspace into an
	// image, captures its digest, and records an Image row. It is what turns a
	// commit into a deployable artifact.
	UsesBuild = "build"
	// UsesDeploy is the built-in terminal step that deploys an app by the image
	// the pipeline built (deploy-by-digest, no rebuild).
	UsesDeploy = "deploy"
)

// Spec is a parsed pipeline-as-code document.
type Spec struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   SpecMeta `yaml:"metadata"`
	On         Triggers `yaml:"on"`
	Steps      []Step   `yaml:"steps"`
}

// SpecMeta is the pipeline's identity block.
type SpecMeta struct {
	Name string `yaml:"name"`
}

// Triggers declares how a pipeline starts.
type Triggers struct {
	Push     *PushTrigger `yaml:"push,omitempty"`
	Manual   bool         `yaml:"manual,omitempty"`
	Schedule string       `yaml:"schedule,omitempty"`
}

// PushTrigger fires on a push to one of the named branches (empty = any).
type PushTrigger struct {
	Branches []string `yaml:"branches,omitempty"`
}

// Step is one unit of work in a pipeline.
type Step struct {
	Name string `yaml:"name"`
	// Image + Run execute a command in an isolated container.
	Image string `yaml:"image,omitempty"`
	Run   string `yaml:"run,omitempty"`
	// Uses selects a built-in step ("build" | "deploy") instead of a container.
	Uses string `yaml:"uses,omitempty"`
	// Dockerfile names the Dockerfile for a `uses: build` step, relative to the
	// workspace root. Empty defaults to "Dockerfile".
	Dockerfile string `yaml:"dockerfile,omitempty"`
	// App overrides the deploy target for a `uses: deploy` step.
	App string            `yaml:"app,omitempty"`
	Env map[string]string `yaml:"env,omitempty"`
	// ContinueOnError keeps the run going when this step fails (the step is still
	// marked failed, the run can still succeed) — like GitHub Actions'
	// `continue-on-error: true`.
	ContinueOnError bool `yaml:"continue-on-error,omitempty"`
}

// FiresOnBranch reports whether a push trigger matches the given branch.
func (t Triggers) FiresOnBranch(branch string) bool {
	if t.Push == nil {
		return false
	}
	if len(t.Push.Branches) == 0 {
		return true
	}
	for _, b := range t.Push.Branches {
		if b == branch {
			return true
		}
	}
	return false
}

// ParseSpec decodes and validates a kind: Pipeline document. Unknown fields are
// rejected so typos surface.
func ParseSpec(data []byte) (*Spec, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var s Spec
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("parse pipeline: %w", err)
	}
	if s.APIVersion != apiVersion {
		return nil, fmt.Errorf("apiVersion must be %q, got %q", apiVersion, s.APIVersion)
	}
	if s.Kind != kindPipeline {
		return nil, fmt.Errorf("kind must be %q, got %q", kindPipeline, s.Kind)
	}
	if len(s.Steps) == 0 {
		return nil, fmt.Errorf("pipeline must declare at least one step")
	}
	names := map[string]bool{}
	for i, st := range s.Steps {
		if strings.TrimSpace(st.Name) == "" {
			return nil, fmt.Errorf("step %d: name is required", i+1)
		}
		if names[st.Name] {
			return nil, fmt.Errorf("duplicate step name %q", st.Name)
		}
		names[st.Name] = true
		if st.Uses == "" && strings.TrimSpace(st.Image) == "" {
			return nil, fmt.Errorf("step %q: image is required (or use a built-in via 'uses')", st.Name)
		}
		if st.Uses != "" && st.Uses != UsesDeploy && st.Uses != UsesBuild {
			return nil, fmt.Errorf("step %q: unknown built-in %q", st.Name, st.Uses)
		}
		if st.Dockerfile != "" && st.Uses != UsesBuild {
			return nil, fmt.Errorf("step %q: 'dockerfile' is only valid on a 'uses: build' step", st.Name)
		}
	}
	return &s, nil
}
