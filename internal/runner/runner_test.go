package runner

import (
	"errors"
	"testing"

	"github.com/blindly/ops/internal/config"
)

func TestHighestExitCode(t *testing.T) {
	tests := []struct {
		name    string
		results []TaskResult
		want    int
	}{
		{
			name:    "empty",
			results: nil,
			want:    0,
		},
		{
			name: "all zero",
			results: []TaskResult{
				{TaskName: "a", ExitCode: 0},
				{TaskName: "b", ExitCode: 0},
			},
			want: 0,
		},
		{
			name: "mixed",
			results: []TaskResult{
				{TaskName: "a", ExitCode: 0},
				{TaskName: "b", ExitCode: 2},
				{TaskName: "c", ExitCode: 1},
			},
			want: 2,
		},
		{
			name: "negative ignored",
			results: []TaskResult{
				{TaskName: "a", ExitCode: -1},
				{TaskName: "b", ExitCode: 5},
			},
			want: 5,
		},
		{
			name: "all negative yields zero",
			results: []TaskResult{
				{TaskName: "a", ExitCode: -1},
				{TaskName: "b", ExitCode: -3},
			},
			want: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jr := &JobResult{TaskResults: tc.results}
			got := jr.HighestExitCode()
			if got != tc.want {
				t.Fatalf("HighestExitCode = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestHighestExitCode_Pointer(t *testing.T) {
	// Method should be callable on a pointer receiver and not panic with nil slice.
	jr := &JobResult{}
	if got := jr.HighestExitCode(); got != 0 {
		t.Fatalf("HighestExitCode on nil slice = %d, want 0", got)
	}
}

func TestFilterServers(t *testing.T) {
	servers := []config.Server{
		{Alias: "web1"},
		{Alias: "web2"},
		{Alias: "db1"},
	}

	t.Run("subset match", func(t *testing.T) {
		got := filterServers(servers, []string{"web1", "db1"})
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if got[0].Alias != "web1" || got[1].Alias != "db1" {
			t.Fatalf("order mismatch: %+v", got)
		}
	})

	t.Run("no match", func(t *testing.T) {
		got := filterServers(servers, []string{"missing"})
		if len(got) != 0 {
			t.Fatalf("len = %d, want 0", len(got))
		}
	})

	t.Run("all match", func(t *testing.T) {
		got := filterServers(servers, []string{"web1", "web2", "db1"})
		if len(got) != 3 {
			t.Fatalf("len = %d, want 3", len(got))
		}
	})

	t.Run("empty names yields empty", func(t *testing.T) {
		got := filterServers(servers, nil)
		if len(got) != 0 {
			t.Fatalf("len = %d, want 0", len(got))
		}
	})

	t.Run("empty servers yields empty", func(t *testing.T) {
		got := filterServers(nil, []string{"web1"})
		if len(got) != 0 {
			t.Fatalf("len = %d, want 0", len(got))
		}
	})
}

func TestMergeEnv(t *testing.T) {
	t.Run("task overrides global", func(t *testing.T) {
		global := map[string]string{"A": "1", "B": "2"}
		task := map[string]string{"B": "22", "C": "3"}
		got := mergeEnv(global, task)
		if got["A"] != "1" {
			t.Errorf("A = %q, want %q", got["A"], "1")
		}
		if got["B"] != "22" {
			t.Errorf("B = %q, want %q (task should override)", got["B"], "22")
		}
		if got["C"] != "3" {
			t.Errorf("C = %q, want %q", got["C"], "3")
		}
	})

	t.Run("global only", func(t *testing.T) {
		got := mergeEnv(map[string]string{"X": "g"}, nil)
		if got["X"] != "g" {
			t.Errorf("X = %q, want %q", got["X"], "g")
		}
		if len(got) != 1 {
			t.Errorf("len = %d, want 1", len(got))
		}
	})

	t.Run("task only", func(t *testing.T) {
		got := mergeEnv(nil, map[string]string{"Y": "t"})
		if got["Y"] != "t" {
			t.Errorf("Y = %q, want %q", got["Y"], "t")
		}
	})

	t.Run("both empty returns non-nil empty map", func(t *testing.T) {
		got := mergeEnv(nil, nil)
		if got == nil {
			t.Fatal("got nil map, want non-nil")
		}
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("does not mutate inputs", func(t *testing.T) {
		global := map[string]string{"A": "1"}
		task := map[string]string{"B": "2"}
		_ = mergeEnv(global, task)
		if len(global) != 1 || global["A"] != "1" {
			t.Errorf("global mutated: %+v", global)
		}
		if len(task) != 1 || task["B"] != "2" {
			t.Errorf("task mutated: %+v", task)
		}
	})
}

func TestErrString(t *testing.T) {
	if got := errString(nil); got != "" {
		t.Fatalf("errString(nil) = %q, want empty", got)
	}
	err := errors.New("boom")
	if got := errString(err); got != "boom" {
		t.Fatalf("errString(err) = %q, want %q", got, "boom")
	}
}

func TestRenderEnv(t *testing.T) {
	vars := map[string]string{"version": "1.24"}
	env := map[string]string{
		"VERSION":      "{{ .version }}",
		"STATIC":       "literal",
		"WITH_VERSION": "app-{{ .version }}",
	}
	got, err := renderEnv(env, vars)
	if err != nil {
		t.Fatalf("renderEnv: %v", err)
	}
	want := map[string]string{
		"VERSION":      "1.24",
		"STATIC":       "literal",
		"WITH_VERSION": "app-1.24",
	}
	for k, w := range want {
		if got[k] != w {
			t.Fatalf("env[%q] = %q, want %q", k, got[k], w)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("renderEnv returned %d values, want %d", len(got), len(want))
	}
}
