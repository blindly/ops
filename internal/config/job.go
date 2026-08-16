package config

// Job is the top-level job file.
type Job struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Servers     ServersSource     `yaml:"servers"`
	Defaults    Defaults          `yaml:"defaults"`
	Vars        map[string]string `yaml:"vars"`
	Env         map[string]string `yaml:"env"`
	Tasks       []Task            `yaml:"tasks"`
}

// Defaults holds per-job defaults.
type Defaults struct {
	Interpreter string `yaml:"interpreter"`
	WorkDir     string `yaml:"workdir"`
}

// Task is one step in a job.
type Task struct {
	Name         string            `yaml:"name"`
	Command      string            `yaml:"command"`
	Script       string            `yaml:"script"`
	Shell        string            `yaml:"shell"`
	Upload       string            `yaml:"upload"`
	Dest         string            `yaml:"dest"`
	Interpreter  string            `yaml:"interpreter"`
	WorkDir      string            `yaml:"workdir"`
	Env          map[string]string `yaml:"env"`
	Timeout      string            `yaml:"timeout"`
	IgnoreErrors bool              `yaml:"ignore_errors"`
	Servers      []string          `yaml:"servers"`
}

// ServersSource can be a path string or inline list.
type ServersSource struct {
	Path  string
	List  []string
	IsSet bool
}
