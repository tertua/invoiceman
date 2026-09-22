package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/logger"
)

// ErrCacheMiss is returned when a key is absent or expired.
var ErrCacheMiss = errors.New("cache miss")

// AggCache stores per-user JSON aggregates (dashboard, reports) with
// identical behaviour on both backends: Redis when REDIS_HOST is set,
// process-local memory otherwise. Only replica-sharing differs, never the
// contract — resource-limited builds without Redis keep working.
type AggCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	DeletePrefix(ctx context.Context, prefix string) error
}

// AggTTL returns the aggregate time-to-live from the central config.
// Writes invalidate explicitly, so this only bounds staleness for
// multi-replica deployments without Redis.
func AggTTL() time.Duration {
	if s := configs.Get().Cache.AggTTLSeconds; s > 0 {
		return time.Duration(s) * time.Second
	}
	return 60 * time.Second
}

// AggKey builds the cache key for one aggregate slice. Currency is
// normalized so "?currency=" and "?currency=USD" never collide.
func AggKey(userID, kind, currency string) string {
	cur := strings.ToLower(strings.TrimSpace(currency))
	if cur == "" {
		cur = "-"
	}
	return "agg:v1:" + userID + ":" + kind + ":" + cur
}

// AggScope returns the key prefix covering every aggregate of a user,
// for invalidation after writes.
func AggScope(userID string) string {
	return "agg:v1:" + userID + ":"
}

// InvalidateUser drops every cached aggregate of a user. Best-effort:
// callers already committed the write, so a failure only costs TTL
// staleness and is reported, never fatal.
func InvalidateUser(ctx context.Context, userID string) error {
	ac, err := Aggregates()
	if err != nil {
		return err
	}
	return ac.DeletePrefix(ctx, AggScope(userID))
}

var (
	aggOnce   sync.Once
	sharedAgg AggCache
	aggErr    error
)

// Aggregates returns the shared aggregate cache: Redis when REDIS_HOST is
// set, in-memory otherwise (same selection rule as sessions and limiters).
func Aggregates() (AggCache, error) {
	aggOnce.Do(func() {
		if !configs.Get().Redis.Enabled() {
			sharedAgg = newMemoryAgg()
			return
		}
		client, err := RedisConnection()
		if err != nil {
			aggErr = err
			return
		}
		sharedAgg = &redisAgg{client: client}
	})
	return sharedAgg, aggErr
}

// FetchJSON returns the cached value for key, or futures it exactly once
// across concurrent callers (singleflight): the winner runs fetch, stores
// the JSON, and every waiter decodes the same bytes. Any cache failure
// fails open to a direct fetch, so the API keeps working when Redis is
// down — at DB cost, not error cost.
func FetchJSON[T any](ctx context.Context, key string, fetch func(context.Context) (T, error)) (T, error) {
	var zero T
	ac, err := Aggregates()
	if err != nil {
		logger.L().Debug("aggregate cache unavailable, fetching direct", "err", err)
		return fetch(ctx)
	}
	if raw, err := ac.Get(ctx, key); err == nil {
		if uerr := json.Unmarshal(raw, &zero); uerr == nil {
			return zero, nil
		}
	} else if !errors.Is(err, ErrCacheMiss) {
		logger.L().Debug("aggregate cache read failed, fetching direct", "key", key, "err", err)
		return fetch(ctx)
	}

	c := flightStart(key)
	if c == nil {
		// Another goroutine is fetching; wait for its bytes. When the
		// winner already finished (rare race), fall back to direct.
		c, ok := flightWait(key)
		if !ok {
			return fetch(ctx)
		}
		if c.err != nil {
			return zero, c.err
		}
		if err := json.Unmarshal(c.data, &zero); err != nil {
			return zero, err
		}
		return zero, nil
	}
	defer flightDone(key, c)
	val, ferr := fetch(ctx)
	if ferr != nil {
		c.err = ferr
		return zero, ferr
	}
	raw, merr := json.Marshal(val)
	if merr != nil {
		c.err = merr
		return zero, merr
	}
	if serr := ac.Set(ctx, key, raw, AggTTL()); serr != nil {
		logger.L().Debug("aggregate cache write failed", "key", key, "err", serr)
	}
	c.data = raw
	if err := json.Unmarshal(raw, &zero); err != nil {
		return zero, err
	}
	return zero, nil
}

// --- singleflight (stdlib only) ---

type flightCall struct {
	wg   sync.WaitGroup
	data []byte
	err  error
}

var flight = struct {
	sync.Mutex
	calls map[string]*flightCall
}{calls: map[string]*flightCall{}}

// flightStart registers the caller as the winner for key, or returns nil
// when another goroutine already owns the fetch.
func flightStart(key string) *flightCall {
	flight.Lock()
	defer flight.Unlock()
	if _, ok := flight.calls[key]; ok {
		return nil
	}
	c := &flightCall{}
	c.wg.Add(1)
	flight.calls[key] = c
	return c
}

// flightWait blocks until the winner finishes and returns its call.
// ok is false when the winner already finished (caller falls back).
func flightWait(key string) (c *flightCall, ok bool) {
	flight.Lock()
	c, ok = flight.calls[key]
	flight.Unlock()
	if !ok {
		return nil, false
	}
	c.wg.Wait()
	return c, true
}

func flightDone(key string, c *flightCall) {
	c.wg.Done()
	flight.Lock()
	delete(flight.calls, key)
	flight.Unlock()
}

// --- Redis backend ---

type redisAgg struct {
	client *redis.Client
}

func (a *redisAgg) Get(ctx context.Context, key string) ([]byte, error) {
	raw, err := a.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrCacheMiss
	}
	return raw, err
}

func (a *redisAgg) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return a.client.Set(ctx, key, value, ttl).Err()
}

func (a *redisAgg) DeletePrefix(ctx context.Context, prefix string) error {
	iter := a.client.Scan(ctx, 0, prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		if err := a.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// --- memory backend ---

type memAggEntry struct {
	data    []byte
	expires time.Time
}

type memoryAgg struct {
	mu    sync.Mutex
	items map[string]memAggEntry
}

func newMemoryAgg() *memoryAgg {
	a := &memoryAgg{items: map[string]memAggEntry{}}
	go a.janitor()
	return a
}

func (a *memoryAgg) Get(_ context.Context, key string) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	e, ok := a.items[key]
	if !ok || time.Now().After(e.expires) {
		delete(a.items, key)
		return nil, ErrCacheMiss
	}
	return e.data, nil
}

func (a *memoryAgg) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.items[key] = memAggEntry{data: value, expires: time.Now().Add(ttl)}
	return nil
}

func (a *memoryAgg) DeletePrefix(_ context.Context, prefix string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for k := range a.items {
		if strings.HasPrefix(k, prefix) {
			delete(a.items, k)
		}
	}
	return nil
}

func (a *memoryAgg) janitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		a.mu.Lock()
		for k, e := range a.items {
			if now.After(e.expires) {
				delete(a.items, k)
			}
		}
		a.mu.Unlock()
	}
}
