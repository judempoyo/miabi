// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package stack

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParsePortMapping(t *testing.T) {
	cases := []struct {
		in        string
		host      int
		container int
		proto     string
		ok        bool
	}{
		{"9000:9000", 9000, 9000, "tcp", true},
		{"8080:80", 8080, 80, "tcp", true},
		{"80", 0, 80, "tcp", true}, // exposed, not published
		{"53:53/udp", 53, 53, "udp", true},
		{"127.0.0.1:9000:9000", 9000, 9000, "tcp", true},
		{"notaport", 0, 0, "", false},
	}
	for _, c := range cases {
		m, ok := parsePortMapping(c.in)
		if ok != c.ok {
			t.Errorf("%q: ok=%v, want %v", c.in, ok, c.ok)
			continue
		}
		if ok && (m.Host != c.host || m.Container != c.container || m.Proto != c.proto) {
			t.Errorf("%q: got %d:%d/%s, want %d:%d/%s", c.in, m.Host, m.Container, m.Proto, c.host, c.container, c.proto)
		}
	}
}

func TestParseComposeVolume(t *testing.T) {
	cases := []struct {
		in     string
		source string
		target string
		bind   bool
		ok     bool
	}{
		{"db_data:/var/lib/postgresql/data", "db_data", "/var/lib/postgresql/data", false, true},
		{"redis_data:/data", "redis_data", "/data", false, true},
		{"config:/etc/app:ro", "config", "/etc/app", false, true},
		{"/host/path:/container", "/host/path", "/container", true, true}, // bind
		{"./rel:/container", "./rel", "/container", true, true},           // bind
		{"/data", "", "/data", false, true},                               // anonymous
		{"", "", "", false, false},
	}
	for _, c := range cases {
		m, ok := parseComposeVolume(c.in)
		if ok != c.ok {
			t.Errorf("%q: ok=%v want %v", c.in, ok, c.ok)
			continue
		}
		if ok && (m.Source != c.source || m.Target != c.target || m.Bind != c.bind) {
			t.Errorf("%q: got {%q %q bind=%v} want {%q %q bind=%v}", c.in, m.Source, m.Target, m.Bind, c.source, c.target, c.bind)
		}
	}
}

func TestComposeUnmarshal_ParsesServiceVolumes(t *testing.T) {
	const y = `
services:
  posta-db:
    image: postgres:17-alpine
    volumes:
      - db_data:/var/lib/postgresql/data
  posta-redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
volumes:
  db_data:
  redis_data:
`
	var cf composeFile
	if err := yaml.Unmarshal([]byte(y), &cf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	db := cf.Services["posta-db"]
	if len(db.Volumes) != 1 || db.Volumes[0] != "db_data:/var/lib/postgresql/data" {
		t.Errorf("posta-db volumes = %v", db.Volumes)
	}
}

func TestComposeUnmarshal_EnvAndCommandForms(t *testing.T) {
	const y = `
services:
  web:
    image: nginx:1.25
    command: node server.js
    environment:
      FOO: bar
      BAZ: qux
  worker:
    image: myorg/worker
    command: ["celery", "worker"]
    environment:
      - A=1
      - B=2
`
	var cf composeFile
	if err := yaml.Unmarshal([]byte(y), &cf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(cf.Services) != 2 {
		t.Fatalf("got %d services, want 2", len(cf.Services))
	}
	web := cf.Services["web"]
	if len(web.Command) != 2 || web.Command[0] != "node" || web.Command[1] != "server.js" {
		t.Errorf("web command string form = %v", web.Command)
	}
	if len(web.Environment) != 2 {
		t.Errorf("web env mapping form = %v", web.Environment)
	}
	worker := cf.Services["worker"]
	if len(worker.Command) != 2 || worker.Command[1] != "worker" {
		t.Errorf("worker command list form = %v", worker.Command)
	}
	if len(worker.Environment) != 2 || worker.Environment[0].Key != "A" || worker.Environment[0].Value != "1" {
		t.Errorf("worker env list form = %v", worker.Environment)
	}
}
