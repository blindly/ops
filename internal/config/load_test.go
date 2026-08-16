package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJob(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "job.yaml")
	content := `
name: test job
servers: [web1]
tasks:
  - name: ping
    command: echo pong
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	job, err := LoadJob(path)
	if err != nil {
		t.Fatal(err)
	}
	if job.Name != "test job" {
		t.Fatalf("expected name 'test job', got %q", job.Name)
	}
	if len(job.Tasks) != 1 || job.Tasks[0].Command != "echo pong" {
		t.Fatalf("unexpected tasks: %+v", job.Tasks)
	}
}

func TestLoadJobServersScalarPath(t *testing.T) {
	dir := t.TempDir()
	serversPath := filepath.Join(dir, "servers.yaml")
	if err := os.WriteFile(serversPath, []byte("servers:\n  - web1\n  - db1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	jobPath := filepath.Join(dir, "job.yaml")
	content := `
name: scalar servers job
servers: ` + serversPath + `
tasks:
  - name: ping
    command: echo pong
`
	if err := os.WriteFile(jobPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	job, err := LoadJob(jobPath)
	if err != nil {
		t.Fatal(err)
	}
	if !job.Servers.IsSet {
		t.Fatal("expected Servers.IsSet to be true")
	}
	if job.Servers.Path != serversPath {
		t.Fatalf("expected servers path %q, got %q", serversPath, job.Servers.Path)
	}
	if len(job.Servers.List) != 0 {
		t.Fatalf("expected empty list, got %v", job.Servers.List)
	}
}

func TestLoadServers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "servers.yaml")
	content := "servers:\n  - web1\n  - db1\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadServers(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Servers) != 2 || s.Servers[0] != "web1" {
		t.Fatalf("unexpected servers: %+v", s.Servers)
	}
}

func TestJobValidateErrors(t *testing.T) {
	tests := []struct {
		name    string
		job     Job
		wantErr string
	}{
		{
			name:    "no tasks",
			job:     Job{Name: "test", Servers: ServersSource{IsSet: true, List: []string{"web1"}}},
			wantErr: "job must have at least one task",
		},
		{
			name: "task missing command/script/shell",
			job: Job{
				Name:    "test",
				Servers: ServersSource{IsSet: true, List: []string{"web1"}},
				Tasks:   []Task{{Name: "noop"}},
			},
			wantErr: "task noop: one of command, script, shell, or upload is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.job.Validate()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestServersSourceUnmarshalValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "empty path",
			content: "name: test\nservers: ''\ntasks:\n  - name: ping\n    command: echo pong\n",
			wantErr: "servers path must be a non-empty string",
		},
		{
			name:    "explicit null",
			content: "name: test\nservers: !!null\ntasks:\n  - name: ping\n    command: echo pong\n",
			wantErr: "servers path must be a non-empty string",
		},
		{
			name:    "non-scalar list item",
			content: "name: test\nservers:\n  - [nested]\ntasks:\n  - name: ping\n    command: echo pong\n",
			wantErr: "servers list must contain only scalar aliases",
		},
		{
			name:    "empty string list entry",
			content: "name: test\nservers: [web1, \"\"]\ntasks:\n  - name: ping\n    command: echo pong\n",
			wantErr: "servers list entries must be non-empty strings",
		},
		{
			name:    "null list entry",
			content: "name: test\nservers: [web1, null]\ntasks:\n  - name: ping\n    command: echo pong\n",
			wantErr: "servers list entries must be non-empty strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "job.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadJob(path)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}
