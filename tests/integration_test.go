package tests

import (
	"context"
	"os"
	"testing"

	"github.com/blindly/ops/internal/config"
	"github.com/blindly/ops/internal/runner"
)

// TestRunnerLocalhost requires an SSH server on localhost with key auth.
// It is skipped unless OPS_TEST_HOST is set.
func TestRunnerLocalhost(t *testing.T) {
	alias := os.Getenv("OPS_TEST_HOST")
	if alias == "" {
		t.Skip("set OPS_TEST_HOST to run integration test")
	}

	servers, err := config.ResolveServers([]string{alias}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}

	job := &config.Job{
		Name: "integration",
		Defaults: config.Defaults{Interpreter: "bash"},
		Tasks: []config.Task{
			{Name: "ping", Command: "echo pong"},
		},
	}

	res, err := runner.Run(context.Background(), runner.Options{
		Job:     job,
		Servers: servers,
		Workers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.TaskResults) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res.TaskResults))
	}
	if res.TaskResults[0].ExitCode != 0 || res.TaskResults[0].Stdout != "pong\n" {
		t.Fatalf("unexpected result: %+v", res.TaskResults[0])
	}
}