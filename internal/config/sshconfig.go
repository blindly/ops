package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	ssh_config "github.com/kevinburke/ssh_config"
)

// ResolveServers reads ssh_config and returns Server structs for aliases.
func ResolveServers(aliases []string, configPath string) ([]Server, error) {
	var cfg *ssh_config.Config
	var cfgErr error
	var cfgOpen bool
	var currentUser *user.User
	var currentUserErr error

	var out []Server
	for _, alias := range aliases {
		if alias == "local" {
			out = append(out, Server{Alias: "local", HostName: "local"})
			continue
		}

		if !cfgOpen {
			if configPath == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return nil, err
				}
				configPath = filepath.Join(home, ".ssh", "config")
			}
			f, err := os.Open(configPath)
			if err != nil {
				return nil, fmt.Errorf("open ssh config: %w", err)
			}
			cfg, cfgErr = ssh_config.Decode(f)
			_ = f.Close()
			if cfgErr != nil {
				return nil, fmt.Errorf("parse ssh config: %w", cfgErr)
			}
			cfgOpen = true
		}

		username, _ := cfg.Get(alias, "User")
		if username == "" {
			if currentUser == nil && currentUserErr == nil {
				currentUser, currentUserErr = user.Current()
			}
			if currentUserErr != nil {
				return nil, fmt.Errorf("get current user: %w", currentUserErr)
			}
			username = currentUser.Username
		}
		hostname, _ := cfg.Get(alias, "HostName")
		if hostname == "" {
			hostname = alias
		}
		portStr, _ := cfg.Get(alias, "Port")
		if portStr == "" {
			portStr = "22"
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("server %s: invalid port %q: %w", alias, portStr, err)
		}
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("server %s: invalid port %d", alias, port)
		}
		key, _ := cfg.Get(alias, "IdentityFile")

		out = append(out, Server{
			Alias:        alias,
			HostName:     hostname,
			User:         username,
			Port:         portStr,
			IdentityFile: key,
		})
	}
	return out, nil
}
