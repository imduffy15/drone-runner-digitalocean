// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package resource

import (
	"testing"

	"github.com/drone/runner-go/manifest"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	got, err := manifest.ParseFile("testdata/manifest.yml")
	if err != nil {
		t.Error(err)
		return
	}

	want := []manifest.Resource{
		&manifest.Signature{
			Kind: "signature",
			Hmac: "a8842634682b78946a2",
		},
		&Pipeline{
			Kind:    "pipeline",
			Type:    "gcp",
			Name:    "default",
			Version: "1",
			Server: Server{
				Image:  "docker-18-04",
				Region: "nyc1",
				Size:   "s-1vcpu-1gb",
			},
			Workspace: manifest.Workspace{
				Path: "/drone/src",
			},
			Platform: manifest.Platform{
				OS:   "linux",
				Arch: "arm64",
			},
			Clone: manifest.Clone{
				Depth: 50,
			},
			Trigger: manifest.Conditions{
				Branch: manifest.Condition{
					Include: []string{"master"},
				},
			},
			Steps: []*Step{
				{
					Name:      "build",
					Shell:     "/bin/sh",
					Detach:    false,
					DependsOn: []string{"clone"},
					Commands: []string{
						"go build",
						"go test",
					},
					Environment: map[string]*manifest.Variable{
						"GOOS":   &manifest.Variable{Value: "linux"},
						"GOARCH": &manifest.Variable{Value: "arm64"},
					},
					Failure: "never",
					When: manifest.Conditions{
						Event: manifest.Condition{
							Include: []string{"push"},
						},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(got.Resources, want); diff != "" {
		t.Errorf("Unexpected manifest")
		t.Log(diff)
	}
}

func TestParseErr(t *testing.T) {
	_, err := manifest.ParseFile("testdata/malformed.yml")
	if err == nil {
		t.Errorf("Expect error when malformed yaml")
	}
}

func TestParseLintErr(t *testing.T) {
	_, err := manifest.ParseFile("testdata/linterr.yml")
	if err == nil {
		t.Errorf("Expect linter returns error")
		return
	}
}

func TestParseNoMatch(t *testing.T) {
	r := &manifest.RawResource{Kind: "pipeline", Type: "docker"}
	_, match, _ := parse(r)
	if match {
		t.Errorf("Expect no match")
	}
}

func TestMatch(t *testing.T) {
	r := &manifest.RawResource{
		Kind: "pipeline",
		Type: "gcp",
	}
	if match(r) == false {
		t.Errorf("Expect match, got false")
	}

	r = &manifest.RawResource{
		Kind: "approval",
		Type: "gcp",
	}
	if match(r) == true {
		t.Errorf("Expect kind mismatch, got true")
	}

	r = &manifest.RawResource{
		Kind: "pipeline",
		Type: "docker",
	}
	if match(r) == true {
		t.Errorf("Expect type mismatch, got true")
	}

}

func TestLint(t *testing.T) {
	p := new(Pipeline)
	p.Server = Server{
		Image:  "docker-18-04",
		Region: "nyc1",
		Size:   "s-1vcpu-1gb",
	}
	p.Steps = []*Step{{Name: "build"}, {Name: "test"}}
	if err := lint(p); err != nil {
		t.Errorf("Expect no lint error, got %s", err)
	}

	p.Steps = []*Step{{Name: "build"}, {Name: "build"}}
	if err := lint(p); err == nil {
		t.Errorf("Expect error when duplicate name")
	}

	p.Steps = []*Step{{Name: "build"}, {Name: ""}}
	if err := lint(p); err == nil {
		t.Errorf("Expect error when empty name")
	}

	p.Steps = []*Step{{Name: "build", Detach: true}}
	if err := lint(p); err == nil {
		t.Errorf("Expect error when step detached")
	}
}

func TestLint_ServerError(t *testing.T) {
	p := new(Pipeline)
	p.Server = Server{
		Image:  "docker-18-04",
		Region: "nyc1",
		Size:   "s-1vcpu-1gb",
	}
	if err := lint(p); err != nil {
		t.Errorf("Expect no lint error, got %s", err)
		return
	}
}
