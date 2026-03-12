package cache

import (
	"sync"
	"testing"
	"time"
)

func TestGetSetBasic(t *testing.T) {
	c := New(time.Minute)
	c.Set("foo", "bar")

	v, ok := c.Get("foo")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if v != "bar" {
		t.Fatalf("expected bar, got %v", v)
	}
}

func TestGetMiss(t *testing.T) {
	c := New(time.Minute)
	_, ok := c.Get("missing")
	if ok {
		t.Fatal("expected cache miss")
	}
}

func TestTTLExpiry(t *testing.T) {
	c := New(time.Millisecond)
	c.Set("key", "value")

	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("key")
	if ok {
		t.Fatal("expected cache miss after TTL expiry")
	}
}

func TestInvalidateByPrefix(t *testing.T) {
	c := New(time.Minute)
	c.Set("ticket:get:1:users", "t1")
	c.Set("ticket:get:2:users", "t2")
	c.Set("ticket:list:50", "list")
	c.Set("search:query1", "s1")

	c.Invalidate("ticket:get:1:")

	if _, ok := c.Get("ticket:get:1:users"); ok {
		t.Fatal("expected ticket:get:1:users to be invalidated")
	}
	if _, ok := c.Get("ticket:get:2:users"); !ok {
		t.Fatal("ticket:get:2:users should not be invalidated")
	}
	if _, ok := c.Get("ticket:list:50"); !ok {
		t.Fatal("ticket:list should not be invalidated")
	}
	if _, ok := c.Get("search:query1"); !ok {
		t.Fatal("search should not be invalidated")
	}
}

func TestInvalidateMultiplePrefixes(t *testing.T) {
	c := New(time.Minute)
	c.Set("ticket:list:a", "1")
	c.Set("search:q", "2")
	c.Set("ticket:get:5:", "3")

	c.Invalidate("ticket:list:", "search:")

	if _, ok := c.Get("ticket:list:a"); ok {
		t.Fatal("expected ticket:list to be invalidated")
	}
	if _, ok := c.Get("search:q"); ok {
		t.Fatal("expected search to be invalidated")
	}
	if _, ok := c.Get("ticket:get:5:"); !ok {
		t.Fatal("ticket:get:5 should not be invalidated")
	}
}

func TestClear(t *testing.T) {
	c := New(time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)

	c.Clear()

	if _, ok := c.Get("a"); ok {
		t.Fatal("expected all entries cleared")
	}
	if _, ok := c.Get("b"); ok {
		t.Fatal("expected all entries cleared")
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := New(time.Minute)
	var wg sync.WaitGroup

	// Concurrent writers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key:" + string(rune('a'+i%26))
			c.Set(key, i)
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key:" + string(rune('a'+i%26))
			c.Get(key)
		}(i)
	}

	// Concurrent invalidation
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Invalidate("key:")
		}()
	}

	wg.Wait()
}
