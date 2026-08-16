package runner

import "time"

type TaskResult struct {
	TaskName string
	Server   string
	ExitCode int
	Stdout   string
	Stderr   string
	Error    string
	DryRun   bool
	Duration int64 // milliseconds
}

type JobResult struct {
	TaskResults   []TaskResult
	StartedAt     time.Time
	TotalDuration int64 // milliseconds
}

func (r *JobResult) HighestExitCode() int {
	max := 0
	for _, tr := range r.TaskResults {
		if tr.ExitCode > max {
			max = tr.ExitCode
		}
	}
	return max
}
