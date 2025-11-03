package counter

import "sync"

type CommonCounter struct {
	counter int
	mu      sync.RWMutex
}

func (c *CommonCounter) Increment() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counter++
	return c.counter
}

func (c *CommonCounter) Counter() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.counter
}
