package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blindly/ops/internal/config"
	"github.com/blindly/ops/internal/local"
	"github.com/blindly/ops/internal/ssh"
	"github.com/blindly/ops/internal/template"
)

type Options struct {
	Job           *config.Job
	Servers       []config.Server
	Workers       int
	FailFast      bool
	Keep          bool
	IgnoreHostKey bool
	Stream        bool
	Progress      io.Writer
	Local         bool
	DryRun        bool
}

// Executor abstracts the transport used to run a task.
type Executor interface {
	RunCommand(ctx context.Context, interpreter, workingDir, command string, env map[string]string, stdout, stderr io.Writer) (int, error)
	RunScript(ctx context.Context, interpreter, name, workingDir string, data []byte, env map[string]string, stdout, stderr io.Writer, keep bool) (int, error)
	UploadFile(ctx context.Context, dest string, data []byte) error
}

func Run(ctx context.Context, opts Options) (*JobResult, error) {
	if opts.Workers <= 0 {
		opts.Workers = 10
	}
	executors := make(map[string]Executor)
	var mu sync.Mutex

	switch {
	case opts.DryRun:
		for _, s := range opts.Servers {
			executors[s.Alias] = nil
		}
	case opts.Local:
		for _, s := range opts.Servers {
			executors[s.Alias] = local.NewExecutor()
		}
	default:
		// Open all clients up front so failures are detected early.
		for _, s := range opts.Servers {
			if s.Alias == "local" {
				executors[s.Alias] = local.NewExecutor()
				continue
			}
			c, e := ssh.NewClient(s, ssh.AuthOptions{
				IdentityFile:  s.IdentityFile,
				IgnoreHostKey: opts.IgnoreHostKey,
			})
			var ex Executor
			if e == nil {
				ex = ssh.NewExecutor(c)
			}
			mu.Lock()
			executors[s.Alias] = ex
			mu.Unlock()
		}
	}
	defer func() {
		for _, ex := range executors {
			if c, ok := ex.(io.Closer); ok {
				_ = c.Close()
			}
		}
	}()
	start := time.Now()

	var total int64
	for _, task := range opts.Job.Tasks {
		taskServers := opts.Servers
		if len(task.Servers) > 0 {
			taskServers = filterServers(opts.Servers, task.Servers)
		}
		total += int64(len(taskServers))
	}

	var completed int64
	var progressWg sync.WaitGroup
	var progressDone chan struct{}
	if opts.Progress != nil && total > 0 {
		progressDone = make(chan struct{})
		progressWg.Add(1)
		go func() {
			defer progressWg.Done()
			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-progressDone:
					_, _ = fmt.Fprint(opts.Progress, "\r\x1b[2K")
					return
				case <-ticker.C:
					drawProgress(opts.Progress, atomic.LoadInt64(&completed), total)
				}
			}
		}()
		defer func() {
			if progressDone != nil {
				close(progressDone)
				progressWg.Wait()
			}
		}()
	}

	var results []TaskResult
	var resMu sync.Mutex
	var outMu sync.Mutex
	for _, task := range opts.Job.Tasks {
		taskServers := opts.Servers
		if len(task.Servers) > 0 {
			taskServers = filterServers(opts.Servers, task.Servers)
		}

		var wg sync.WaitGroup
		sem := make(chan struct{}, opts.Workers)
		cancel := make(chan struct{})
		var once sync.Once

		for _, server := range taskServers {
			wg.Add(1)
			sem <- struct{}{}
			go func(srv config.Server, t config.Task) {
				defer wg.Done()
				defer func() { <-sem }()
				defer func() { atomic.AddInt64(&completed, 1) }()

				select {
				case <-cancel:
					resMu.Lock()
					results = append(results, TaskResult{
						TaskName: t.Name,
						Server:   srv.Alias,
						Error:    "skipped due to fail-fast",
					})
					resMu.Unlock()
					return
				default:
				}

				exec := executors[srv.Alias]
				if exec == nil && !opts.DryRun {
					resMu.Lock()
					results = append(results, TaskResult{
						TaskName: t.Name,
						Server:   srv.Alias,
						DryRun:   opts.DryRun,
						Error:    "connection failed",
					})
					resMu.Unlock()
					if opts.FailFast {
						once.Do(func() { close(cancel) })
					}
					return
				}

				start := time.Now()
				exit, stdout, stderr, runErr := runTask(ctx, exec, srv, opts.Job, t, opts.Keep, opts.Stream, opts.DryRun, &outMu)
				dur := time.Since(start).Milliseconds()

				resMu.Lock()
				results = append(results, TaskResult{
					TaskName: t.Name,
					Server:   srv.Alias,
					DryRun:   opts.DryRun,
					ExitCode: exit,
					Stdout:   stdout,
					Stderr:   stderr,
					Error:    errString(runErr),
					Duration: dur,
				})
				resMu.Unlock()

				if opts.FailFast && (exit != 0 || runErr != nil) {
					once.Do(func() { close(cancel) })
				}
			}(server, task)
		}
		wg.Wait()

		// Check if we should stop.
		if opts.FailFast {
			for _, r := range results[len(results)-len(taskServers):] {
				if r.ExitCode != 0 || r.Error != "" {
					return &JobResult{TaskResults: results, StartedAt: start, TotalDuration: time.Since(start).Milliseconds()}, fmt.Errorf("fail-fast: task %s failed on %s", r.TaskName, r.Server)
				}
			}
		}
	}

	return &JobResult{TaskResults: results, StartedAt: start, TotalDuration: time.Since(start).Milliseconds()}, nil
}

func runTask(ctx context.Context, exec Executor, srv config.Server, job *config.Job, task config.Task, keep bool, stream bool, dryRun bool, mu *sync.Mutex) (int, string, string, error) {
	interpreter := task.Interpreter
	if interpreter == "" {
		interpreter = job.Defaults.Interpreter
	}
	if interpreter == "" {
		interpreter = "bash"
	}
	workingDir := task.WorkDir
	if workingDir == "" {
		workingDir = job.Defaults.WorkDir
	}

	env, err := renderEnv(mergeEnv(job.Env, task.Env), job.Vars)
	if err != nil {
		return 0, "", "", fmt.Errorf("task %s: %w", task.Name, err)
	}
	vars := job.Vars

	var wout, werr io.Writer
	var outBuf, errBuf bytes.Buffer
	if !dryRun && stream {
		wout = &lockedWriter{w: os.Stdout, mu: mu}
		werr = &lockedWriter{w: os.Stderr, mu: mu}
		fmt.Fprintf(wout, "── %s [%s] ──\n", task.Name, srv.Alias)
	} else {
		wout = &outBuf
		werr = &errBuf
	}

	switch {
	case task.Command != "":
		rendered, err := template.Render(task.Command, vars)
		if err != nil {
			return 0, "", "", err
		}
		if dryRun {
			fmt.Fprintf(wout, "command: %s", rendered)
			return 0, outBuf.String(), errBuf.String(), nil
		}
		exit, err := exec.RunCommand(ctx, interpreter, workingDir, rendered, env, wout, werr)
		if err != nil {
			return 0, "", "", err
		}
		return exit, outBuf.String(), errBuf.String(), nil

	case task.Shell != "":
		rendered, err := template.Render(task.Shell, vars)
		if err != nil {
			return 0, "", "", err
		}
		if dryRun {
			fmt.Fprintf(wout, "shell: %d bytes", len(rendered))
			return 0, outBuf.String(), errBuf.String(), nil
		}
		exit, err := exec.RunScript(ctx, interpreter, task.Name, workingDir, []byte(rendered), env, wout, werr, keep)
		if err != nil {
			return 0, "", "", err
		}
		return exit, outBuf.String(), errBuf.String(), nil

	case task.Script != "":
		if dryRun {
			fmt.Fprintf(wout, "script: %s", task.Script)
			return 0, outBuf.String(), errBuf.String(), nil
		}
		b, err := os.ReadFile(task.Script)
		if err != nil {
			return 0, "", "", err
		}
		exit, err := exec.RunScript(ctx, interpreter, task.Name, workingDir, b, env, wout, werr, keep)
		if err != nil {
			return 0, "", "", err
		}
		return exit, outBuf.String(), errBuf.String(), nil

	case task.Upload != "":
		if dryRun {
			fmt.Fprintf(wout, "upload: %s -> %s", task.Upload, task.Dest)
			return 0, outBuf.String(), errBuf.String(), nil
		}
		b, err := os.ReadFile(task.Upload)
		if err != nil {
			return 0, "", "", err
		}
		if err := exec.UploadFile(ctx, task.Dest, b); err != nil {
			return 0, "", "", err
		}
		fmt.Fprintf(wout, "uploaded %s -> %s\n", task.Upload, task.Dest)
		return 0, outBuf.String(), errBuf.String(), nil
	}

	return 0, "", "", fmt.Errorf("no runnable content in task %s", task.Name)
}

func mergeEnv(global, task map[string]string) map[string]string {
	out := make(map[string]string)
	for k, v := range global {
		out[k] = v
	}
	for k, v := range task {
		out[k] = v
	}
	return out
}

func renderEnv(env, vars map[string]string) (map[string]string, error) {
	if len(env) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(env))
	for k, v := range env {
		rv, err := template.Render(v, vars)
		if err != nil {
			return nil, fmt.Errorf("render env %s: %w", k, err)
		}
		out[k] = rv
	}
	return out, nil
}

func filterServers(servers []config.Server, names []string) []config.Server {
	allowed := make(map[string]struct{}, len(names))
	for _, n := range names {
		allowed[n] = struct{}{}
	}
	var out []config.Server
	for _, s := range servers {
		if _, ok := allowed[s.Alias]; ok {
			out = append(out, s)
		}
	}
	return out
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func drawProgress(w io.Writer, completed, total int64) {
	if total == 0 {
		return
	}
	const width = 30
	filled := int(float64(width) * float64(completed) / float64(total))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	_, _ = fmt.Fprintf(w, "\r\x1b[2K[%s] %d/%d", bar, completed, total)
}

type lockedWriter struct {
	w  io.Writer
	mu *sync.Mutex
}

func (lw *lockedWriter) Write(p []byte) (int, error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	return lw.w.Write(p)
}
