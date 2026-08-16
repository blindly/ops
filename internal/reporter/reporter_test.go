package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/blindly/ops/internal/runner"
)

func TestPrintText(t *testing.T) {
	var buf bytes.Buffer
	PrintText(&buf, &runner.JobResult{
		TaskResults: []runner.TaskResult{
			{TaskName: "ping", Server: "web1", ExitCode: 0, Duration: 100},
		},
	})
	if !strings.Contains(buf.String(), "ping") {
		t.Fatalf("expected ping in output, got:\n%s", buf.String())
	}
}

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintJSON(&buf, &runner.JobResult{
		TaskResults: []runner.TaskResult{
			{TaskName: "ping", Server: "web1", ExitCode: 0, Duration: 100},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "ping") {
		t.Fatalf("expected ping in JSON, got:\n%s", buf.String())
	}
}

func TestPrintTextShowsOutput(t *testing.T) {
	var buf bytes.Buffer
	PrintText(&buf, &runner.JobResult{
		TaskResults: []runner.TaskResult{
			{TaskName: "uptime", Server: "web1", ExitCode: 0, Stdout: " 12:34:56 up 1 day\n", Duration: 50},
		},
	})
	out := buf.String()
	if !strings.Contains(out, "12:34:56 up 1 day") {
		t.Fatalf("expected stdout in text output, got:\n%s", out)
	}
}

func TestPrintTextQuietHidesOutput(t *testing.T) {
	var buf bytes.Buffer
	PrintTextWithOpts(&buf, &runner.JobResult{
		TaskResults: []runner.TaskResult{
			{TaskName: "uptime", Server: "web1", ExitCode: 0, Stdout: "should not appear", Duration: 50},
		},
	}, false)
	out := buf.String()
	if strings.Contains(out, "should not appear") {
		t.Fatalf("expected stdout hidden in quiet mode, got:\n%s", out)
	}
	if !strings.Contains(out, "uptime") {
		t.Fatalf("expected task name in quiet output, got:\n%s", out)
	}
}

func TestPrintTextShowsStderr(t *testing.T) {
	var buf bytes.Buffer
	PrintText(&buf, &runner.JobResult{
		TaskResults: []runner.TaskResult{
			{TaskName: "fail", Server: "web1", ExitCode: 1, Stdout: "partial output\n", Stderr: "permission denied\n", Duration: 50},
		},
	})
	out := buf.String()
	if !strings.Contains(out, "partial output") {
		t.Fatalf("expected stdout in output, got:\n%s", out)
	}
	if !strings.Contains(out, "permission denied") {
		t.Fatalf("expected stderr in output, got:\n%s", out)
	}
}
