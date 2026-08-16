package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const sampleJob = `---
description: example job description
servers: ./servers.yaml
defaults:
  interpreter: bash
vars:
  name: world
tasks:
  - name: say hello
    command: echo "hello {{ .name }}"
`

const sampleServers = `servers:
  - web1
  - db1
`

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "new [directory]",
		Short: "Create a sample job and servers file",
		Long:  "Create a starter jobs.yaml and servers.yaml in the given directory. Defaults to the current directory. Existing files are not overwritten.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}
			jobPath := filepath.Join(dir, "jobs.yaml")
			serversPath := filepath.Join(dir, "servers.yaml")
			if err := writeNewFile(jobPath, sampleJob); err != nil {
				return err
			}
			if err := writeNewFile(serversPath, sampleServers); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created sample files:\n  %s\n  %s\n", jobPath, serversPath)
			return nil
		},
	})
}

func writeNewFile(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check %s: %w", path, err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}
