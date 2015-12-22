package cache

import (
	"time"
	"sync"
)

const (
	sleeptime = time.Minute
)

caches []*bytes
bytesMutex sync.Mutex

type entry struct {
	mutex sync.Mutex
	lastAccess int64
	data []byte
}

type bytes struct {
	vals []*entry
	mem, max int64
}

func init() {
	go cleaner()
}

// Creates a new cache
func New(size int, megabytes int64) *bytes {
	c := &bytes{vals: make([]*entry, size), max: megabytes * 1024}
	bytesMutex.Lock()
	caches = append(caches, c)
	bytesMutex.Unlock()
	return c
}

// Gets the slice of bytes assigned to this ID in the cache
func (c *bytes) Get(id int) ([]byte, bool) {
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
func (c *bytes) Store(id int, p []byte) {
	item := c.vals[id]
	if item == nil {
		item = &entry{data: p, lastAccess: time.Now().Unix()}
		c.vals[id] = item
		atomic.AddInt64(&c.mem, int64(len(p)))
	}
}

// Caches the item, replaces it if it already exists
func (c *bytes) Replace(id int, p []byte) {
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
func (c *bytes) Close() {
	c.vals = nil
	mem, max = 0
}

// Removes all entries in the cache last accessed less than this time ago (UNIX Timestamp)
func (c *bytes) Purge(olderThan int64) {
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
		bytesMutex.Lock()
		newCachesSlice := caches
		bytesMutex.Unlock()
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
