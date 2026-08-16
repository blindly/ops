package ssh

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

type AuthOptions struct {
	IdentityFile   string
	Password       string
	IgnoreHostKey  bool
	KnownHostsPath string
}

func buildAuthMethods(opts AuthOptions) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	if opts.Password != "" {
		methods = append(methods, ssh.Password(opts.Password))
	}

	if opts.IdentityFile != "" {
		keyPath := opts.IdentityFile
		if strings.HasPrefix(keyPath, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			keyPath = filepath.Join(home, keyPath[2:])
		}
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("read identity file: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		conn, err := net.Dial("unix", sock)
		if err == nil {
			methods = append(methods, ssh.PublicKeysCallback(agent.NewClient(conn).Signers))
		}
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no authentication methods available")
	}
	return methods, nil
}

func hostKeyCallback(opts AuthOptions) (ssh.HostKeyCallback, error) {
	if opts.IgnoreHostKey {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	path := opts.KnownHostsPath
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".ssh", "known_hosts")
	}
	cb, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("known_hosts: %w", err)
	}
	return cb, nil
}
