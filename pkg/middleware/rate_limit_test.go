package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/pkg/configs"
)

// mapStorage is an in-process fiber.Storage shared across limiters in tests.
type mapStorage struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newMapStorage() *mapStorage { return &mapStorage{data: make(map[string][]byte)} }

func (s *mapStorage) GetWithContext(_ context.Context, key string) ([]byte, error) { return s.get(key) }

func (s *mapStorage) Get(key string) ([]byte, error) { return s.get(key) }

func (s *mapStorage) get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[key]
	if !ok {
		return nil, nil
	}
	return v, nil
}

func (s *mapStorage) SetWithContext(_ context.Context, key string, val []byte, _ time.Duration) error {
	return s.set(key, val)
}

func (s *mapStorage) Set(key string, val []byte, _ time.Duration) error { return s.set(key, val) }

func (s *mapStorage) set(key string, val []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
	return nil
}

func (s *mapStorage) DeleteWithContext(_ context.Context, key string) error { return s.del(key) }

func (s *mapStorage) Delete(key string) error { return s.del(key) }

func (s *mapStorage) del(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *mapStorage) ResetWithContext(_ context.Context) error { return s.reset() }

func (s *mapStorage) Reset() error { return s.reset() }

func (s *mapStorage) reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string][]byte)
	return nil
}

func (s *mapStorage) Close() error { return nil }

// withSharedRateLimitStorage injects one shared storage, mirroring the Redis setup where all limiters point at the same keys.
func withSharedRateLimitStorage(t *testing.T) {
	t.Helper()
	shared := newMapStorage()
	prev := rateLimitStorage
	rateLimitStorage = func() fiber.Storage { return shared }
	t.Cleanup(func() { rateLimitStorage = prev })
}

func limiterTestApp(t *testing.T) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Get("/gen", GeneralLimiter(), func(c fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/auth", AuthLimiter(), func(c fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func hit(t *testing.T, app *fiber.App, route string) int {
	t.Helper()
	res, err := app.Test(httptest.NewRequest(http.MethodGet, route, http.NoBody), fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	return res.StatusCode
}

// TestLimiterClassesDoNotShareCounters proves general traffic cannot eat the auth budget: bare IP keys in one shared storage would merge every limiter into a single counter.
func TestLimiterClassesDoNotShareCounters(t *testing.T) {
	withSharedRateLimitStorage(t)
	app := limiterTestApp(t)
	authMax := configs.Get().RateLimit.Auth
	for i := 0; i < authMax+5; i++ {
		require.Equal(t, http.StatusOK, hit(t, app, "/gen"), "general hit %d", i)
	}
	assert.Equal(t, http.StatusOK, hit(t, app, "/auth"), "auth budget must not be consumed by general traffic")
}

// TestAuthLimiterEnforcesOwnMax keeps the class limit itself: the (authMax+1)th auth request is still throttled.
func TestAuthLimiterEnforcesOwnMax(t *testing.T) {
	withSharedRateLimitStorage(t)
	app := limiterTestApp(t)
	authMax := configs.Get().RateLimit.Auth
	for i := 0; i < authMax; i++ {
		require.Equal(t, http.StatusOK, hit(t, app, "/auth"), "auth hit %d", i)
	}
	assert.Equal(t, http.StatusTooManyRequests, hit(t, app, "/auth"))
}
