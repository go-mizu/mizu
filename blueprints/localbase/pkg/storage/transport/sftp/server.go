// File: lib/storage/transport/sftp/server.go

// Package sftp provides an SFTP transport layer for storage.Storage backends.
//
// This package implements an SFTP server that exposes any storage.Storage
// implementation over the SSH File Transfer Protocol, allowing standard SFTP
// clients to interact with the storage backend.
//
// Path Mapping:
//
//	/                    → list buckets
//	/<bucket>/           → bucket root
//	/<bucket>/<key>      → object
//
// Example:
//
//	store, _ := storage.Open(ctx, "local:///data")
//	hostKey, _ := ssh.ParsePrivateKey(keyBytes)
//
//	cfg := &sftp.Config{
//	    Addr:     ":2022",
//	    HostKeys: []ssh.Signer{hostKey},
//	    Auth: sftp.AuthConfig{
//	        PublicKeyCallback: validateKey,
//	    },
//	}
//
//	server := sftp.New(store, cfg)
//	server.ListenAndServe()
package sftp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Config configures the SFTP server.
type Config struct {
	// Addr is the address to listen on (default ":2022").
	Addr string

	// HostKeys are the server's SSH host keys. At least one is required.
	HostKeys []ssh.Signer

	// Auth configures authentication.
	Auth AuthConfig

	// MaxConnections limits concurrent SSH connections. 0 means unlimited.
	MaxConnections int

	// IdleTimeout closes inactive sessions after this duration.
	// Default is 10 minutes.
	IdleTimeout time.Duration

	// ReadOnly disables all write operations when true.
	ReadOnly bool

	// Logger for server events. If nil, a default logger is used.
	Logger *slog.Logger

	// Banner is an optional message displayed before authentication.
	Banner string

	// WriteBufferSize is the max buffer size for uploads before spilling to disk.
	// Default is 32MB.
	WriteBufferSize int64

	// TempDir for buffering large uploads. Default is os.TempDir().
	TempDir string

	// HomeBucket returns the bucket a user is restricted to.
	// If nil or returns empty string, user can access all buckets.
	HomeBucket func(username string) string

	// AllowedCiphers restricts SSH ciphers. Nil uses defaults.
	AllowedCiphers []string

	// AllowedMACs restricts MAC algorithms. Nil uses defaults.
	AllowedMACs []string

	// AllowedKeyExchanges restricts key exchange algorithms. Nil uses defaults.
	AllowedKeyExchanges []string

	// DefaultUID/GID for file attributes. Defaults to 1000.
	DefaultUID uint32
	DefaultGID uint32
}

// AuthConfig configures SSH authentication.
type AuthConfig struct {
	// PublicKeyCallback validates public key authentication.
	PublicKeyCallback func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error)

	// PasswordCallback validates password authentication. Optional.
	PasswordCallback func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error)

	// NoClientAuth allows anonymous access. Use with caution.
	NoClientAuth bool

	// MaxAuthTries limits authentication attempts. Default is 6.
	MaxAuthTries int

	// KeyboardInteractiveCallback for keyboard-interactive auth. Optional.
	KeyboardInteractiveCallback func(conn ssh.ConnMetadata, client ssh.KeyboardInteractiveChallenge) (*ssh.Permissions, error)
}

func (c *Config) clone() *Config {
	if c == nil {
		return &Config{
			Addr:            ":2022",
			IdleTimeout:     10 * time.Minute,
			WriteBufferSize: 32 << 20, // 32MB
			DefaultUID:      1000,
			DefaultGID:      1000,
		}
	}

	cp := *c
	if cp.Addr == "" {
		cp.Addr = ":2022"
	}
	if cp.IdleTimeout == 0 {
		cp.IdleTimeout = 10 * time.Minute
	}
	if cp.WriteBufferSize == 0 {
		cp.WriteBufferSize = 32 << 20
	}
	if cp.TempDir == "" {
		cp.TempDir = os.TempDir()
	}
	if cp.DefaultUID == 0 {
		cp.DefaultUID = 1000
	}
	if cp.DefaultGID == 0 {
		cp.DefaultGID = 1000
	}
	if cp.Logger == nil {
		cp.Logger = slog.Default()
	}
	return &cp
}

// Server is an SFTP server backed by storage.Storage.
type Server struct {
	store    storage.Storage
	cfg      *Config
	sshCfg   *ssh.ServerConfig
	listener net.Listener

	mu           sync.Mutex
	sessions     map[*session]struct{}
	connCount    int32
	shuttingDown atomic.Bool

	done chan struct{}
}

// New creates a new SFTP server.
func New(store storage.Storage, cfg *Config) *Server {
	if store == nil {
		panic("sftp: storage is nil")
	}

	cfg = cfg.clone()

	sshCfg := &ssh.ServerConfig{
		NoClientAuth: cfg.Auth.NoClientAuth,
		MaxAuthTries: cfg.Auth.MaxAuthTries,
		BannerCallback: func(conn ssh.ConnMetadata) string {
			return cfg.Banner
		},
	}

	if sshCfg.MaxAuthTries == 0 {
		sshCfg.MaxAuthTries = 6
	}

	if cfg.Auth.PublicKeyCallback != nil {
		sshCfg.PublicKeyCallback = cfg.Auth.PublicKeyCallback
	}
	if cfg.Auth.PasswordCallback != nil {
		sshCfg.PasswordCallback = cfg.Auth.PasswordCallback
	}
	if cfg.Auth.KeyboardInteractiveCallback != nil {
		sshCfg.KeyboardInteractiveCallback = cfg.Auth.KeyboardInteractiveCallback
	}

	for _, key := range cfg.HostKeys {
		sshCfg.AddHostKey(key)
	}

	if len(cfg.AllowedCiphers) > 0 {
		sshCfg.Ciphers = cfg.AllowedCiphers
	}
	if len(cfg.AllowedMACs) > 0 {
		sshCfg.MACs = cfg.AllowedMACs
	}
	if len(cfg.AllowedKeyExchanges) > 0 {
		sshCfg.KeyExchanges = cfg.AllowedKeyExchanges
	}

	return &Server{
		store:    store,
		cfg:      cfg,
		sshCfg:   sshCfg,
		sessions: make(map[*session]struct{}),
		done:     make(chan struct{}),
	}
}

// ListenAndServe starts the SFTP server on the configured address.
func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return fmt.Errorf("sftp: listen %s: %w", s.cfg.Addr, err)
	}
	return s.Serve(ln)
}

// Serve accepts connections on the provided listener.
func (s *Server) Serve(ln net.Listener) error {
	s.listener = ln
	s.cfg.Logger.Info("sftp server started", "addr", ln.Addr().String())

	for {
		conn, err := ln.Accept()
		if err != nil {
			if s.shuttingDown.Load() {
				return nil
			}
			s.cfg.Logger.Error("accept error", "error", err)
			continue
		}

		if s.cfg.MaxConnections > 0 && atomic.LoadInt32(&s.connCount) >= int32(s.cfg.MaxConnections) {
			s.cfg.Logger.Warn("max connections reached, rejecting", "remote", conn.RemoteAddr())
			conn.Close()
			continue
		}

		atomic.AddInt32(&s.connCount, 1)
		go s.handleConnection(conn)
	}
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.shuttingDown.Store(true)

	if s.listener != nil {
		s.listener.Close()
	}

	s.mu.Lock()
	for sess := range s.sessions {
		sess.Close()
	}
	s.mu.Unlock()

	// Wait for sessions to finish
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			s.mu.Lock()
			count := len(s.sessions)
			s.mu.Unlock()
			if count == 0 {
				return nil
			}
		}
	}
}

// Close immediately closes the server.
func (s *Server) Close() error {
	s.shuttingDown.Store(true)

	var err error
	if s.listener != nil {
		err = s.listener.Close()
	}

	s.mu.Lock()
	for sess := range s.sessions {
		sess.Close()
	}
	s.mu.Unlock()

	return err
}

func (s *Server) handleConnection(netConn net.Conn) {
	defer func() {
		atomic.AddInt32(&s.connCount, -1)
	}()

	sshConn, chans, reqs, err := ssh.NewServerConn(netConn, s.sshCfg)
	if err != nil {
		s.cfg.Logger.Debug("ssh handshake failed", "remote", netConn.RemoteAddr(), "error", err)
		netConn.Close()
		return
	}

	s.cfg.Logger.Info("ssh connection established",
		"remote", sshConn.RemoteAddr(),
		"user", sshConn.User(),
	)

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			s.cfg.Logger.Debug("rejecting channel", "type", newChannel.ChannelType())
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			s.cfg.Logger.Error("accept channel failed", "error", err)
			continue
		}

		go s.handleSession(channel, requests, sshConn.User())
	}
}

func (s *Server) handleSession(channel ssh.Channel, requests <-chan *ssh.Request, username string) {
	sess := &session{
		server:   s,
		channel:  channel,
		username: username,
		handles:  make(map[string]handle),
		nextID:   1,
	}

	s.mu.Lock()
	s.sessions[sess] = struct{}{}
	s.mu.Unlock()

	defer func() {
		sess.Close()
		s.mu.Lock()
		delete(s.sessions, sess)
		s.mu.Unlock()
	}()

	var sftpStarted bool

	for req := range requests {
		switch req.Type {
		case "subsystem":
			if len(req.Payload) >= 4 {
				subsystem := string(req.Payload[4:])
				if subsystem == "sftp" {
					if req.WantReply {
						req.Reply(true, nil)
					}
					sftpStarted = true
					sess.serveSFTP()
					return
				}
			}
			if req.WantReply {
				req.Reply(false, nil)
			}
		default:
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}

	if !sftpStarted {
		s.cfg.Logger.Debug("session closed without sftp subsystem", "user", username)
	}
}

// session represents a single SFTP session.
type session struct {
	server   *Server
	channel  ssh.Channel
	username string

	mu      sync.Mutex
	handles map[string]handle
	nextID  uint64
	closed  bool
}

// handle is an open file or directory handle.
type handle interface {
	io.Closer
}

// fileReadHandle is a handle for reading files.
type fileReadHandle struct {
	bucket string
	key    string
	reader io.ReadCloser
	obj    *storage.Object
	offset int64
}

func (h *fileReadHandle) Close() error {
	if h.reader != nil {
		return h.reader.Close()
	}
	return nil
}

// fileWriteHandle is a handle for writing files.
type fileWriteHandle struct {
	bucket      string
	key         string
	contentType string
	buffer      *writeBuffer
	sess        *session
}

func (h *fileWriteHandle) Close() error {
	if h.buffer == nil {
		return nil
	}

	defer h.buffer.Close()

	ctx := context.Background()
	bkt := h.sess.server.store.Bucket(h.bucket)

	size, reader, err := h.buffer.Reader()
	if err != nil {
		return err
	}

	_, err = bkt.Write(ctx, h.key, reader, size, h.contentType, nil)
	return err
}

// dirHandle is a handle for reading directories.
type dirHandle struct {
	path    string
	entries []os.FileInfo
	offset  int
}

func (h *dirHandle) Close() error {
	return nil
}

func (s *session) allocHandle(h handle) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("%d", s.nextID)
	s.nextID++
	s.handles[id] = h
	return id
}

func (s *session) getHandle(id string) (handle, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.handles[id]
	return h, ok
}

func (s *session) closeHandle(id string) error {
	s.mu.Lock()
	h, ok := s.handles[id]
	if ok {
		delete(s.handles, id)
	}
	s.mu.Unlock()

	if h != nil {
		return h.Close()
	}
	return nil
}

func (s *session) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true

	for id, h := range s.handles {
		delete(s.handles, id)
		if h != nil {
			h.Close()
		}
	}
	s.mu.Unlock()

	return s.channel.Close()
}

func (s *session) serveSFTP() {
	handler := &sftpHandler{sess: s}

	server := sftp.NewRequestServer(s.channel, sftp.Handlers{
		FileGet:  handler,
		FilePut:  handler,
		FileCmd:  handler,
		FileList: handler,
	})

	if err := server.Serve(); err != nil {
		if err != io.EOF {
			s.server.cfg.Logger.Error("sftp serve error", "user", s.username, "error", err)
		}
	}
	server.Close()
}

// sftpHandler implements sftp.Handlers.
type sftpHandler struct {
	sess *session
}

// Fileread implements sftp.FileReader.
func (h *sftpHandler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	return h.openReader(r)
}

// Filewrite implements sftp.FileWriter.
func (h *sftpHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	if h.sess.server.cfg.ReadOnly {
		return nil, sftp.ErrSSHFxPermissionDenied
	}
	return h.openWriter(r)
}

// Filecmd implements sftp.FileCmder.
func (h *sftpHandler) Filecmd(r *sftp.Request) error {
	ctx := context.Background()
	p := h.resolvePath(r.Filepath)

	switch r.Method {
	case "Remove":
		return h.remove(ctx, p)
	case "Rename":
		target := h.resolvePath(r.Target)
		return h.rename(ctx, p, target)
	case "Mkdir":
		return h.mkdir(ctx, p)
	case "Rmdir":
		return h.rmdir(ctx, p)
	case "Setstat":
		// Limited support - ignore most setstat operations
		return nil
	case "Symlink":
		return sftp.ErrSSHFxOpUnsupported
	default:
		return sftp.ErrSSHFxOpUnsupported
	}
}

// Filelist implements sftp.FileLister.
func (h *sftpHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	ctx := context.Background()
	p := h.resolvePath(r.Filepath)

	switch r.Method {
	case "List":
		return h.listDir(ctx, p)
	case "Stat":
		return h.stat(ctx, p)
	case "Readlink":
		return nil, sftp.ErrSSHFxOpUnsupported
	default:
		return nil, sftp.ErrSSHFxOpUnsupported
	}
}

// resolvePath resolves the SFTP path considering user restrictions.
func (h *sftpHandler) resolvePath(p string) string {
	p = path.Clean("/" + p)

	if h.sess.server.cfg.HomeBucket != nil {
		home := h.sess.server.cfg.HomeBucket(h.sess.username)
		if home != "" {
			// User is chrooted to a specific bucket
			if p == "/" {
				return "/" + home
			}
			return "/" + home + p
		}
	}

	return p
}

// parsePath splits a path into bucket and key.
func parsePath(p string) (bucket, key string) {
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return "", ""
	}

	parts := strings.SplitN(p, "/", 2)
	bucket = parts[0]
	if len(parts) > 1 {
		key = parts[1]
	}
	return bucket, key
}

func (h *sftpHandler) openReader(r *sftp.Request) (io.ReaderAt, error) {
	ctx := context.Background()
	p := h.resolvePath(r.Filepath)
	bucket, key := parsePath(p)

	if bucket == "" {
		return nil, sftp.ErrSSHFxNoSuchFile
	}
	if key == "" {
		return nil, sftp.ErrSSHFxNoSuchFile // Can't read a bucket as a file
	}

	bkt := h.sess.server.store.Bucket(bucket)
	reader, obj, err := bkt.Open(ctx, key, 0, -1, nil)
	if err != nil {
		return nil, mapError(err)
	}

	return &readerAtWrapper{
		reader: reader,
		obj:    obj,
		ctx:    ctx,
		bkt:    bkt,
		key:    key,
	}, nil
}

func (h *sftpHandler) openWriter(r *sftp.Request) (io.WriterAt, error) {
	p := h.resolvePath(r.Filepath)
	bucket, key := parsePath(p)

	if bucket == "" {
		return nil, sftp.ErrSSHFxPermissionDenied
	}
	if key == "" {
		return nil, sftp.ErrSSHFxPermissionDenied // Can't write to a bucket directly
	}

	buf := newWriteBuffer(h.sess.server.cfg.WriteBufferSize, h.sess.server.cfg.TempDir)

	return &writerAtWrapper{
		buffer:      buf,
		bucket:      bucket,
		key:         key,
		contentType: detectContentType(key),
		sess:        h.sess,
	}, nil
}

func (h *sftpHandler) remove(ctx context.Context, p string) error {
	if h.sess.server.cfg.ReadOnly {
		return sftp.ErrSSHFxPermissionDenied
	}

	bucket, key := parsePath(p)
	if bucket == "" {
		return sftp.ErrSSHFxNoSuchFile
	}
	if key == "" {
		// Removing bucket - use DeleteBucket
		return sftp.ErrSSHFxPermissionDenied // Use rmdir for buckets
	}

	bkt := h.sess.server.store.Bucket(bucket)
	err := bkt.Delete(ctx, key, nil)
	return mapError(err)
}

func (h *sftpHandler) rename(ctx context.Context, src, dst string) error {
	if h.sess.server.cfg.ReadOnly {
		return sftp.ErrSSHFxPermissionDenied
	}

	srcBucket, srcKey := parsePath(src)
	dstBucket, dstKey := parsePath(dst)

	if srcBucket == "" || dstBucket == "" {
		return sftp.ErrSSHFxPermissionDenied
	}
	if srcKey == "" || dstKey == "" {
		return sftp.ErrSSHFxPermissionDenied // Can't rename buckets
	}

	if srcBucket != dstBucket {
		// Cross-bucket move not supported by most backends
		return sftp.ErrSSHFxOpUnsupported
	}

	bkt := h.sess.server.store.Bucket(srcBucket)
	_, err := bkt.Move(ctx, dstKey, srcBucket, srcKey, nil)
	return mapError(err)
}

func (h *sftpHandler) mkdir(ctx context.Context, p string) error {
	if h.sess.server.cfg.ReadOnly {
		return sftp.ErrSSHFxPermissionDenied
	}

	bucket, key := parsePath(p)
	if bucket == "" {
		return sftp.ErrSSHFxPermissionDenied
	}

	if key == "" {
		// Creating a bucket
		_, err := h.sess.server.store.CreateBucket(ctx, bucket, nil)
		if err != nil {
			if errors.Is(err, storage.ErrExist) {
				return nil // Bucket already exists, that's fine
			}
			return mapError(err)
		}
		return nil
	}

	// Creating a directory within a bucket
	// In object storage, directories are virtual. We can either:
	// 1. Do nothing (directories are implicit)
	// 2. Create a marker object (e.g., key + "/.keep")
	// We'll do nothing for now since most clients don't need explicit directories.
	return nil
}

func (h *sftpHandler) rmdir(ctx context.Context, p string) error {
	if h.sess.server.cfg.ReadOnly {
		return sftp.ErrSSHFxPermissionDenied
	}

	bucket, key := parsePath(p)
	if bucket == "" {
		return sftp.ErrSSHFxPermissionDenied
	}

	if key == "" {
		// Removing a bucket
		err := h.sess.server.store.DeleteBucket(ctx, bucket, nil)
		return mapError(err)
	}

	// Removing a "directory" - delete all objects with this prefix
	bkt := h.sess.server.store.Bucket(bucket)
	prefix := strings.TrimSuffix(key, "/") + "/"

	iter, err := bkt.List(ctx, prefix, 0, 0, storage.Options{"recursive": true})
	if err != nil {
		return mapError(err)
	}
	defer iter.Close()

	var count int
	for {
		obj, err := iter.Next()
		if err != nil {
			return mapError(err)
		}
		if obj == nil {
			break
		}
		count++
		if err := bkt.Delete(ctx, obj.Key, nil); err != nil {
			return mapError(err)
		}
	}

	if count == 0 {
		// No objects found - directory doesn't exist
		return sftp.ErrSSHFxNoSuchFile
	}

	return nil
}

func (h *sftpHandler) listDir(ctx context.Context, p string) (sftp.ListerAt, error) {
	bucket, key := parsePath(p)

	var entries []os.FileInfo

	if bucket == "" {
		// List buckets at root
		iter, err := h.sess.server.store.Buckets(ctx, 0, 0, nil)
		if err != nil {
			return nil, mapError(err)
		}
		defer iter.Close()

		for {
			info, err := iter.Next()
			if err != nil {
				return nil, mapError(err)
			}
			if info == nil {
				break
			}
			entries = append(entries, &fileInfo{
				name:    info.Name,
				size:    0,
				mode:    os.ModeDir | 0755,
				modTime: info.CreatedAt,
				isDir:   true,
				uid:     h.sess.server.cfg.DefaultUID,
				gid:     h.sess.server.cfg.DefaultGID,
			})
		}
	} else {
		// List objects in bucket
		bkt := h.sess.server.store.Bucket(bucket)
		prefix := key
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}

		iter, err := bkt.List(ctx, prefix, 0, 0, nil)
		if err != nil {
			return nil, mapError(err)
		}
		defer iter.Close()

		seen := make(map[string]bool)

		for {
			obj, err := iter.Next()
			if err != nil {
				return nil, mapError(err)
			}
			if obj == nil {
				break
			}

			// Extract the immediate child name
			remaining := strings.TrimPrefix(obj.Key, prefix)
			if remaining == "" {
				continue
			}

			parts := strings.SplitN(remaining, "/", 2)
			name := parts[0]

			if seen[name] {
				continue
			}
			seen[name] = true

			isDir := len(parts) > 1 || obj.IsDir
			var mode os.FileMode
			if isDir {
				mode = os.ModeDir | 0755
			} else {
				mode = 0644
			}

			entries = append(entries, &fileInfo{
				name:    name,
				size:    obj.Size,
				mode:    mode,
				modTime: obj.Updated,
				isDir:   isDir,
				uid:     h.sess.server.cfg.DefaultUID,
				gid:     h.sess.server.cfg.DefaultGID,
			})
		}
	}

	return listerat(entries), nil
}

func (h *sftpHandler) stat(ctx context.Context, p string) (sftp.ListerAt, error) {
	bucket, key := parsePath(p)

	if bucket == "" {
		// Root directory
		return listerat([]os.FileInfo{&fileInfo{
			name:    "/",
			size:    0,
			mode:    os.ModeDir | 0755,
			modTime: time.Now(),
			isDir:   true,
			uid:     h.sess.server.cfg.DefaultUID,
			gid:     h.sess.server.cfg.DefaultGID,
		}}), nil
	}

	if key == "" {
		// Stat a bucket
		bkt := h.sess.server.store.Bucket(bucket)
		info, err := bkt.Info(ctx)
		if err != nil {
			return nil, mapError(err)
		}
		return listerat([]os.FileInfo{&fileInfo{
			name:    bucket,
			size:    0,
			mode:    os.ModeDir | 0755,
			modTime: info.CreatedAt,
			isDir:   true,
			uid:     h.sess.server.cfg.DefaultUID,
			gid:     h.sess.server.cfg.DefaultGID,
		}}), nil
	}

	// Stat an object
	bkt := h.sess.server.store.Bucket(bucket)
	obj, err := bkt.Stat(ctx, key, nil)
	if err != nil {
		// Check if it's a directory prefix
		iter, listErr := bkt.List(ctx, key+"/", 1, 0, nil)
		if listErr == nil {
			defer iter.Close()
			if obj, _ := iter.Next(); obj != nil {
				// It's a directory
				return listerat([]os.FileInfo{&fileInfo{
					name:    path.Base(key),
					size:    0,
					mode:    os.ModeDir | 0755,
					modTime: time.Now(),
					isDir:   true,
					uid:     h.sess.server.cfg.DefaultUID,
					gid:     h.sess.server.cfg.DefaultGID,
				}}), nil
			}
		}
		return nil, mapError(err)
	}

	var mode os.FileMode
	if obj.IsDir {
		mode = os.ModeDir | 0755
	} else {
		mode = 0644
	}

	return listerat([]os.FileInfo{&fileInfo{
		name:    path.Base(key),
		size:    obj.Size,
		mode:    mode,
		modTime: obj.Updated,
		isDir:   obj.IsDir,
		uid:     h.sess.server.cfg.DefaultUID,
		gid:     h.sess.server.cfg.DefaultGID,
	}}), nil
}

// mapError maps storage errors to SFTP errors.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, storage.ErrNotExist):
		return sftp.ErrSSHFxNoSuchFile
	case errors.Is(err, storage.ErrExist):
		return sftp.ErrSSHFxFailure
	case errors.Is(err, storage.ErrPermission):
		return sftp.ErrSSHFxPermissionDenied
	case errors.Is(err, storage.ErrUnsupported):
		return sftp.ErrSSHFxOpUnsupported
	default:
		return sftp.ErrSSHFxFailure
	}
}

// fileInfo implements os.FileInfo for SFTP responses.
type fileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
	uid     uint32
	gid     uint32
}

func (f *fileInfo) Name() string       { return f.name }
func (f *fileInfo) Size() int64        { return f.size }
func (f *fileInfo) Mode() os.FileMode  { return f.mode }
func (f *fileInfo) ModTime() time.Time { return f.modTime }
func (f *fileInfo) IsDir() bool        { return f.isDir }
func (f *fileInfo) Sys() interface{} {
	return &sftp.FileStat{
		Size:  uint64(f.size),
		Mode:  uint32(f.mode),
		Mtime: uint32(f.modTime.Unix()),
		Atime: uint32(f.modTime.Unix()),
		UID:   f.uid,
		GID:   f.gid,
	}
}

// listerat implements sftp.ListerAt.
type listerat []os.FileInfo

func (l listerat) ListAt(entries []os.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}

	n := copy(entries, l[offset:])
	if offset+int64(n) >= int64(len(l)) {
		return n, io.EOF
	}
	return n, nil
}

// readerAtWrapper wraps io.Reader to provide io.ReaderAt.
type readerAtWrapper struct {
	reader io.ReadCloser
	obj    *storage.Object
	ctx    context.Context
	bkt    storage.Bucket
	key    string
	mu     sync.Mutex
	offset int64
}

func (r *readerAtWrapper) ReadAt(p []byte, off int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If we need to seek, reopen with offset
	if off != r.offset {
		if r.reader != nil {
			r.reader.Close()
			r.reader = nil
		}

		// Open with -1 length to read all remaining bytes from offset
		reader, _, err := r.bkt.Open(r.ctx, r.key, off, -1, nil)
		if err != nil {
			return 0, err
		}
		r.reader = reader
		r.offset = off
	}

	// Read exactly len(p) bytes or until EOF
	n, err := io.ReadFull(r.reader, p)
	r.offset += int64(n)

	// Convert io.ErrUnexpectedEOF to io.EOF for partial reads at end
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	return n, err
}

func (r *readerAtWrapper) Close() error {
	if r.reader != nil {
		return r.reader.Close()
	}
	return nil
}

// writerAtWrapper wraps writeBuffer to provide io.WriterAt.
type writerAtWrapper struct {
	buffer      *writeBuffer
	bucket      string
	key         string
	contentType string
	sess        *session
	mu          sync.Mutex
	closed      bool
}

func (w *writerAtWrapper) WriteAt(p []byte, off int64) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, errors.New("writer closed")
	}

	return w.buffer.WriteAt(p, off)
}

func (w *writerAtWrapper) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.mu.Unlock()

	ctx := context.Background()
	bkt := w.sess.server.store.Bucket(w.bucket)

	size, reader, err := w.buffer.Reader()
	if err != nil {
		w.buffer.Close()
		return err
	}

	_, err = bkt.Write(ctx, w.key, reader, size, w.contentType, nil)
	w.buffer.Close()
	return err
}

// detectContentType guesses content type from file extension.
func detectContentType(filename string) string {
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".txt":
		return "text/plain"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz":
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}
