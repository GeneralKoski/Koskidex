package engine

import "testing"

func TestLRUCacheGetPut(t *testing.T) {
	c := NewLRUCache(2)

	c.Put("a", 1)
	c.Put("b", 2)

	if v, ok := c.Get("a"); !ok || v.(int) != 1 {
		t.Fatalf("expected a=1, got %v ok=%v", v, ok)
	}
	if _, ok := c.Get("missing"); ok {
		t.Fatal("expected miss for unknown key")
	}
}

func TestLRUCacheEviction(t *testing.T) {
	c := NewLRUCache(2)

	c.Put("a", 1)
	c.Put("b", 2)
	// Access "a" so "b" becomes least-recently-used.
	c.Get("a")
	c.Put("c", 3)

	if _, ok := c.Get("b"); ok {
		t.Fatal("expected b to be evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatal("expected a to survive eviction")
	}
	if _, ok := c.Get("c"); !ok {
		t.Fatal("expected c to be present")
	}
}

func TestLRUCacheInvalidatePrefix(t *testing.T) {
	c := NewLRUCache(10)

	c.Put("movies|matrix", 1)
	c.Put("movies|alien", 2)
	c.Put("books|dune", 3)

	c.InvalidatePrefix("movies|")

	if _, ok := c.Get("movies|matrix"); ok {
		t.Fatal("expected movies|matrix to be invalidated")
	}
	if _, ok := c.Get("movies|alien"); ok {
		t.Fatal("expected movies|alien to be invalidated")
	}
	if _, ok := c.Get("books|dune"); !ok {
		t.Fatal("expected books|dune to survive")
	}
}
