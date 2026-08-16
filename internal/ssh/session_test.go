package ssh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// startTestSSHServer launches an in-process SSH server that accepts a single
// connection and a session channel. The simulated "command" runs indefinitely
// until the client signals the session, at which point the channel is closed.
// It returns the listener address and a shutdown function.
func startTestSSHServer(t *testing.T) (string, func()) {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}

	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
		if err != nil {
			return
		}
		defer sshConn.Close()
		go ssh.DiscardRequests(reqs)

		for newChan := range chans {
			if newChan.ChannelType() != "session" {
				newChan.Reject(ssh.UnknownChannelType, "only session")
				continue
			}
			channel, requests, err := newChan.Accept()
			if err != nil {
				continue
			}
			go func(in <-chan *ssh.Request) {
				for req := range in {
					switch req.Type {
					case "exec":
						// Acknowledge the command. The "command" is simulated as
						// an indefinitely running process by keeping the channel
						// open until a signal arrives.
						if req.WantReply {
							req.Reply(true, nil)
						}
					case "env":
						if req.WantReply {
							req.Reply(true, nil)
						}
					case "signal":
						// Simulate the remote process being killed by the signal:
						// close the channel so the client's session.Run unblocks.
						channel.Close()
						if req.WantReply {
							req.Reply(true, nil)
						}
						return
					default:
						if req.WantReply {
							req.Reply(false, nil)
						}
					}
				}
			}(requests)
		}
	}()

	cleanup := func() {
		listener.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
	return listener.Addr().String(), cleanup
}

// TestExec_ContextCancel verifies that Exec returns context.Canceled when the
// passed context is cancelled while a command is still running on the remote.
func TestExec_ContextCancel(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")

	addr, shutdown := startTestSSHServer(t)
	defer shutdown()

	// Build a client config with no auth (server uses NoClientAuth) and an
	// insecure host key callback since we own the ephemeral host key.
	clientConfig := &ssh.ClientConfig{
		User:            "test",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		t.Fatalf("dial test server: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately so Exec observes ctx.Done() while the long sleep is
	// still running on the server.
	cancel()

	resultCh := make(chan error, 1)
	go func() {
		_, err := Exec(ctx, client, "sleep 30", nil, io.Discard, io.Discard)
		resultCh <- err
	}()

	select {
	case err := <-resultCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Exec did not return after context cancellation")
	}
}
