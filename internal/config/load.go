package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadJob(path string) (*Job, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// yaml.v3 does not invoke ServersSource.UnmarshalYAML for !!null when the
	// field is a value (not a pointer), so we explicitly reject `servers: null`
	// before the normal unmarshal.
	var root yaml.Node
	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil, err
	}
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		if serversNode := findMapValue(root.Content[0], "servers"); serversNode != nil && serversNode.ShortTag() == "!!null" {
			return nil, fmt.Errorf("servers path must be a non-empty string")
		}
	}
	var j Job
	if err := yaml.Unmarshal(b, &j); err != nil {
		return nil, err
	}
	if err := j.Validate(); err != nil {
		return nil, err
	}
	return &j, nil
}

func findMapValue(n *yaml.Node, key string) *yaml.Node {
	if n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(n.Content)-1; i += 2 {
		if n.Content[i].Kind == yaml.ScalarNode && n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

func LoadServers(path string) (*Servers, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Servers
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (j *Job) Validate() error {
	if len(j.Tasks) == 0 {
		return fmt.Errorf("job must have at least one task")
	}
	for i, t := range j.Tasks {
		if t.Name == "" {
			return fmt.Errorf("task %d: name is required", i)
		}
		if t.Command == "" && t.Script == "" && t.Shell == "" && t.Upload == "" {
			return fmt.Errorf("task %s: one of command, script, shell, or upload is required", t.Name)
		}
		if t.Upload != "" && t.Dest == "" {
			return fmt.Errorf("task %s: upload requires a dest", t.Name)
		}
	}
	return nil
}

// UnmarshalYAML allows `servers: ./servers.yaml` or `servers: [a, b]`.
func (s *ServersSource) UnmarshalYAML(node *yaml.Node) error {
	s.IsSet = true
	s.Path = ""
	s.List = nil
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag == "!!null" || strings.TrimSpace(node.Value) == "" {
			return fmt.Errorf("servers path must be a non-empty string")
		}
		s.Path = strings.TrimSpace(node.Value)
	case yaml.SequenceNode:
		s.List = make([]string, len(node.Content))
		for i, n := range node.Content {
			if n.Kind != yaml.ScalarNode {
				return fmt.Errorf("servers list must contain only scalar aliases")
			}
			if n.Tag == "!!null" || strings.TrimSpace(n.Value) == "" {
				return fmt.Errorf("servers list entries must be non-empty strings")
			}
			s.List[i] = strings.TrimSpace(n.Value)
		}
	default:
		return fmt.Errorf("servers must be a file path or a list of aliases")
	}
	return nil
}
