package cache

import (
	"time"
	"sync"
)

const (
	sleeptime = time.Minute
)

caches []*cache
cacheMutex sync.Mutex

type entry struct {
	mutex sync.Mutex
	lastAccess int64
	data []byte
}

type cache struct {
	vals []*entry
	mem, max int64
}

func init() {
	go cleaner()
}

// Creates a new cache
func New(size int, megabytes int64) *cache {
	c := &cache{vals: make([]*entry, size), max: megabytes * 1024}
	cacheMutex.Lock()
	caches = append(caches, c)
	cacheMutex.Unlock()
	return c
}

// Gets the slice of bytes assigned to this ID in the cache
func (c *cache) Get(id int) ([]byte, bool) {
	item := c.vals[id]
	if item == nil {
		return nil, false
	}
	tim := time.Now().Unix()
	item.mutex.Lock()
	item.lastAccess = tim
	res := item.data
	item.mutex.Unlock()
	return res, true
}

// Caches the item, does nothing if it already exists
func (c *cache) Store(id int, p []byte) {
	item := c.vals[id]
	if item == nil {
		item = &entry{data: p, lastAccess: time.Now().Unix()}
		c.vals[id] = item
		atomic.AddInt64(&c.mem, int64(len(p)))
	}
}

// Caches the item, replaces it if it already exists
func (c *cache) Replace(id int, p []byte) {
	tim := time.Now().Unix()
	item := c.vals[id]
	if item == nil {
		item = &entry{data: p, lastAccess: tim}
		c.vals[id] = item
		atomic.AddInt64(&c.mem, int64(len(p)))
	} else {
		item.mutex.Lock()
		memdif := len(p) - len(item.data)
		item.data = p
		item.lastAccess = tim
		item.mutex.Unlock()
		atomic.AddInt64(&c.mem, int64(memdif))
	}
}

// Closes the cache, releasing the memory
func (c *cache) Close() {
	c.vals = nil
	mem, max = 0
}

// Removes all entries in the cache last accessed less than this time ago (UNIX Timestamp)
func (c *cache) Purge(olderThan int64) {
	for i, item := range c.vals {
		if item != nil {
			item.mutex.Lock()
			if item.lastAccess < olderThan {
				atomic.AddInt64(&c.mem, 0 - int64(len(item.data)))
				c.vals[i] = nil
			}
			item.mutex.Unlock()
		}
	}
}

// Automatically purges all caches
func cleaner() {
	for {
		time.Sleep(sleeptime)
		cacheMutex.Lock()
		newCachesSlice := caches
		cacheMutex.Unlock()
		for _, c := range newCachesSlice {
			if atomic.LoadInt64(&c.mem) > atomic.LoadInt64(&c.max) {
				c.Purge(time.Now().Unix() - 86400) // 1 day ago
			} else {
				continue
			}
			if atomic.LoadInt64(&c.mem) > atomic.LoadInt64(&c.max) {
				c.Purge(time.Now().Unix() - 3600) // 1 hour ago
			} else {
				continue
			}
			if atomic.LoadInt64(&c.mem) > atomic.LoadInt64(&c.max) {
				c.Purge(time.Now().Unix() - 600) // 10 minutes ago
			} else {
				continue
			}
			if atomic.LoadInt64(&c.mem) > atomic.LoadInt64(&c.max) {
				c.Purge(time.Now().Unix() + 3600) // everything
			} else {
				continue
			}
		}
	}
}
