// Package tunnel provides SSH tunneling support for database connections.
package tunnel

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// Config holds SSH tunnel configuration.
type Config struct {
	// SSH server settings
	Host       string `json:"host"`
	Port       int    `json:"port"` // default 22
	User       string `json:"user"`
	AuthMethod string `json:"auth_method"` // password, ssh-key

	// Authentication
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`

	// Target database
	RemoteHost string `json:"remote_host"`
	RemotePort int    `json:"remote_port"`

	// Timeouts
	DialTimeout    time.Duration `json:"dial_timeout"`
	KeepAlive      time.Duration `json:"keep_alive"`
	MaxRetries     int           `json:"max_retries"`
}

// Tunnel manages an SSH tunnel connection.
type Tunnel struct {
	config     Config
	client     *ssh.Client
	listener   net.Listener
	localAddr  string
	mu         sync.Mutex
	closed     bool
	activeConns sync.WaitGroup
}

// New creates a new SSH tunnel with the given configuration.
func New(config Config) *Tunnel {
	if config.Port <= 0 {
		config.Port = 22
	}
	if config.DialTimeout <= 0 {
		config.DialTimeout = 30 * time.Second
	}
	if config.KeepAlive <= 0 {
		config.KeepAlive = 30 * time.Second
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}

	return &Tunnel{
		config: config,
	}
}

// Start establishes the SSH tunnel and returns the local address to connect to.
func (t *Tunnel) Start(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.client != nil {
		return t.localAddr, nil
	}

	// Build SSH client config
	sshConfig, err := t.buildSSHConfig()
	if err != nil {
		return "", fmt.Errorf("build ssh config: %w", err)
	}

	// Connect to SSH server
	sshAddr := fmt.Sprintf("%s:%d", t.config.Host, t.config.Port)
	client, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		return "", fmt.Errorf("ssh dial %s: %w", sshAddr, err)
	}
	t.client = client

	// Start local listener on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		client.Close()
		t.client = nil
		return "", fmt.Errorf("start local listener: %w", err)
	}
	t.listener = listener
	t.localAddr = listener.Addr().String()

	// Start forwarding goroutine
	go t.forward()

	// Start keepalive goroutine
	go t.keepAlive(ctx)

	return t.localAddr, nil
}

// buildSSHConfig creates an SSH client configuration.
func (t *Tunnel) buildSSHConfig() (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod

	switch t.config.AuthMethod {
	case "password":
		if t.config.Password == "" {
			return nil, fmt.Errorf("password required for password auth")
		}
		authMethods = append(authMethods, ssh.Password(t.config.Password))

	case "ssh-key", "key":
		if t.config.PrivateKey == "" {
			return nil, fmt.Errorf("private key required for key auth")
		}

		var signer ssh.Signer
		var err error

		if t.config.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(t.config.PrivateKey), []byte(t.config.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(t.config.PrivateKey))
		}

		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))

	default:
		// Try password first, then key
		if t.config.Password != "" {
			authMethods = append(authMethods, ssh.Password(t.config.Password))
		}
		if t.config.PrivateKey != "" {
			var signer ssh.Signer
			var err error
			if t.config.Passphrase != "" {
				signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(t.config.PrivateKey), []byte(t.config.Passphrase))
			} else {
				signer, err = ssh.ParsePrivateKey([]byte(t.config.PrivateKey))
			}
			if err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method configured")
	}

	return &ssh.ClientConfig{
		User:            t.config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Add proper host key verification
		Timeout:         t.config.DialTimeout,
	}, nil
}

// forward accepts connections and forwards them through the SSH tunnel.
func (t *Tunnel) forward() {
	remoteAddr := fmt.Sprintf("%s:%d", t.config.RemoteHost, t.config.RemotePort)

	for {
		localConn, err := t.listener.Accept()
		if err != nil {
			t.mu.Lock()
			closed := t.closed
			t.mu.Unlock()
			if closed {
				return
			}
			continue
		}

		t.activeConns.Add(1)
		go func() {
			defer t.activeConns.Done()
			defer localConn.Close()

			// Connect to remote through SSH
			remoteConn, err := t.client.Dial("tcp", remoteAddr)
			if err != nil {
				return
			}
			defer remoteConn.Close()

			// Bidirectional copy
			done := make(chan struct{}, 2)

			go func() {
				io.Copy(remoteConn, localConn)
				done <- struct{}{}
			}()

			go func() {
				io.Copy(localConn, remoteConn)
				done <- struct{}{}
			}()

			// Wait for either direction to finish
			<-done
		}()
	}
}

// keepAlive sends periodic keep-alive messages.
func (t *Tunnel) keepAlive(ctx context.Context) {
	ticker := time.NewTicker(t.config.KeepAlive)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.mu.Lock()
			client := t.client
			closed := t.closed
			t.mu.Unlock()

			if closed || client == nil {
				return
			}

			// Send keep-alive request
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				// Connection might be dead, but don't close here
				// Let the forward loop handle reconnection
			}
		}
	}
}

// Close closes the SSH tunnel and all connections.
func (t *Tunnel) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	t.mu.Unlock()

	var errs []error

	if t.listener != nil {
		if err := t.listener.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// Wait for active connections to finish
	t.activeConns.Wait()

	if t.client != nil {
		if err := t.client.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close tunnel: %v", errs)
	}
	return nil
}

// LocalAddr returns the local address to connect to.
func (t *Tunnel) LocalAddr() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.localAddr
}

// IsConnected returns whether the tunnel is connected.
func (t *Tunnel) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.client != nil && !t.closed
}

// TunneledConfig wraps a database config to use an SSH tunnel.
type TunneledConfig struct {
	Tunnel     *Tunnel
	OrigHost   string
	OrigPort   int
	LocalHost  string
	LocalPort  int
}

// ParseLocalAddr parses the local address into host and port.
func ParseLocalAddr(addr string) (host string, port int, err error) {
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	var portNum int
	_, err = fmt.Sscanf(p, "%d", &portNum)
	if err != nil {
		return "", 0, err
	}
	return h, portNum, nil
}
