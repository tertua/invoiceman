package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestRedisBackends validates the Redis session and aggregate stores on a
// real server. It runs only when INVOICEMAN_TEST_REDIS_ADDR (host:port) is
// set; otherwise it skips, so resource-limited builds never need a Redis
// service. Keys are test-scoped and removed afterwards — the server is
// never flushed. The shared singletons are untouched (structs are built
// directly), so this test cannot pin global state.
func TestRedisBackends(t *testing.T) {
	addr := os.Getenv("INVOICEMAN_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set INVOICEMAN_TEST_REDIS_ADDR (host:port) to run")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("expected Redis ping, got: %v", err)
	}

	prefix := "test:" + t.Name() + ":"
	t.Cleanup(func() {
		iter := client.Scan(ctx, 0, prefix+"*", 100).Iterator()
		for iter.Next(ctx) {
			_ = client.Del(ctx, iter.Val()).Err()
		}
	})

	// Session store round trip.
	ss := &redisStore{client: client}
	skey := prefix + "session:u1"
	if err := ss.Set(ctx, skey, "refresh-token", time.Minute); err != nil {
		t.Fatalf("expected session set, got: %v", err)
	}
	val, err := ss.Get(ctx, skey)
	if err != nil || val != "refresh-token" {
		t.Fatalf("expected session hit, got %q (%v)", val, err)
	}
	if err := ss.Delete(ctx, skey); err != nil {
		t.Fatalf("expected session delete, got: %v", err)
	}
	if _, err := ss.Get(ctx, skey); err == nil {
		t.Fatal("expected miss after session delete")
	}

	// Aggregate store round trip, including prefix invalidation.
	ag := &redisAgg{client: client}
	akey := prefix + "agg:v1:u1:reports:USD"
	other := prefix + "agg:v1:u2:reports:USD"
	if err := ag.Set(ctx, akey, []byte(`{"t":1}`), time.Minute); err != nil {
		t.Fatalf("expected agg set, got: %v", err)
	}
	if err := ag.Set(ctx, other, []byte(`{"t":2}`), time.Minute); err != nil {
		t.Fatalf("expected agg set, got: %v", err)
	}
	if err := ag.DeletePrefix(ctx, prefix+"agg:v1:u1:"); err != nil {
		t.Fatalf("expected prefix delete, got: %v", err)
	}
	if _, err := ag.Get(ctx, akey); err == nil {
		t.Fatal("expected u1 aggregate invalidated")
	}
	raw, err := ag.Get(ctx, other)
	if err != nil || string(raw) != `{"t":2}` {
		t.Fatalf("expected u2 aggregate kept, got %q (%v)", raw, err)
	}
}
