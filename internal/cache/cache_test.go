package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSetBasic(t *testing.T) {
	c := New(time.Minute)
	c.Set("foo", "bar")

	v, ok := c.Get("foo")
	require.True(t, ok, "expected cache hit")
	assert.Equal(t, "bar", v)
}

func TestGetMiss(t *testing.T) {
	c := New(time.Minute)
	_, ok := c.Get("missing")
	assert.False(t, ok, "expected cache miss")
}

func TestTTLExpiry(t *testing.T) {
	c := New(time.Millisecond)
	c.Set("key", "value")

	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("key")
	assert.False(t, ok, "expected cache miss after TTL expiry")
}

func TestInvalidateByPrefix(t *testing.T) {
	c := New(time.Minute)
	c.Set("ticket:get:1:users", "t1")
	c.Set("ticket:get:2:users", "t2")
	c.Set("ticket:list:50", "list")
	c.Set("search:query1", "s1")

	c.Invalidate("ticket:get:1:")

	_, ok := c.Get("ticket:get:1:users")
	assert.False(t, ok, "expected ticket:get:1:users to be invalidated")

	_, ok = c.Get("ticket:get:2:users")
	assert.True(t, ok, "ticket:get:2:users should not be invalidated")

	_, ok = c.Get("ticket:list:50")
	assert.True(t, ok, "ticket:list should not be invalidated")

	_, ok = c.Get("search:query1")
	assert.True(t, ok, "search should not be invalidated")
}

func TestInvalidateMultiplePrefixes(t *testing.T) {
	c := New(time.Minute)
	c.Set("ticket:list:a", "1")
	c.Set("search:q", "2")
	c.Set("ticket:get:5:", "3")

	c.Invalidate("ticket:list:", "search:")

	_, ok := c.Get("ticket:list:a")
	assert.False(t, ok, "expected ticket:list to be invalidated")

	_, ok = c.Get("search:q")
	assert.False(t, ok, "expected search to be invalidated")

	_, ok = c.Get("ticket:get:5:")
	assert.True(t, ok, "ticket:get:5 should not be invalidated")
}

func TestClear(t *testing.T) {
	c := New(time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)

	c.Clear()

	_, ok := c.Get("a")
	assert.False(t, ok, "expected all entries cleared")

	_, ok = c.Get("b")
	assert.False(t, ok, "expected all entries cleared")
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
