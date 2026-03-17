package chat

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"now/pkg/auth"
	pkgchat "now/pkg/chat"
	"now/store/duckdb"

	"github.com/spf13/cobra"
)

type chatConfig struct {
	Actor       string `json:"actor"`
	PublicKey   string `json:"public_key"`
	PrivateKey  string `json:"private_key"`
	Fingerprint string `json:"fingerprint"`
}

// authResult holds the output of signAndVerify.
type authResult struct {
	Actor     auth.VerifiedActor
	Signature []byte
}

type deps struct {
	identity *auth.Identity // nil if no config exists (ok for init)
	svc      *pkgchat.Service
	keys     auth.KeyStore
	nonces   auth.NonceStore
	db       *duckdb.DB
}

// New returns the chat command.
func New() *cobra.Command {
	d := &deps{}

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Chat commands",
		Long: `Chat commands for direct and room conversations.

Use chat to create chats, join chats, send messages,
and read message history.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Load identity (lenient — nil for init).
			actorFlag, _ := cmd.Flags().GetString("actor")
			d.identity = loadIdentity(actorFlag)

			// Open DuckDB.
			dbPath, _ := cmd.Flags().GetString("db")
			if dbPath == "" {
				dbPath = os.Getenv("NOW_DB")
			}
			if dbPath == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				dbPath = filepath.Join(home, "data", "now", "chat.duckdb")
			}

			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return err
			}

			db, err := duckdb.Open(dbPath)
			if err != nil {
				return err
			}
			d.db = db
			d.svc = pkgchat.NewService(db.Chats(), db.Members(), db.Messages())
			d.keys = db.Keys()
			d.nonces = auth.NewMemNonceStore(auth.DefaultTimestampWindow)

			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if d.db != nil {
				return d.db.Close()
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().String("actor", "", "actor identity (e.g. u/alice, a/bot1)")
	cmd.PersistentFlags().String("db", "", "database path (default: $HOME/data/now/chat.duckdb)")

	cmd.AddCommand(
		newInitCmd(d),
		newCreateCmd(d),
		newGetCmd(d),
		newListCmd(d),
		newJoinCmd(d),
		newSendCmd(d),
		newMessagesCmd(d),
	)

	return cmd
}

func loadIdentity(actorOverride string) *auth.Identity {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	path := filepath.Join(home, ".config", "now", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var cfg chatConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}

	pub, err := base64.URLEncoding.DecodeString(cfg.PublicKey)
	if err != nil {
		return nil
	}
	priv, err := base64.URLEncoding.DecodeString(cfg.PrivateKey)
	if err != nil {
		return nil
	}

	actor := cfg.Actor
	if actorOverride != "" {
		actor = actorOverride
	}

	return &auth.Identity{
		Actor:       actor,
		Fingerprint: cfg.Fingerprint,
		PublicKey:   ed25519.PublicKey(pub),
		PrivateKey:  ed25519.PrivateKey(priv),
	}
}

func requireIdentity(d *deps) error {
	if d.identity == nil {
		return errors.New("no actor configured, run \"now chat init\" or use --actor")
	}
	return nil
}

// signAndVerify signs a request and verifies it, returning the verified
// actor and the raw signature bytes (for non-repudiation storage).
func signAndVerify(cmd *cobra.Command, d *deps, operation string, params map[string]string) (*authResult, error) {
	if err := requireIdentity(d); err != nil {
		return nil, err
	}

	signed, err := auth.Sign(d.identity, operation, params)
	if err != nil {
		return nil, err
	}

	verified, err := auth.Verify(cmd.Context(), signed, d.keys, d.nonces)
	if err != nil {
		return nil, err
	}

	return &authResult{
		Actor:     *verified,
		Signature: signed.Signature,
	}, nil
}
