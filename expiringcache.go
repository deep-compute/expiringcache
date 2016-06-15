// Implements an expiring cache
package expiringcache

import (
	"github.com/prashanthellina/go-avltree"
	"math/rand"
	"sync"
	"time"
)

type CacheValue struct {
	Key      string
	Value    interface{}
	ExpireAt int64
}

func (p CacheValue) Compare(b avltree.Interface) int {
	if p.Key < b.(*CacheValue).Key {
		return -1
	}

	if p.Key > b.(*CacheValue).Key {
		return 1
	}

	return 0
}

// Cache that supports expiry of keys
type Cache struct {
	Duration int // Number of seconds to keep key
	// in cache
	Max        int // max number of keys in cache
	NEvictions int // number of evictions to perform
	// when keys reaches max limit
	NSamples int // number of keys to consider for

	// Interval in seconds between which evictions are done periodically
	// By default this is 0 i.e. disabled
	PeriodicEvictionInterval uint64
	// performing an eviction
	data *avltree.ObjectTree
	sync.Mutex
}

func (p *Cache) Init() {
	p.data = avltree.NewObjectTree(0)
	if p.PeriodicEvictionInterval == 0 {
		return
	}

	go p.evictPeriodically()
}

func (p *Cache) evictPeriodically() {
	numSeconds := time.Duration(p.PeriodicEvictionInterval) * time.Second
	var cv *CacheValue
	for {
		time.Sleep(numSeconds)
		p.Lock()
		now := time.Now().UTC().Unix()
		to_remove := make([]*CacheValue, 0)
		for v := range p.data.Iter() {
			cv = v.(*CacheValue)
			// if it is going to expire in the future, leave it
			if cv.ExpireAt > now {
				continue
			}
			// it should expire now. add it to things to remove
			to_remove = append(to_remove, cv)
		}

		for _, cv = range to_remove {
			p.data.Remove(cv)
		}

		p.Unlock()
	}
}

func (p *Cache) Put(key string, value interface{}) {
	p.PutWithExpiry(key, value, p.Duration)
}

func (p *Cache) PutWithExpiry(key string, value interface{}, duration int) {
	p.Lock()

	p.update()

	v := CacheValue{ExpireAt: time.Now().UTC().Unix() + int64(duration),
		Key: key, Value: value}

	// Add kv to data
	av, is_dup := p.data.Add(&v)
	if is_dup {
		// If already exists, update value
		_v := av.(*CacheValue)
		_v.Value = value
	}

	p.Unlock()
}

func (p *Cache) Get(key string) interface{} {
	var r interface{} = nil
	p.Lock()

	v := p.data.Find(&CacheValue{Key: key})
	if v != nil {
		r = v.(*CacheValue).Value
	}

	p.Unlock()
	return r
}

func (p *Cache) Del(key string) {
	p.Lock()
	p.data.Remove(&CacheValue{Key: key})
	p.Unlock()
}

func (p *Cache) PopRandom() interface{} {
	var r interface{} = nil

	p.Lock()

	length := p.data.Len()
	if length != 0 {
		index := rand.Intn(p.data.Len())

		v := p.data.At(index).(*CacheValue)
		p.data.Remove(v)

		r = v.Value
	}

	p.Unlock()
	return r
}

func (p *Cache) Exists(key string) bool {
	p.Lock()
	v := p.data.Find(&CacheValue{Key: key})
	p.Unlock()
	return v != nil
}

func (p *Cache) Count() int {
	p.Lock()
	count := p.data.Len()
	p.Unlock()
	return count
}

func (p *Cache) Iter() <-chan *CacheValue {
	wc := make(chan *CacheValue)
	rc := p.data.Iter()

	go func() {
		for v := range rc {
			wc <- v.(*CacheValue)
		}

		close(wc)
	}()

	return wc
}

func (p *Cache) evictKey() {
	n := p.NSamples
	if n == 0 {
		n = 1
	}

	// init min ts to a big value in the future (for min ts finding
	// logic below to work)
	var min_ts int64 = time.Now().UTC().Unix() + (365 * 86400)
	var min_v *CacheValue = nil

	for i := 0; i < n; i++ {
		v := p.data.At(rand.Intn(p.data.Len())).(*CacheValue)
		if v.ExpireAt < min_ts {
			min_ts = v.ExpireAt
			min_v = v
		}
	}

	if min_v != nil {
		p.data.Remove(min_v)
	}
}

func (p *Cache) update() {

	if p.Max == 0 || p.data.Len() < p.Max {
		return
	}

	// Make space by removing keys
	// Break when keys become empty
	for i := 0; i < p.NEvictions && p.data.Len() > 0; i++ {
		p.evictKey()
	}
}
