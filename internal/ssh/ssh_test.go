package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildAuthMethods_NoMethods(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	_, err := buildAuthMethods(AuthOptions{})
	if err == nil {
		t.Fatal("expected error when no auth methods available")
	}
	if got := err.Error(); got != "no authentication methods available" {
		t.Fatalf("unexpected error message: %q", got)
	}
}

func TestBuildAuthMethods_Password(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	methods, err := buildAuthMethods(AuthOptions{Password: "secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(methods))
	}
}

func TestBuildAuthMethods_IdentityFile(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_rsa")

	keyData := []byte(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACAQnC0s0r0S8UwbJVLXdKiAcAy+QBgeXRv8fKK40Vn8DAAAAIj3OjSF9zo0
hQAAAAtzc2gtZWQyNTUxOQAAACAQnC0s0r0S8UwbJVLXdKiAcAy+QBgeXRv8fKK40Vn8DA
AAAEBhKW37dSi5oa+zIIPXBLC+QX5IrcgrlGGiDeb6yX2IIhCcLSzSvRLxTBslUtd0qIBw
DL5AGB5dG/x8orjRWfwMAAAABHRlc3QB
-----END OPENSSH PRIVATE KEY-----
`)

	if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
		t.Fatalf("write test key: %v", err)
	}

	methods, err := buildAuthMethods(AuthOptions{IdentityFile: keyPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(methods))
	}
}

func TestBuildAuthMethods_IdentityFileNotFound(t *testing.T) {
	_, err := buildAuthMethods(AuthOptions{IdentityFile: "/nonexistent/key"})
	if err == nil {
		t.Fatal("expected error for missing identity file")
	}
}

func TestBuildAuthMethods_IdentityFileInvalid(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "bad_key")
	if err := os.WriteFile(keyPath, []byte("not a key"), 0600); err != nil {
		t.Fatalf("write test key: %v", err)
	}

	_, err := buildAuthMethods(AuthOptions{IdentityFile: keyPath})
	if err == nil {
		t.Fatal("expected error for invalid identity file")
	}
}

func TestBuildAuthMethods_TildeExpansion(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	fakeHome := t.TempDir()
	keyPath := filepath.Join(fakeHome, "id_ed25519")
	keyData := []byte(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACAQnC0s0r0S8UwbJVLXdKiAcAy+QBgeXRv8fKK40Vn8DAAAAIj3OjSF9zo0
hQAAAAtzc2gtZWQyNTUxOQAAACAQnC0s0r0S8UwbJVLXdKiAcAy+QBgeXRv8fKK40Vn8DA
AAAEBhKW37dSi5oa+zIIPXBLC+QX5IrcgrlGGiDeb6yX2IIhCcLSzSvRLxTBslUtd0qIBw
DL5AGB5dG/x8orjRWfwMAAAABHRlc3QB
-----END OPENSSH PRIVATE KEY-----
`)
	if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
		t.Fatalf("write test key: %v", err)
	}

	t.Setenv("HOME", fakeHome)
	methods, err := buildAuthMethods(AuthOptions{IdentityFile: "~/id_ed25519"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(methods))
	}
}

func TestHostKeyCallback_IgnoreHostKey(t *testing.T) {
	cb, err := hostKeyCallback(AuthOptions{IgnoreHostKey: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cb == nil {
		t.Fatal("expected non-nil callback")
	}
}

func TestHostKeyCallback_MissingKnownHosts(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	_, err := hostKeyCallback(AuthOptions{})
	if err == nil {
		t.Fatal("expected error when known_hosts file is missing")
	}
}

func TestHostKeyCallback_ProvidedKnownHosts(t *testing.T) {
	dir := t.TempDir()
	hostsPath := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(hostsPath, []byte{}, 0644); err != nil {
		t.Fatalf("write empty known_hosts: %v", err)
	}

	cb, err := hostKeyCallback(AuthOptions{KnownHostsPath: hostsPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cb == nil {
		t.Fatal("expected non-nil callback")
	}
}

func TestIntegrationExec(t *testing.T) {
	if os.Getenv("OPS_SSH_TEST_ADDR") == "" {
		t.Skip("set OPS_SSH_TEST_ADDR to run integration test")
	}
}
