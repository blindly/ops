package config

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveServers(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "ssh_config")
	content := `
Host web1
    HostName 10.0.0.5
    User admin
    Port 2222
    IdentityFile ~/.ssh/web1

Host db1
    HostName db.internal
    User ubuntu

Host fallback
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	currentUser, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}

	servers, err := ResolveServers([]string{"web1", "db1", "fallback"}, configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(servers))
	}

	if servers[0].HostName != "10.0.0.5" || servers[0].User != "admin" || servers[0].Port != "2222" {
		t.Fatalf("unexpected web1: %+v", servers[0])
	}
	if servers[0].IdentityFile != "~/.ssh/web1" {
		t.Fatalf("unexpected web1 IdentityFile: got %q, want %q", servers[0].IdentityFile, "~/.ssh/web1")
	}

	if servers[1].HostName != "db.internal" || servers[1].Port != "22" {
		t.Fatalf("unexpected db1: %+v", servers[1])
	}

	if servers[2].HostName != "fallback" {
		t.Fatalf("fallback HostName should default to alias, got %q", servers[2].HostName)
	}
	if servers[2].User != currentUser.Username {
		t.Fatalf("fallback User should default to current user, got %q, want %q", servers[2].User, currentUser.Username)
	}
	if servers[2].Port != "22" {
		t.Fatalf("fallback Port should default to 22, got %q", servers[2].Port)
	}
}

func TestResolveServersDefaultPath(t *testing.T) {
	dir := t.TempDir()
	sshDir := filepath.Join(dir, ".ssh")
	if err := os.MkdirAll(sshDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(sshDir, "config")
	content := `
Host web1
    HostName 10.0.0.5
    User admin
    Port 2222
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", dir)

	servers, err := ResolveServers([]string{"web1"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].HostName != "10.0.0.5" || servers[0].User != "admin" || servers[0].Port != "2222" {
		t.Fatalf("unexpected web1: %+v", servers[0])
	}
}

func TestResolveServersInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "ssh_config")
	content := `Match
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ResolveServers([]string{"web1"}, configPath)
	if err == nil {
		t.Fatal("expected error for invalid ssh config, got nil")
	}
}

func TestResolveServersInvalidPort(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		wantErr string
	}{
		{"port too high", "70000", "invalid port 70000"},
		{"port zero", "0", "invalid port 0"},
		{"port negative", "-1", "invalid port -1"},
		{"port non-numeric", "abc", "invalid port \"abc\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "ssh_config")
			content := "Host bad\n    Port " + tt.port + "\n"
			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := ResolveServers([]string{"bad"}, configPath)
			if err == nil {
				t.Fatal("expected error for invalid port, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error to contain %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}
