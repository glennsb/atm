//http://openmymind.net/Shard-Your-Hash-table-to-reduce-write-locks/
package atm

import (
	"crypto/sha1"
	"fmt"
	"sync"
)

type Cache map[string]*shard

type shard struct {
	items map[string]string
	lock  *sync.RWMutex
}

func NewCache() Cache {
	c := make(Cache, 256)
	for i := 0; i < 256; i++ {
		c[fmt.Sprintf("%02x", i)] = &shard{
			items: make(map[string]string, 2048),
			lock:  new(sync.RWMutex),
		}
	}
	return c
}

func (c Cache) Get(key string) string {
	shard := c.GetShard(key)
	shard.lock.RLock()
	defer shard.lock.RUnlock()
	return shard.items[key]
}

func (c Cache) Set(key string, data string) {
	shard := c.GetShard(key)
	shard.lock.Lock()
	defer shard.lock.Unlock()
	shard.items[key] = data
}

func (c Cache) Delete(key string) {
	shard := c.GetShard(key)
	shard.lock.Lock()
	defer shard.lock.Unlock()
	delete(shard.items, key)
}

func (c Cache) GetShard(key string) (shard *shard) {
	hasher := sha1.New()
	hasher.Write([]byte(key))
	shardKey := fmt.Sprintf("%x", hasher.Sum(nil))[0:2]
	return c[shardKey]
}
