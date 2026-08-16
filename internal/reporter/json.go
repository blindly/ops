package reporter

import (
	"encoding/json"
	"io"
	"time"

	"github.com/blindly/ops/internal/runner"
)

func PrintJSON(w io.Writer, res *runner.JobResult) error {
	return json.NewEncoder(w).Encode(struct {
		StartedAt string              `json:"startedAt"`
		Duration  int64               `json:"durationMs"`
		Tasks     []runner.TaskResult `json:"tasks"`
	}{
		StartedAt: res.StartedAt.Format(time.RFC3339),
		Duration:  res.TotalDuration,
		Tasks:     res.TaskResults,
	})
}
