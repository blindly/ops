package cli

import (
	"fmt"
	"os"
	"runtime"

	"github.com/blindly/ops/internal/config"
	"github.com/blindly/ops/internal/reporter"
	"github.com/blindly/ops/internal/runner"
	"github.com/blindly/ops/internal/updater"
	"github.com/blindly/ops/internal/version"
	"github.com/spf13/cobra"
)

const defaultRepo = "blindly/ops"

var rootCmd = &cobra.Command{
	Use:   "ops",
	Short: "Run shell jobs on remote servers",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	runCmd := &cobra.Command{
		Use:     "run [job-file]",
		Short:   "Run a job",
		Aliases: []string{"plan"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobPath := "jobs.yaml"
			if len(args) > 0 {
				jobPath = args[0]
			}
			job, err := config.LoadJob(jobPath)
			if err != nil {
				return fmt.Errorf("load job: %w", err)
			}

			var aliases []string
			local, _ := cmd.Flags().GetBool("local")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			dryRun = dryRun || cmd.CalledAs() == "plan"
			serversFile, _ := cmd.Flags().GetString("servers")
			serverList, _ := cmd.Flags().GetStringSlice("server-list")

			if local {
				if serversFile != "" || len(serverList) > 0 {
					return fmt.Errorf("--local cannot be used with --servers or --server-list")
				}
			} else {
				if job.Servers.IsSet {
					if len(job.Servers.List) > 0 {
						aliases = job.Servers.List
					} else if job.Servers.Path != "" {
						s, err := config.LoadServers(job.Servers.Path)
						if err != nil {
							return fmt.Errorf("load servers: %w", err)
						}
						aliases = s.Servers
					}
				}
				if len(serverList) > 0 {
					aliases = serverList
				}
				if serversFile != "" {
					s, err := config.LoadServers(serversFile)
					if err != nil {
						return fmt.Errorf("load servers file: %w", err)
					}
					aliases = s.Servers
				}
				if len(aliases) == 0 {
					if !dryRun {
						return fmt.Errorf("no servers specified")
					}
				}
			}

			var servers []config.Server
			if local {
				servers = []config.Server{{Alias: "local"}}
			} else if dryRun && len(aliases) == 0 {
			servers = []config.Server{{Alias: "-"}}
			} else {
				sshConfigPath, _ := cmd.Flags().GetString("ssh-config")
				servers, err = config.ResolveServers(aliases, sshConfigPath)
				if err != nil {
					return fmt.Errorf("resolve servers: %w", err)
				}
			}

			workers, _ := cmd.Flags().GetInt("workers")
			failFast, _ := cmd.Flags().GetBool("fail-fast")
			keep, _ := cmd.Flags().GetBool("keep")
			ignoreHostKey, _ := cmd.Flags().GetBool("insecure-ignore-hostkey")
			output, _ := cmd.Flags().GetString("output")
			stream, _ := cmd.Flags().GetBool("stream")
			quiet, _ := cmd.Flags().GetBool("quiet")

			if stream && output == "json" {
				return fmt.Errorf("--stream cannot be used with --output json")
			}

			progress := cmd.OutOrStderr()
			if quiet || stream || output == "json" {
				progress = nil
			}

			if job.Env == nil {
				job.Env = make(map[string]string)
			}
			if job.Vars == nil {
				job.Vars = make(map[string]string)
			}
			envVars, _ := cmd.Flags().GetStringToString("env")
			for k, v := range envVars {
				job.Env[k] = v
			}
			varVars, _ := cmd.Flags().GetStringToString("var")
			for k, v := range varVars {
				job.Vars[k] = v
			}

			res, err := runner.Run(cmd.Context(), runner.Options{
				Job:           job,
				Servers:       servers,
				Workers:       workers,
				FailFast:      failFast,
				Keep:          keep,
				IgnoreHostKey: ignoreHostKey,
				Stream:        stream,
				Progress:      progress,
				Local:         local,
				DryRun:        dryRun,
			})
			if err != nil && !failFast {
				return err
			}

			if output == "json" {
				if err := reporter.PrintJSON(cmd.OutOrStdout(), res); err != nil {
					return err
				}
			} else if dryRun {
				reporter.PrintDryRun(cmd.OutOrStdout(), res)
			} else {
				reporter.PrintTextWithOpts(cmd.OutOrStdout(), res, !quiet && !stream)
			}

			if err != nil {
				return err
			}
			os.Exit(res.HighestExitCode())
			return nil
		},
	}
	runCmd.Flags().StringP("servers", "s", "", "servers YAML file")
	runCmd.Flags().StringSliceP("server-list", "S", nil, "inline server aliases")
	runCmd.Flags().String("ssh-config", "", "path to ssh config file")
	runCmd.Flags().IntP("workers", "w", 10, "concurrency")
	runCmd.Flags().Bool("fail-fast", false, "stop on first failure")
	runCmd.Flags().Bool("keep", false, "keep uploaded scripts")
	runCmd.Flags().Bool("insecure-ignore-hostkey", false, "skip host key verification (insecure)")
	runCmd.Flags().StringP("output", "o", "text", "output format (text|json)")
	runCmd.Flags().BoolP("quiet", "q", false, "suppress command output (show summary table only)")
	runCmd.Flags().Bool("stream", false, "stream command output as it runs")
	runCmd.Flags().Bool("local", false, "run tasks on the local machine")
	runCmd.Flags().Bool("dry-run", false, "show what would run without executing")
	runCmd.Flags().StringToStringP("env", "e", nil, "environment variables")
	runCmd.Flags().StringToString("var", nil, "job variables")
	rootCmd.AddCommand(runCmd)

	rootCmd.AddCommand(&cobra.Command{
		Use:   "validate [job-file]",
		Short: "Validate a job file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobPath := "jobs.yaml"
			if len(args) > 0 {
				jobPath = args[0]
			}
			_, err := config.LoadJob(jobPath)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "valid")
			return nil
		},
	})

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update ops to the latest release",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, _ := cmd.Flags().GetString("repo")
			if repo == "" {
				repo = defaultRepo
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Current version: %s\n", version.Version)

			release, err := updater.FetchLatestRelease(repo)
			if err != nil {
				return fmt.Errorf("check for updates: %w", err)
			}

			force, _ := cmd.Flags().GetBool("force")
			if release.TagName == version.Version && !force {
				fmt.Fprintf(cmd.OutOrStdout(), "Already up to date (%s).\n", version.Version)
				return nil
			}

			if release.TagName == "" {
				return fmt.Errorf("no release found")
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Latest version:  %s\n", release.TagName)

			checkOnly, _ := cmd.Flags().GetBool("check")
			if checkOnly {
				fmt.Fprintln(cmd.OutOrStdout(), "Update available. Run without --check to install.")
				return nil
			}

			asset := updater.FindAsset(release)
			if asset == nil {
				return fmt.Errorf("no binary found for %s/%s in release %s", osForDisplay(), archForDisplay(), release.TagName)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Downloading %s...\n", asset.Name)
			tmpPath, err := updater.DownloadAsset(asset)
			if err != nil {
				return err
			}
			defer os.Remove(tmpPath)

			fmt.Fprintln(cmd.OutOrStdout(), "Installing...")
			if err := updater.ReplaceBinary(tmpPath); err != nil {
				return fmt.Errorf("replace binary: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updated to %s.\n", release.TagName)
			return nil
		},
	}
	updateCmd.Flags().String("repo", "", "GitHub repo (owner/name) to update from")
	updateCmd.Flags().Bool("check", false, "check for updates without installing")
	updateCmd.Flags().Bool("force", false, "force update even if already on the latest version")
	rootCmd.AddCommand(updateCmd)

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the ops version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), version.Version)
			return nil
		},
	}
	rootCmd.AddCommand(versionCmd)
}

func osForDisplay() string {
	return runtime.GOOS
}

func archForDisplay() string {
	return runtime.GOARCH
}
