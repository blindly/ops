package config

// Servers is the top-level servers file.
type Servers struct {
	Servers []string `yaml:"servers"`
}

// Server holds resolved connection details from ssh_config.
type Server struct {
	Alias        string
	HostName     string
	User         string
	Port         string
	IdentityFile string
	Shell        string // not from ssh_config; provided by job default or task override
}
