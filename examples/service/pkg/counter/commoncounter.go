package counter

import "sync"

type CommonCounter struct {
	Counter int
	mu      sync.RWMutex
}

func (c *CommonCounter) Increment() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Counter++
	return c.Counter
}
