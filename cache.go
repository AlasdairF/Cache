package cache

import (
	"time"
	"sync"
	"sync/atomic"
)

const (
	sleeptime = time.Minute
)

var caches1 []*Bytes
var caches2 []*Interface
var cachesMutex1 sync.Mutex
var cachesMutex2 sync.Mutex

type entryBytes struct {
	mutex sync.Mutex
	lastAccess int64
	data []byte
}

type Bytes struct {
	vals []*entryBytes
	mem, max int64
	size int
}

type entryInterface struct {
	mutex sync.Mutex
	lastAccess int64
	data interface{}
	size int64
}

type Interface struct {
	vals []*entryInterface
	mem, max int64
	size int
}

func init() {
	go cleaner()
}

// Creates a new cache
func NewBytes(size int, megabytes int64) *Bytes {
	c := &Bytes{vals: make([]*entryBytes, size), size: size, max: megabytes * 1048576}
	cachesMutex1.Lock()
	caches1 = append(caches1, c)
	cachesMutex1.Unlock()
	return c
}

// Gets the slice of bytes assigned to this ID in the cache
func (c *Bytes) Get(id int) ([]byte, bool) {
	if id >= c.size {
		return nil, false
	}
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
func (c *Bytes) Store(id int, p []byte) {
	if id >= c.size {
		return
	}
	item := c.vals[id]
	if item == nil {
		item = &entryBytes{data: p, lastAccess: time.Now().Unix()}
		c.vals[id] = item
		atomic.AddInt64(&c.mem, int64(len(p)))
	}
}

// Caches the item, replaces it if it already exists
func (c *Bytes) Replace(id int, p []byte) {
	if id >= c.size {
		return
	}
	tim := time.Now().Unix()
	item := c.vals[id]
	if item == nil {
		item = &entryBytes{data: p, lastAccess: tim}
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

// Delete an entry from the cache
func (c *Bytes) Remove(id int) {
	if id >= c.size {
		return
	}
	c.vals[id] = false
}

// Closes the cache, releasing the memory
func (c *Bytes) Close() {
	c.size = 0
	c.vals = nil
	c.mem = 0
	c.max = 0
}

// Removes all entries in the cache last accessed less than this time ago (UNIX Timestamp)
func (c *Bytes) Purge(olderThan int64) {
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

// Creates a new cache
func New(size int, megabytes int64) *Interface {
	c := &Interface{vals: make([]*entryInterface, size), size: size, max: megabytes * 1024}
	cachesMutex2.Lock()
	caches2 = append(caches2, c)
	cachesMutex2.Unlock()
	return c
}

// Gets the slice of bytes assigned to this ID in the cache
func (c *Interface) Get(id int) (interface{}, bool) {
	if id >= c.size {
		return nil, false
	}
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
func (c *Interface) Store(id int, p interface{}, size int64) {
	if id >= c.size {
		return
	}
	item := c.vals[id]
	if item == nil {
		item = &entryInterface{data: p, lastAccess: time.Now().Unix(), size: size}
		c.vals[id] = item
		atomic.AddInt64(&c.mem, size)
	}
}

// Caches the item, replaces it if it already exists
func (c *Interface) Replace(id int, p []byte, size int64) {
	if id >= c.size {
		return
	}
	tim := time.Now().Unix()
	item := c.vals[id]
	if item == nil {
		item = &entryInterface{data: p, lastAccess: tim, size: size}
		c.vals[id] = item
		atomic.AddInt64(&c.mem, size)
	} else {
		item.mutex.Lock()
		size -= item.size
		item.data = p
		item.lastAccess = tim
		item.mutex.Unlock()
		atomic.AddInt64(&c.mem, size)
	}
}

// Delete an entry from the cache
func (c *Interface) Remove(id int) {
	if id >= c.size {
		return
	}
	c.vals[id] = false
}

// Closes the cache, releasing the memory
func (c *Interface) Close() {
	c.size = 0
	c.vals = nil
	c.mem = 0
	c.max = 0
}

// Removes all entries in the cache last accessed less than this time ago (UNIX Timestamp)
func (c *Interface) Purge(olderThan int64) {
	for i, item := range c.vals {
		if item != nil {
			item.mutex.Lock()
			if item.lastAccess < olderThan {
				atomic.AddInt64(&c.mem, 0 - item.size)
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
		// Clean []byte cache
		cachesMutex1.Lock()
		newCachesSlice1 := caches1
		cachesMutex1.Unlock()
		for _, c := range newCachesSlice1 {
			if atomic.LoadInt64(&c.mem) > atomic.LoadInt64(&c.max) {
				c.Purge(time.Now().Unix() - 432000) // 5 days ago
			} else {
				continue
			}
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
		// Clean interface{} cache
		cachesMutex2.Lock()
		newCachesSlice2 := caches2
		cachesMutex2.Unlock()
		for _, c := range newCachesSlice2 {
			if atomic.LoadInt64(&c.mem) > atomic.LoadInt64(&c.max) {
				c.Purge(time.Now().Unix() - 432000) // 5 days ago
			} else {
				continue
			}
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
