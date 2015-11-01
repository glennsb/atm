//http://openmymind.net/Shard-Your-Hash-table-to-reduce-write-locks/
package atm

import (
	"crypto/sha1"
	"fmt"
	"runtime"
	"sync"
	"time"
)

type Cache map[string]*shard

type shard struct {
	items map[string]interface{}
	lock  *sync.RWMutex
}

type expiringItem struct {
	Object     interface{}
	Expiration *time.Time
}

func (i *expiringItem) Expired() bool {
	if nil == i.Expiration {
		return false
	}
	return i.Expiration.Before(time.Now().UTC())
}

type ExpiringCache struct {
	*exCache
}

type exCache struct {
	*Cache
	scrubber *scrubber
}

type scrubber struct {
	Interval time.Duration
	stop     chan bool
}

func (s *scrubber) Run(c *exCache) {
	s.stop = make(chan bool)
	ticker := time.NewTimer(s.Interval)
	for {
		select {
		case <-ticker.C:
			c.DeleteExpired()
		case <-s.stop:
			ticker.Stop()
			return
		}
	}
}

func startSrubber(e *exCache, d time.Duration) {
	s := &scrubber{
		Interval: d,
	}
	e.scrubber = s
	go s.Run(e)
}

func stopScrubber(c *ExpiringCache) {
	c.scrubber.stop <- true
}

func NewExpiringCache(scrubInterval time.Duration) *ExpiringCache {
	ex := &exCache{
		Cache: NewCache(),
	}
	E := &ExpiringCache{ex}
	startSrubber(ex, scrubInterval)
	runtime.SetFinalizer(E, stopScrubber)
	return E
}

func (c *ExpiringCache) Set(k string, v interface{}, d time.Duration) {
	t := time.Now().UTC().Add(d)
	o := &expiringItem{
		Object:     v,
		Expiration: &t,
	}
	c.Cache.Set(k, o)
}

func (c *ExpiringCache) Get(k string) (interface{}, bool) {
	i, f := c.Cache.Get(k)
	if !f {
		return nil, f
	}
	o := i.(*expiringItem)
	if o.Expired() {
		return nil, false
	}
	return o.Object, true
}

func NewCache() *Cache {
	c := make(Cache, 256)
	for i := 0; i < 256; i++ {
		c[fmt.Sprintf("%02x", i)] = &shard{
			items: make(map[string]interface{}, 2048),
			lock:  new(sync.RWMutex),
		}
	}
	return &c
}

func (c *exCache) DeleteExpired() {
	for _, shard := range *c.Cache {
		shard.lock.Lock()
		for k, v := range shard.items {
			i := v.(*expiringItem)
			if i.Expired() {
				delete(shard.items, k)
			}
		}
		shard.lock.Unlock()
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	shard := c.getShard(key)
	shard.lock.RLock()
	i, r := shard.items[key]
	shard.lock.RUnlock()
	return i, r
}

func (c *Cache) Set(key string, data interface{}) {
	shard := c.getShard(key)
	shard.lock.Lock()
	shard.items[key] = data
	shard.lock.Unlock()
}

func (c *Cache) Delete(key string) {
	shard := c.getShard(key)
	shard.lock.Lock()
	delete(shard.items, key)
	shard.lock.Unlock()
}

func (c *Cache) getShard(key string) (shard *shard) {
	hasher := sha1.New()
	hasher.Write([]byte(key))
	shardKey := fmt.Sprintf("%x", hasher.Sum(nil))[0:2]
	return (*c)[shardKey]
}
