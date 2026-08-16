package ssh

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// UploadScript uploads a script to the remote host via SFTP and returns the
// remote path together with a cleanup function that removes the uploaded
// file. The interpreter determines the file extension and the temp directory
// used for the script ("powershell"/"pwsh" target Windows, everything else
// targets POSIX /tmp).
func UploadScript(client *ssh.Client, interpreter string, scriptName string, content []byte) (remotePath string, cleanup func(), err error) {
	remote, err := uploadPath(interpreter, scriptName)
	if err != nil {
		return "", nil, err
	}

	cleanup = func() {
		_, _ = Exec(context.Background(), client, cleanupCommand(interpreter, remote), nil, io.Discard, io.Discard)
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return "", nil, fmt.Errorf("sftp client: %w", err)
	}
	defer sftpClient.Close()

	f, err := sftpClient.Create(remote)
	if err != nil {
		return "", nil, fmt.Errorf("sftp create: %w", err)
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		cleanup()
		return "", nil, fmt.Errorf("sftp write: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("sftp close: %w", err)
	}

	return remote, cleanup, nil
}

func isPowerShell(interpreter string) bool {
	return interpreter == "powershell" || interpreter == "pwsh"
}

// ShellQuote wraps s in single quotes for safe use in a shell command. Any
// embedded single quotes are escaped using the standard POSIX idiom
// (close, double-quote a single quote, reopen: '"'"').
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// PowerShellQuote wraps s in single quotes for safe use in a PowerShell
// command. PowerShell escapes a single quote by doubling it (”), so any
// embedded single quote is replaced with two single quotes.
func PowerShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// cleanupCommand builds the command used to remove the uploaded script once
// it has been executed. PowerShell targets use Remove-Item; everything else
// uses POSIX rm. The remote path is shell-quoted to prevent injection via
// free-form script names.
func cleanupCommand(interpreter, remote string) string {
	if isPowerShell(interpreter) {
		return fmt.Sprintf("Remove-Item -Force %s", PowerShellQuote(remote))
	}
	return fmt.Sprintf("rm -f %s", ShellQuote(remote))
}

// uploadPath generates a unique remote path for a script given its interpreter
// and base name. PowerShell/pwsh scripts land in C:\Windows\Temp with a .ps1
// extension; all other interpreters land in /tmp with a .sh extension.
func uploadPath(interpreter, scriptName string) (string, error) {
	var ext string
	if isPowerShell(interpreter) {
		ext = ".ps1"
	} else {
		ext = ".sh"
	}

	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random suffix: %w", err)
	}
	hash := hex.EncodeToString(b)

	if isPowerShell(interpreter) {
		return fmt.Sprintf(`C:\Windows\Temp\ops-%s-%s%s`, hash, scriptName, ext), nil
	}
	return fmt.Sprintf("/tmp/ops-%s-%s%s", hash, scriptName, ext), nil
}

// UploadFile writes content to the remote path via SFTP.
func UploadFile(client *ssh.Client, remote string, content []byte) error {
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("sftp client: %w", err)
	}
	defer sftpClient.Close()

	f, err := sftpClient.Create(remote)
	if err != nil {
		return fmt.Errorf("create remote file %s: %w", remote, err)
	}
	_, err = f.Write(content)
	closeErr := f.Close()
	if err != nil {
		return fmt.Errorf("write remote file %s: %w", remote, err)
	}
	if closeErr != nil {
		return fmt.Errorf("close remote file %s: %w", remote, closeErr)
	}
	return nil
}
