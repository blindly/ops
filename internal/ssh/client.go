package ssh

import (
	"fmt"
	"strconv"
	"time"

	"github.com/blindly/ops/internal/config"
	"golang.org/x/crypto/ssh"
)

func NewClient(server config.Server, opts AuthOptions) (*ssh.Client, error) {
	methods, err := buildAuthMethods(opts)
	if err != nil {
		return nil, err
	}
	hostKey, err := hostKeyCallback(opts)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(server.Port)
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ClientConfig{
		User:            server.User,
		Auth:            methods,
		HostKeyCallback: hostKey,
		Timeout:         30 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", server.HostName, port)
	return ssh.Dial("tcp", addr, cfg)
}
