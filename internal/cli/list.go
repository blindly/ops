package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/blindly/ops/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:     "list [directory]",
		Aliases: []string{"ls"},
		Short:   "List job files in a directory",
		Long:    "List job files in the current or given directory, showing each file and description. Use -r to search subdirectories.",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			dir = filepath.Clean(dir)

			recursive, _ := cmd.Flags().GetBool("recursive")

			type item struct {
				file string
				desc string
			}
			var items []item

			if recursive {
				err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if d.IsDir() {
						if d.Name() != filepath.Base(dir) && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules" || d.Name() == "vendor") {
							return filepath.SkipDir
						}
						return nil
					}
					ext := filepath.Ext(d.Name())
					if ext != ".yaml" && ext != ".yml" {
						return nil
					}
					job, err := config.LoadJob(path)
					if err != nil {
						return nil
					}
					rel, _ := filepath.Rel(dir, path)
					if rel == "" {
						rel = d.Name()
					}
					items = append(items, item{rel, job.Description})
					return nil
				})
				if err != nil {
					return fmt.Errorf("walk directory: %w", err)
				}
			} else {
				entries, err := os.ReadDir(dir)
				if err != nil {
					return fmt.Errorf("read directory: %w", err)
				}
				for _, e := range entries {
					if e.IsDir() {
						continue
					}
					ext := filepath.Ext(e.Name())
					if ext != ".yaml" && ext != ".yml" {
						continue
					}
					path := filepath.Join(dir, e.Name())
					job, err := config.LoadJob(path)
					if err != nil {
						continue
					}
					rel, _ := filepath.Rel(dir, path)
					if rel == "" {
						rel = e.Name()
					}
					items = append(items, item{rel, job.Description})
				}
			}

			if len(items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no jobs found")
				return nil
			}

			for i := range items {
				if items[i].desc == "" {
					items[i].desc = "—"
				}
			}

			fileHeader := "FILE"
			if recursive {
				fileHeader = "PATH"
			}
			descHeader := "DESCRIPTION"

			fileW := utf8.RuneCountInString(fileHeader)
			for _, it := range items {
				if w := utf8.RuneCountInString(it.file); w > fileW {
					fileW = w
				}
			}

			bold := color.New(color.Bold)
			bold.Fprintf(cmd.OutOrStdout(), "%-*s  %s\n", fileW, fileHeader, descHeader)
			for _, it := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%-*s  %s\n", fileW, it.file, it.desc)
			}
			return nil
		},
	}
	cmd.Flags().BoolP("recursive", "r", false, "recursively list job files in subdirectories")
	rootCmd.AddCommand(cmd)
}
