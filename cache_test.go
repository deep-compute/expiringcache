package expiringcache

import (
	"testing"
	"time"
)

func TestExpiringCache(t *testing.T) {
	cache := Cache{Duration: 1, Max: 1, NEvictions: 10}
	cache.Init()

	cache.Put("a", 1)

	if cache.Count() != 1 {
		t.Errorf("Put failed to add entry")
	}

	time.Sleep(100 * time.Millisecond)
	if cache.Count() != 1 {
		t.Errorf("Expired too soon")
	}

	time.Sleep(2 * time.Second)
	if cache.Count() != 1 {
		t.Errorf("Expiry happened without reaching max capacity")
	}

	cache.Put("b", 2)
	if cache.Count() != 1 {
		t.Errorf("Key 'a' not expired even on reaching max capacity")
	}

	if cache.Get("b").(int) != 2 {
		t.Errorf("Get did not fetch correct value")
	}
}
