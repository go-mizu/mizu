package chat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"now/pkg/auth"
)

// ErrPermissionDenied is returned when an actor lacks access.
var ErrPermissionDenied = errors.New("permission denied")

// Service implements API using store interfaces.
type Service struct {
	chats    ChatStore
	members  MemberStore
	messages MessageStore
}

// NewService returns a new Service.
func NewService(chats ChatStore, members MemberStore, messages MessageStore) *Service {
	return &Service{chats: chats, members: members, messages: messages}
}

// Create creates a chat and auto-joins the creator.
func (svc *Service) Create(ctx context.Context, in CreateInput, actor auth.VerifiedActor) (Chat, error) {
	id, err := newID("c_")
	if err != nil {
		return Chat{}, err
	}

	c := Chat{
		ID:          id,
		Kind:        in.Kind,
		Title:       in.Title,
		Creator:     actor.Actor,
		Fingerprint: actor.Fingerprint,
		Time:        time.Now().UTC(),
	}

	if err := svc.chats.Create(ctx, c); err != nil {
		return Chat{}, err
	}

	// Explicitly join creator as member. Do not rely on store
	// auto-join behavior — other store backends may not do this.
	if err := svc.members.Join(ctx, id, actor.Actor); err != nil {
		return Chat{}, err
	}

	return c, nil
}

// Join adds an actor to a chat. Rooms are open. Direct chats
// are limited to 2 members.
func (svc *Service) Join(ctx context.Context, in JoinInput, actor auth.VerifiedActor) error {
	c, err := svc.chats.Get(ctx, in.Chat)
	if err != nil {
		return err
	}

	if c.Kind == KindDirect {
		members, err := svc.members.List(ctx, in.Chat, 0)
		if err != nil {
			return err
		}
		if len(members) >= 2 {
			return ErrPermissionDenied
		}
	}

	return svc.members.Join(ctx, in.Chat, actor.Actor)
}

// Get returns a chat. Members only.
func (svc *Service) Get(ctx context.Context, in GetInput, actor auth.VerifiedActor) (Chat, error) {
	ok, err := svc.members.Has(ctx, in.ID, actor.Actor)
	if err != nil {
		return Chat{}, err
	}
	if !ok {
		return Chat{}, ErrPermissionDenied
	}

	return svc.chats.Get(ctx, in.ID)
}

// List returns chats the actor is a member of.
// Limit is omitted from the store call because membership filtering
// happens after retrieval — applying the store limit pre-filter would
// miss chats the actor is a member of.
func (svc *Service) List(ctx context.Context, in ListInput, actor auth.VerifiedActor) (Chats, error) {
	all, err := svc.chats.List(ctx, ListInput{Kind: in.Kind})
	if err != nil {
		return Chats{}, err
	}

	var filtered []Chat
	for _, c := range all.Items {
		ok, err := svc.members.Has(ctx, c.ID, actor.Actor)
		if err != nil {
			return Chats{}, err
		}
		if ok {
			filtered = append(filtered, c)
			if in.Limit > 0 && len(filtered) >= in.Limit {
				break
			}
		}
	}

	return Chats{Items: filtered}, nil
}

// Send creates a message in a chat. Members only.
func (svc *Service) Send(ctx context.Context, in SendInput, actor auth.VerifiedActor) (Message, error) {
	ok, err := svc.members.Has(ctx, in.Chat, actor.Actor)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return Message{}, ErrPermissionDenied
	}

	id, err := newID("m_")
	if err != nil {
		return Message{}, err
	}

	m := Message{
		ID:          id,
		Chat:        in.Chat,
		Actor:       actor.Actor,
		Fingerprint: actor.Fingerprint,
		Text:        in.Text,
		Signature:   in.Signature,
		Time:        time.Now().UTC(),
	}

	if err := svc.messages.Create(ctx, m); err != nil {
		return Message{}, err
	}

	return m, nil
}

// Messages returns messages in a chat. Members only.
func (svc *Service) Messages(ctx context.Context, in MessagesInput, actor auth.VerifiedActor) (Messages, error) {
	ok, err := svc.members.Has(ctx, in.Chat, actor.Actor)
	if err != nil {
		return Messages{}, err
	}
	if !ok {
		return Messages{}, ErrPermissionDenied
	}

	return svc.messages.List(ctx, in)
}

func newID(prefix string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b), nil
}
