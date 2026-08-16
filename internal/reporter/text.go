package reporter

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/blindly/ops/internal/runner"
	"github.com/fatih/color"
)

func PrintText(w io.Writer, res *runner.JobResult) {
	PrintTextWithOpts(w, res, true)
}

func PrintTextWithOpts(w io.Writer, res *runner.JobResult, showOutput bool) {
	red := color.New(color.FgRed)
	green := color.New(color.FgGreen)
	dim := color.New(color.Faint)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TASK\tSERVER\tSTATUS\tEXIT\tDURATION")
	for _, r := range res.TaskResults {
		status := "ok"
		if r.ExitCode != 0 || r.Error != "" {
			status = "fail"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%dms\n", r.TaskName, r.Server, status, r.ExitCode, r.Duration)
	}
	tw.Flush()

	if !showOutput {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Started: %s\n", res.StartedAt.Format(time.RFC3339))
		fmt.Fprintf(w, "Duration: %dms\n", res.TotalDuration)
		return
	}

	for _, r := range res.TaskResults {
		if r.Stdout == "" && r.Stderr == "" && r.Error == "" {
			continue
		}
		fmt.Fprintln(w)
		dim.Fprintf(w, "── %s [%s] ──\n", r.TaskName, r.Server)

		if r.Error != "" {
			red.Fprintf(w, "error: %s\n", r.Error)
		}

		if r.Stdout != "" {
			if r.ExitCode != 0 || r.Error != "" {
				fmt.Fprint(w, r.Stdout)
			} else {
				green.Fprint(w, r.Stdout)
			}
		}

		if r.Stderr != "" {
			red.Fprint(w, r.Stderr)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Started: %s\n", res.StartedAt.Format(time.RFC3339))
	fmt.Fprintf(w, "Duration: %dms\n", res.TotalDuration)
}
func PrintDryRun(w io.Writer, res *runner.JobResult) {
	dim := color.New(color.Faint)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TASK\tSERVER\tSTATUS\tPLAN")
	for _, r := range res.TaskResults {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.TaskName, r.Server, "plan", r.Stdout)
	}
	tw.Flush()

	for _, r := range res.TaskResults {
		if r.Stderr == "" && r.Error == "" {
			continue
		}
		fmt.Fprintln(w)
		dim.Fprintf(w, "── %s [%s] ──\n", r.TaskName, r.Server)
		if r.Error != "" {
			color.New(color.FgRed).Fprintf(w, "error: %s\n", r.Error)
		}
		if r.Stderr != "" {
			color.New(color.FgRed).Fprint(w, r.Stderr)
		}
	}

}
