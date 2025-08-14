package ygggo_mysql

import (
	"container/list"
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
)

// stmtCache implements a per-connection LRU cache of prepared statements.
type stmtCache struct {
	cap   int
	mu    sync.Mutex
	ll    *list.List                                // front = most recently used
	m     map[string]*list.Element                  // sql -> element
	hits   uint64
	misses uint64
}

type stmtEntry struct {
	key  string
	stmt *sql.Stmt
}

func newStmtCache(capacity int) *stmtCache {
	if capacity < 0 { capacity = 0 }
	return &stmtCache{cap: capacity, ll: list.New(), m: make(map[string]*list.Element)}
}

func (c *stmtCache) getOrPrepare(ctx context.Context, conn *sql.Conn, query string) (*sql.Stmt, bool, error) {
	if c == nil || c.cap == 0 {
		// no caching
		st, err := conn.PrepareContext(ctx, query)
		return st, false, err
	}
	c.mu.Lock()
	if ele, ok := c.m[query]; ok {
		c.ll.MoveToFront(ele)
		atomic.AddUint64(&c.hits, 1)
		st := ele.Value.(*stmtEntry).stmt
		c.mu.Unlock()
		return st, true, nil
	}
	c.mu.Unlock()
	// prepare outside the lock to avoid blocking
	st, err := conn.PrepareContext(ctx, query)
	if err != nil { return nil, false, err }
	c.mu.Lock()
	defer c.mu.Unlock()
	// check again if someone inserted meanwhile (unlikely in Conn scope)
	if ele, ok := c.m[query]; ok {
		// use existing, close the newly prepared one
		_ = st.Close()
		c.ll.MoveToFront(ele)
		atomic.AddUint64(&c.hits, 1)
		return ele.Value.(*stmtEntry).stmt, true, nil
	}
	atomic.AddUint64(&c.misses, 1)
	ele := c.ll.PushFront(&stmtEntry{key: query, stmt: st})
	c.m[query] = ele
	if c.ll.Len() > c.cap {
		c.evictLRU()
	}
	return st, false, nil
}

func (c *stmtCache) evictLRU() {
	back := c.ll.Back()
	if back == nil { return }
	c.ll.Remove(back)
	e := back.Value.(*stmtEntry)
	delete(c.m, e.key)
	_ = e.stmt.Close()
}

func (c *stmtCache) closeAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for e := c.ll.Front(); e != nil; e = e.Next() {
		_ = e.Value.(*stmtEntry).stmt.Close()
	}
	c.ll.Init()
	for k := range c.m { delete(c.m, k) }
}

func (c *stmtCache) stats() (hits, misses uint64, size int) {
	if c == nil { return 0, 0, 0 }
	hits = atomic.LoadUint64(&c.hits)
	misses = atomic.LoadUint64(&c.misses)
	c.mu.Lock()
	size = c.ll.Len()
	c.mu.Unlock()
	return
}

