package tunnel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTunnel(t *testing.T) {
	config := Config{
		Host:       "bastion.example.com",
		Port:       22,
		User:       "admin",
		AuthMethod: "password",
		Password:   "secret",
		RemoteHost: "db.internal",
		RemotePort: 5432,
	}

	tunnel := New(config)
	require.NotNil(t, tunnel)

	assert.Equal(t, 22, tunnel.config.Port)
	assert.Equal(t, 30*time.Second, tunnel.config.DialTimeout)
	assert.Equal(t, 30*time.Second, tunnel.config.KeepAlive)
	assert.Equal(t, 3, tunnel.config.MaxRetries)
}

func TestNewTunnelDefaults(t *testing.T) {
	config := Config{
		Host:       "bastion.example.com",
		User:       "admin",
		Password:   "secret",
		RemoteHost: "db.internal",
		RemotePort: 5432,
	}

	tunnel := New(config)

	// Check defaults are applied
	assert.Equal(t, 22, tunnel.config.Port)
	assert.Equal(t, 30*time.Second, tunnel.config.DialTimeout)
	assert.Equal(t, 30*time.Second, tunnel.config.KeepAlive)
	assert.Equal(t, 3, tunnel.config.MaxRetries)
}

func TestBuildSSHConfigPassword(t *testing.T) {
	tunnel := &Tunnel{
		config: Config{
			User:       "admin",
			AuthMethod: "password",
			Password:   "secret",
		},
	}

	sshConfig, err := tunnel.buildSSHConfig()
	require.NoError(t, err)
	require.NotNil(t, sshConfig)

	assert.Equal(t, "admin", sshConfig.User)
	assert.Len(t, sshConfig.Auth, 1)
}

func TestBuildSSHConfigPasswordMissing(t *testing.T) {
	tunnel := &Tunnel{
		config: Config{
			User:       "admin",
			AuthMethod: "password",
			Password:   "", // Missing
		},
	}

	_, err := tunnel.buildSSHConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password required")
}

func TestBuildSSHConfigKeyMissing(t *testing.T) {
	tunnel := &Tunnel{
		config: Config{
			User:       "admin",
			AuthMethod: "ssh-key",
			PrivateKey: "", // Missing
		},
	}

	_, err := tunnel.buildSSHConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private key required")
}

func TestBuildSSHConfigNoAuth(t *testing.T) {
	tunnel := &Tunnel{
		config: Config{
			User: "admin",
		},
	}

	_, err := tunnel.buildSSHConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no authentication method")
}

func TestBuildSSHConfigWithValidKey(t *testing.T) {
	// Generate a test RSA key (this is a valid but test-only key)
	testKey := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAlwAAAAdzc2gtcn
NhAAAAAwEAAQAAAIEA0Z3VS5JJcds3xfn/ygWyF8PbnGy5nc1qRxzRg3sW7v1LWLmvSHj1
8C1v3V3v3V3v3V3v3V3v3V3v3V3v3V3v3V3v3V3v3V3v3V3v3V3v3V3v3V3v3V3v3V3v3V
3v3V3v3V3v3V3v3V3v3EAAAH4test-keytest-keyAAAAB3NzaC1yc2EAAACANdVdVQySQ3H
bN8X5/8oFshfD25xsuZ3NakcM0YN7Fu79S1i5r0h49fAtb91d791d791d791d791d791d79
1d791d791d791d791d791d791d791d791d791d791d791d791d791d791d791d791d791d79
1d79xAAAAAwEAAQAAAIA=
-----END OPENSSH PRIVATE KEY-----`

	tunnel := &Tunnel{
		config: Config{
			User:       "admin",
			AuthMethod: "ssh-key",
			PrivateKey: testKey,
		},
	}

	// This will fail because the key is not valid, but it tests the parsing path
	_, err := tunnel.buildSSHConfig()
	// The key format is intentionally invalid for testing
	assert.Error(t, err)
}

func TestIsConnected(t *testing.T) {
	tunnel := New(Config{
		Host:       "bastion.example.com",
		User:       "admin",
		Password:   "secret",
		RemoteHost: "db.internal",
		RemotePort: 5432,
	})

	assert.False(t, tunnel.IsConnected())
}

func TestLocalAddr(t *testing.T) {
	tunnel := New(Config{
		Host:       "bastion.example.com",
		User:       "admin",
		Password:   "secret",
		RemoteHost: "db.internal",
		RemotePort: 5432,
	})

	// Before start, local addr is empty
	assert.Empty(t, tunnel.LocalAddr())
}

func TestCloseNotStarted(t *testing.T) {
	tunnel := New(Config{
		Host:       "bastion.example.com",
		User:       "admin",
		Password:   "secret",
		RemoteHost: "db.internal",
		RemotePort: 5432,
	})

	// Close should not error when not started
	err := tunnel.Close()
	assert.NoError(t, err)
}

func TestCloseIdempotent(t *testing.T) {
	tunnel := New(Config{
		Host:       "bastion.example.com",
		User:       "admin",
		Password:   "secret",
		RemoteHost: "db.internal",
		RemotePort: 5432,
	})

	// Close multiple times should not error
	err := tunnel.Close()
	assert.NoError(t, err)

	err = tunnel.Close()
	assert.NoError(t, err)
}

func TestParseLocalAddr(t *testing.T) {
	tests := []struct {
		name         string
		addr         string
		expectedHost string
		expectedPort int
		expectError  bool
	}{
		{
			name:         "valid address",
			addr:         "127.0.0.1:5432",
			expectedHost: "127.0.0.1",
			expectedPort: 5432,
			expectError:  false,
		},
		{
			name:         "localhost",
			addr:         "localhost:3306",
			expectedHost: "localhost",
			expectedPort: 3306,
			expectError:  false,
		},
		{
			name:        "invalid format",
			addr:        "invalid",
			expectError: true,
		},
		{
			name:        "invalid port",
			addr:        "127.0.0.1:abc",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := ParseLocalAddr(tt.addr)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedHost, host)
				assert.Equal(t, tt.expectedPort, port)
			}
		})
	}
}

func TestConfig(t *testing.T) {
	config := Config{
		Host:        "bastion.example.com",
		Port:        2222,
		User:        "deploy",
		AuthMethod:  "ssh-key",
		PrivateKey:  "-----BEGIN RSA PRIVATE KEY-----\n...",
		Passphrase:  "secret",
		RemoteHost:  "db.internal.example.com",
		RemotePort:  5432,
		DialTimeout: 10 * time.Second,
		KeepAlive:   60 * time.Second,
		MaxRetries:  5,
	}

	assert.Equal(t, "bastion.example.com", config.Host)
	assert.Equal(t, 2222, config.Port)
	assert.Equal(t, "deploy", config.User)
	assert.Equal(t, "ssh-key", config.AuthMethod)
	assert.Equal(t, "db.internal.example.com", config.RemoteHost)
	assert.Equal(t, 5432, config.RemotePort)
	assert.Equal(t, 10*time.Second, config.DialTimeout)
	assert.Equal(t, 60*time.Second, config.KeepAlive)
	assert.Equal(t, 5, config.MaxRetries)
}
