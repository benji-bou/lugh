package graph

import (
	"context"
	"sync"
)

type Context interface {
	context.Context
	Initialized()
}

type SyncContext interface {
	Context
	Initializing()
	Synchronize()
}

type WaitGroupContext struct {
	context.Context
	wg       sync.WaitGroup
	barrierC chan struct{}
}

func NewContext(parent context.Context) SyncContext {
	return &WaitGroupContext{Context: parent, wg: sync.WaitGroup{}, barrierC: make(chan struct{})}
}
func (c *WaitGroupContext) Initializing() {
	c.wg.Add(1)
}

func (c *WaitGroupContext) Initialized() {
	c.wg.Done()
	<-c.barrierC
}
func (c *WaitGroupContext) Synchronize() {
	c.wg.Wait()
	close(c.barrierC)
}
