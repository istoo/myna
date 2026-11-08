package agent

import (
	"context"
	"runtime"
	"sync"
	atomicpkg "sync/atomic"
	"testing"
	"time"
)

// resetSpotlight clears the cache a test shares with the package.
func resetSpotlight(t *testing.T) {
	t.Helper()
	spotlight.Lock()
	spotlight.m = nil
	spotlight.w = nil
	spotlight.Unlock()
}

// waitSpotlight waits until done() holds and no query is running, so a
// test's spotlightRun is restored once nothing reads it.
func waitSpotlight(t *testing.T, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		spotlight.Lock()
		running := len(spotlight.w)
		spotlight.Unlock()
		if running == 0 && done() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("Spotlight queries did not finish")
}

// TestSpotlightAsksForEveryAppAtOnce is the cold Agents read: the first
// bundle id asked starts every other app's query, so the fifteen reads
// that follow are cached instead of each waiting for its own mdfind.
func TestSpotlightAsksForEveryAppAtOnce(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Spotlight is macOS's")
	}
	t.Setenv(spotlightSandbox, "")
	resetSpotlight(t)
	var mu sync.Mutex
	called := map[string]int{}
	var inflight atomicpkg.Int32
	old := spotlightRun
	spotlightRun = func(ctx context.Context, id string) ([]byte, error) {
		inflight.Add(1)
		defer inflight.Add(-1)
		mu.Lock()
		called[id]++
		mu.Unlock()
		return []byte("/Applications/" + id + ".app\n"), nil
	}
	defer func() { spotlightRun = old }()

	ids := spotlightIDs()
	if len(ids) < 2 {
		t.Fatalf("known desktop app ids: %v", ids)
	}
	if got := spotlightBundles("com.openai.codex"); len(got) != 1 || got[0] != "/Applications/com.openai.codex.app" {
		t.Fatalf("spotlightBundles(codex) = %v", got)
	}
	waitSpotlight(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(called) == len(ids)
	})
	mu.Lock()
	for _, id := range ids {
		if called[id] != 1 {
			mu.Unlock()
			t.Errorf("%s was asked %d times, want once (the first ask starts every app's)", id, called[id])
			return
		}
	}
	mu.Unlock()
	// cached: another read asks Spotlight nothing
	spotlightBundles("com.openai.codex")
	mu.Lock()
	if called["com.openai.codex"] != 1 {
		t.Errorf("codex asked %d times after the cache, want once", called["com.openai.codex"])
	}
	mu.Unlock()
}

// TestSpotlightOneQueryPerID has two reads ask for the same id at once:
// the second waits for the first's answer instead of asking mdfind again.
func TestSpotlightOneQueryPerID(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Spotlight is macOS's")
	}
	t.Setenv(spotlightSandbox, "")
	resetSpotlight(t)
	var calls atomicpkg.Int32 // codex's alone: the other apps' are prefetched
	release := make(chan struct{})
	old := spotlightRun
	spotlightRun = func(ctx context.Context, id string) ([]byte, error) {
		if id != "com.openai.codex" {
			return nil, nil
		}
		calls.Add(1)
		<-release
		return []byte("/Applications/" + id + ".app\n"), nil
	}
	defer func() { spotlightRun = old }()

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := spotlightBundles("com.openai.codex"); len(got) != 1 {
				t.Errorf("spotlightBundles(codex) = %v", got)
			}
		}()
	}
	// let both reach the query before it is released
	deadline := time.Now().Add(2 * time.Second)
	for calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	close(release)
	wg.Wait()
	waitSpotlight(t, func() bool { return true })
	if n := calls.Load(); n != 1 {
		t.Errorf("mdfind ran %d times for one id, want once", n)
	}
}

// TestSpotlightStaleAsksEveryAppAgain: once the answers are older than
// spotlightFresh, the next miss starts every app's query again, not only
// its own, so a warm Agents read every five minutes isn't fifteen queries
// one after another.
func TestSpotlightStaleAsksEveryAppAgain(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Spotlight is macOS's")
	}
	t.Setenv(spotlightSandbox, "")
	resetSpotlight(t)
	ids := spotlightIDs()
	var mu sync.Mutex
	called := map[string]int{}
	old := spotlightRun
	spotlightRun = func(ctx context.Context, id string) ([]byte, error) {
		mu.Lock()
		called[id]++
		mu.Unlock()
		return nil, nil
	}
	defer func() { spotlightRun = old }()
	// a first read, every app asked
	spotlightBundles("com.openai.codex")
	waitSpotlight(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(called) == len(ids)
	})
	// five minutes on: every answer stale
	stale := time.Now().Add(-spotlightFresh - time.Minute)
	spotlight.Lock()
	for id, a := range spotlight.m {
		a.at = stale
		spotlight.m[id] = a
	}
	spotlight.Unlock()
	mu.Lock()
	clear(called)
	mu.Unlock()
	spotlightBundles("com.openai.codex")
	waitSpotlight(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(called) == len(ids)
	})
}
