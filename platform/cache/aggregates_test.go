package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAggKeyNormalization(t *testing.T) {
	a := AggKey("org-1", "dashboard:stats", "USD")
	if AggKey("org-1", "dashboard:stats", " usd ") != a {
		t.Error("expected case/space-insensitive currency")
	}
	if AggKey("org-1", "dashboard:stats", "") == a {
		t.Error("expected empty currency to differ from USD")
	}
	if AggScope("org-1") != "agg:v1:org-1:" {
		t.Errorf("unexpected scope %q", AggScope("org-1"))
	}
	if AggKey("org-1", "reports", "USD") == a || AggKey("org-2", "dashboard:stats", "USD") == a {
		t.Error("expected kind and org to yield distinct keys")
	}
}

func TestMemoryAggRoundTrip(t *testing.T) {
	ctx := context.Background()
	a := newMemoryAgg()

	if _, err := a.Get(ctx, "k"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expected miss, got %v", err)
	}
	if err := a.Set(ctx, "k", []byte(`{"a":1}`), time.Minute); err != nil {
		t.Fatal(err)
	}
	raw, err := a.Get(ctx, "k")
	if err != nil || string(raw) != `{"a":1}` {
		t.Fatalf("expected stored value, got %q (%v)", raw, err)
	}
	if err := a.Set(ctx, "k", []byte(`x`), time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	if _, err := a.Get(ctx, "k"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expected expiry, got %v", err)
	}
}

func TestInvalidateOrgScope(t *testing.T) {
	ctx := context.Background()
	a := mustAgg(t)
	_ = a.Set(ctx, AggKey("org-1", "reports", "USD"), []byte(`1`), time.Minute)
	_ = a.Set(ctx, AggKey("org-1", "dashboard:stats", ""), []byte(`2`), time.Minute)
	_ = a.Set(ctx, AggKey("org-2", "reports", "USD"), []byte(`3`), time.Minute)

	if err := InvalidateOrg(ctx, "org-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Get(ctx, AggKey("org-1", "reports", "USD")); !errors.Is(err, ErrCacheMiss) {
		t.Error("expected org-1 entries invalidated")
	}
	if _, err := a.Get(ctx, AggKey("org-2", "reports", "USD")); err != nil {
		t.Errorf("expected org-2 entry kept, got %v", err)
	}
}

type aggSample struct {
	Total int `json:"total"`
}

func TestFetchJSONHitMiss(t *testing.T) {
	// Memory backend via the singleton (REDIS_HOST empty in tests).
	ctx := context.Background()
	key := AggKey("fetch-user", "reports", "USD")
	_ = mustAgg(t).DeletePrefix(ctx, AggScope("fetch-user"))

	var calls atomic.Int32
	fetch := func(context.Context) (aggSample, error) {
		calls.Add(1)
		return aggSample{Total: 7}, nil
	}
	first, err := FetchJSON(ctx, key, fetch)
	if err != nil || first.Total != 7 || calls.Load() != 1 {
		t.Fatalf("expected miss+fetch, got %+v (%v) in %d calls", first, err, calls.Load())
	}
	second, err := FetchJSON(ctx, key, fetch)
	if err != nil || second.Total != 7 || calls.Load() != 1 {
		t.Fatalf("expected hit without fetch, got %+v (%v) in %d calls", second, err, calls.Load())
	}
}

func TestFetchJSONSingleflight(t *testing.T) {
	ctx := context.Background()
	key := AggKey("flight-user", "reports", "USD")
	_ = mustAgg(t).DeletePrefix(ctx, AggScope("flight-user"))

	var calls atomic.Int32
	release := make(chan struct{})
	fetch := func(context.Context) (aggSample, error) {
		calls.Add(1)
		<-release
		return aggSample{Total: 9}, nil
	}
	const waiters = 8
	results := make([]aggSample, waiters)
	errs := make([]error, waiters)
	var wg sync.WaitGroup
	for i := 0; i < waiters; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = FetchJSON(ctx, key, fetch)
		}(i)
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("expected exactly 1 fetch, got %d", calls.Load())
	}
	for i := range results {
		if errs[i] != nil || results[i].Total != 9 {
			t.Fatalf("waiter %d got %+v (%v)", i, results[i], errs[i])
		}
	}
}

func TestFetchJSONFetchError(t *testing.T) {
	ctx := context.Background()
	key := AggKey("err-user", "reports", "USD")
	_ = mustAgg(t).DeletePrefix(ctx, AggScope("err-user"))

	want := errors.New("db down")
	_, err := FetchJSON(ctx, key, func(context.Context) (aggSample, error) {
		return aggSample{}, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected fetch error through, got %v", err)
	}
	// Failures are not cached: a later success works.
	v, err := FetchJSON(ctx, key, func(context.Context) (aggSample, error) {
		return aggSample{Total: 1}, nil
	})
	if err != nil || v.Total != 1 {
		t.Fatalf("expected recovery, got %+v (%v)", v, err)
	}
}

func mustAgg(t *testing.T) AggCache {
	t.Helper()
	ac, err := Aggregates()
	if err != nil {
		t.Fatalf("expected aggregate cache, got %v", err)
	}
	return ac
}
