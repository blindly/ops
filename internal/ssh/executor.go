package ssh

import (
	"context"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
)

type executor struct {
	client *ssh.Client
}

// NewExecutor wraps an SSH client as a runner.Executor.
func NewExecutor(client *ssh.Client) *executor {
	return &executor{client: client}
}

// RunCommand runs a single command string through the interpreter.
func (e *executor) RunCommand(ctx context.Context, interpreter, workingDir, command string, env map[string]string, stdout, stderr io.Writer) (int, error) {
	if workingDir != "" {
		if interpreter == "powershell" || interpreter == "pwsh" {
			command = "Set-Location -LiteralPath " + PowerShellQuote(workingDir) + "; " + command
		} else if interpreter != "cmd" {
			command = "cd " + ShellQuote(workingDir) + " && " + command
		}
	}
	cmd := fmt.Sprintf("%s -c %q", interpreter, command)
	if interpreter == "powershell" || interpreter == "pwsh" || interpreter == "cmd" {
		cmd = fmt.Sprintf("%s -Command %q", interpreter, command)
	}
	res, err := Exec(ctx, e.client, cmd, env, stdout, stderr)
	if err != nil {
		return 0, err
	}
	return res.ExitCode, nil
}

// RunScript uploads a script and executes it.
func (e *executor) RunScript(ctx context.Context, interpreter, name, workingDir string, data []byte, env map[string]string, stdout, stderr io.Writer, keep bool) (int, error) {
	remote, cleanup, err := UploadScript(e.client, interpreter, name, data)
	if err != nil {
		return 0, err
	}
	if !keep {
		defer cleanup()
	}
	var cmd string
	if workingDir != "" {
		if interpreter == "powershell" || interpreter == "pwsh" {
			cmd = fmt.Sprintf("Set-Location -LiteralPath %s; %s -ExecutionPolicy Bypass -File %s", PowerShellQuote(workingDir), interpreter, PowerShellQuote(remote))
		} else if interpreter == "cmd" {
			cmd = fmt.Sprintf("%s %s", interpreter, ShellQuote(remote))
		} else {
			cmd = fmt.Sprintf("cd %s && %s %s", ShellQuote(workingDir), interpreter, ShellQuote(remote))
		}
	} else {
		cmd = fmt.Sprintf("%s %s", interpreter, ShellQuote(remote))
		if interpreter == "powershell" || interpreter == "pwsh" {
			cmd = fmt.Sprintf("%s -ExecutionPolicy Bypass -File %s", interpreter, PowerShellQuote(remote))
		}
	}
	res, err := Exec(ctx, e.client, cmd, env, stdout, stderr)
	if err != nil {
		return 0, err
	}
	return res.ExitCode, nil
}

// UploadFile writes a file to the remote host.
func (e *executor) UploadFile(ctx context.Context, dest string, data []byte) error {
	_ = ctx
	return UploadFile(e.client, dest, data)
}

// Close closes the underlying SSH client.
func (e *executor) Close() error {
	return e.client.Close()
}
