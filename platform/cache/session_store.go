package cache

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tertua/invoiceman/pkg/configs"
)

// ErrSessionNotFound is returned when a session key does not exist.
var ErrSessionNotFound = errors.New("session not found")

// SessionStore persists refresh tokens.
type SessionStore interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// RefreshTTL returns the refresh token lifetime from the central config.
func RefreshTTL() time.Duration {
	return time.Hour * time.Duration(configs.Get().JWT.RefreshHours)
}

// EncodeSessionValue packs a session id, refresh token and bound CSRF token.
// The "\n" separators are safe: refresh tokens are hex + "." + digits and
// CSRF tokens are hex, never multiline.
func EncodeSessionValue(sid, refresh, csrf string) string {
	return sid + "\n" + refresh + "\n" + csrf
}

// DecodeSessionValue splits a stored session value back into its parts.
// Legacy entries predate fields: a bare refresh string (pre-sid) yields an
// empty sid and csrf, and a two-part value yields an empty csrf.
func DecodeSessionValue(value string) (sid, refresh, csrf string, ok bool) {
	parts := strings.Split(value, "\n")
	switch len(parts) {
	case 1:
		if parts[0] == "" {
			return "", "", "", false
		}
		return "", parts[0], "", true
	case 2:
		if parts[0] == "" || parts[1] == "" {
			return "", "", "", false
		}
		return parts[0], parts[1], "", true
	default:
		if parts[0] == "" || parts[1] == "" {
			return "", "", "", false
		}
		return parts[0], parts[1], parts[2], true
	}
}

var (
	storeOnce   sync.Once
	sharedStore SessionStore
	storeErr    error
)

// Sessions returns the shared session store: Redis when REDIS_HOST is set,
// in-memory otherwise (single instance, suitable for dev without Redis).
func Sessions() (SessionStore, error) {
	storeOnce.Do(func() {
		if !configs.Get().Redis.Enabled() {
			sharedStore = newMemoryStore()
			return
		}
		client, err := RedisConnection()
		if err != nil {
			storeErr = err
			return
		}
		sharedStore = &redisStore{client: client}
	})
	return sharedStore, storeErr
}

// redisStore is a SessionStore backed by Redis.
type redisStore struct {
	client *redis.Client
}

func (s *redisStore) Get(ctx context.Context, key string) (string, error) {
	value, err := s.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrSessionNotFound
	}
	return value, err
}

func (s *redisStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.client.Set(ctx, key, value, ttl).Err()
}

func (s *redisStore) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

// memoryEntry is one in-memory session with expiry.
type memoryEntry struct {
	value   string
	expires time.Time
}

// memoryStore is a SessionStore backed by a guarded map.
type memoryStore struct {
	mu    sync.Mutex
	items map[string]memoryEntry
}

func newMemoryStore() *memoryStore {
	store := &memoryStore{items: make(map[string]memoryEntry)}
	go store.janitor()
	return store
}

func (s *memoryStore) Get(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.items[key]
	if !ok || time.Now().After(entry.expires) {
		delete(s.items, key)
		return "", ErrSessionNotFound
	}
	return entry.value, nil
}

func (s *memoryStore) Set(_ context.Context, key, value string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[key] = memoryEntry{value: value, expires: time.Now().Add(ttl)}
	return nil
}

func (s *memoryStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, key)
	return nil
}

// janitor removes expired sessions every minute.
func (s *memoryStore) janitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for key, entry := range s.items {
			if now.After(entry.expires) {
				delete(s.items, key)
			}
		}
		s.mu.Unlock()
	}
}
