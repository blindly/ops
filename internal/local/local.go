package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type executor struct{}

// NewExecutor returns an Executor that runs tasks on the local machine.
func NewExecutor() *executor {
	return &executor{}
}

// RunCommand runs a single command string through the interpreter.
func (e *executor) RunCommand(ctx context.Context, interpreter, workingDir, command string, env map[string]string, stdout, stderr io.Writer) (int, error) {
	args := commandArgs(interpreter, command)
	return run(ctx, interpreter, workingDir, args, env, stdout, stderr)
}

// RunScript writes data to a temporary file and executes it.
func (e *executor) RunScript(ctx context.Context, interpreter, name, workingDir string, data []byte, env map[string]string, stdout, stderr io.Writer, keep bool) (int, error) {
	ext := ".sh"
	if interpreter == "powershell" || interpreter == "pwsh" {
		ext = ".ps1"
	} else if interpreter == "cmd" {
		ext = ".bat"
	}

	f, err := os.CreateTemp(os.TempDir(), "ops-"+name+"-*"+ext)
	if err != nil {
		return 0, fmt.Errorf("create temp: %w", err)
	}
	tmp := f.Name()
	if !keep {
		defer os.Remove(tmp)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return 0, err
	}
	if err := f.Close(); err != nil {
		return 0, err
	}
	if ext == ".sh" || ext == ".ps1" {
		_ = os.Chmod(tmp, 0750)
	}

	args := scriptArgs(interpreter, tmp)
	return run(ctx, interpreter, workingDir, args, env, stdout, stderr)
}

// UploadFile writes data to a local file path.
func (e *executor) UploadFile(ctx context.Context, dest string, data []byte) error {
	_ = ctx
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0644)
}

func commandArgs(interpreter, command string) []string {
	switch interpreter {
	case "powershell", "pwsh":
		return []string{"-Command", command}
	case "cmd":
		return []string{"/c", command}
	case "bash", "sh", "zsh":
		return []string{"-c", command}
	default:
		return []string{"-c", command}
	}
}

func scriptArgs(interpreter, path string) []string {
	switch interpreter {
	case "powershell", "pwsh":
		return []string{"-ExecutionPolicy", "Bypass", "-File", path}
	case "cmd":
		return []string{"/c", path}
	case "bash", "sh", "zsh":
		return []string{path}
	default:
		return []string{path}
	}
}

func run(ctx context.Context, interpreter, workingDir string, args []string, env map[string]string, stdout, stderr io.Writer) (int, error) {
	cmd := exec.CommandContext(ctx, interpreter, args...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	if stdout != nil {
		cmd.Stdout = stdout
	}
	if stderr != nil {
		cmd.Stderr = stderr
	}
	cmd.Env = buildEnv(env)
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}

func buildEnv(env map[string]string) []string {
	base := make(map[string]string)
	for _, e := range os.Environ() {
		k, v, _ := strings.Cut(e, "=")
		base[k] = v
	}
	for k, v := range env {
		base[k] = v
	}
	out := make([]string, 0, len(base))
	for k, v := range base {
		out = append(out, k+"="+v)
	}
	return out
}
