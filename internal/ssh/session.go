package ssh

import (
	"context"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
)

// ExecResult holds the exit status of a remote command.
type ExecResult struct {
	ExitCode int
}

// Exec runs cmd on the remote host and copies stdout/stderr to the supplied writers.
func Exec(ctx context.Context, client *ssh.Client, cmd string, env map[string]string, stdout, stderr io.Writer) (*ExecResult, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	for k, v := range env {
		if err := session.Setenv(k, v); err != nil {
			// Some servers reject env; continue on error.
			_ = err
		}
	}

	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	session.Stdout = stdout
	session.Stderr = stderr

	errCh := make(chan error, 1)
	go func() { errCh <- session.Run(cmd) }()

	select {
	case err := <-errCh:
		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				return &ExecResult{ExitCode: exitErr.ExitStatus()}, nil
			}
			return nil, fmt.Errorf("run: %w", err)
		}
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		<-errCh // wait for run to finish
		return nil, ctx.Err()
	}

	return &ExecResult{ExitCode: 0}, nil
}
