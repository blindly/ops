package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpContainsUsage(t *testing.T) {
	cmd := exec.Command("go", "run", "../cmd/ops", "--help")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("running help: %v\noutput:\n%s", err, out.String())
	}
	got := out.String()
	if !strings.Contains(got, "ops") {
		t.Errorf("help output missing 'ops':\n%s", got)
	}
	if !strings.Contains(got, "Run shell jobs on remote servers") {
		t.Errorf("help output missing short description:\n%s", got)
	}
}

// TestRunErrorsOnMissingJob verifies the wired-up run command rejects a
// missing job file rather than silently echoing the argument. The stub
// from the bootstrap task printed "run: <file>"; the real command now loads
// the job, so a nonexistent path must surface a "load job" error.
func TestRunErrorsOnMissingJob(t *testing.T) {
	cmd := exec.Command("go", "run", "../cmd/ops", "run", "does-not-exist.yaml")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit for missing job file, got nil\noutput:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "load job") {
		t.Errorf("expected 'load job' in output, got:\n%s", out.String())
	}
}


func TestNewCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("go", "run", "../cmd/ops", "new", dir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("ops new: %v\n%s", err, out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "jobs.yaml")); err != nil {
		t.Fatalf("jobs.yaml not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "servers.yaml")); err != nil {
		t.Fatalf("servers.yaml not created: %v", err)
	}
}

func TestValidateDefaultsToJobsYAML(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "ops")
	build := exec.Command("go", "build", "-o", bin, "../cmd/ops")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	content := []byte("name: test\nservers: [web1]\ntasks:\n  - name: hello\n    command: echo hi\n")
	if err := os.WriteFile(filepath.Join(dir, "jobs.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "validate")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ops validate: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "valid") {
		t.Fatalf("expected 'valid', got:\n%s", out)
	}
}