/*
This file is part of btrsync.

Btrsync is free software: you can redistribute it and/or modify it under the terms of the
GNU Lesser General Public License as published by the Free Software Foundation, either
version 3 of the License, or (at your option) any later version.

Btrsync is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
See the GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License along with btrsync.
If not, see <https://www.gnu.org/licenses/>.
*/

// Package queue provides a concurrent queue for running functions.
package queue

import (
	"context"
	"io"
	"log"
	"sync"
	"time"
)

// QueueFunc is a function that takes no arguments and returns an error.
type QueueFunc func() error

// QueueOption is a function that configures a queue.
type QueueOption func(*queueConfig)

// ConcurrentQueue is a type that can be used to queue up functions to be run concurrently.
// It handles concurrency and error tracking internally, so that the caller doesn't have to
// worry about it. The queue will stop processing functions after the first error it encounters.
type ConcurrentQueue struct {
	cfg               queueConfig
	current           uint
	queue             []QueueFunc
	errs              chan error
	wg                sync.WaitGroup
	queueMux, waiting sync.Mutex

	ctx    context.Context
	cancel func()
}

type queueConfig struct {
	logger      *log.Logger
	verbosity   int
	concurrency int
}

func (q queueConfig) LogVerbose(v int, format string, args ...interface{}) {
	if q.logger != nil && q.verbosity >= v {
		q.logger.Printf(format, args...)
	}
}

func WithMaxConcurrency(concurrency int) QueueOption {
	return func(c *queueConfig) {
		c.concurrency = concurrency
	}
}

func WithLogger(logger *log.Logger, verbosity int) QueueOption {
	return func(c *queueConfig) {
		c.logger = logger
		c.verbosity = verbosity
	}
}

// NewConcurrentQueue returns a new ConcurrentQueue with the given concurrency.
func NewConcurrentQueue(opts ...QueueOption) *ConcurrentQueue {
	ctx, cancel := context.WithCancel(context.Background())
	q := &ConcurrentQueue{
		cfg: queueConfig{
			concurrency: 1,
			logger:      log.New(io.Discard, "", 0),
		},
		queue:  make([]QueueFunc, 0),
		ctx:    ctx,
		cancel: cancel,
	}
	for _, opt := range opts {
		opt(&q.cfg)
	}
	q.errs = make(chan error, q.cfg.concurrency+1)
	go q.run()
	return q
}

// Push adds a function to the queue. If a function before it errors, then the
// the error for the given function will be lost.
func (c *ConcurrentQueue) Push(f QueueFunc) {
	c.queueMux.Lock()
	defer c.queueMux.Unlock()
	c.queue = append(c.queue, f)
}

// Wait blocks until all functions finish and returns the first error that occurred.
// If no errors occurred, then nil is returned. The queue can no longer be used after
// Wait is called. If no functions are pushed to the queue, then nil is returned. If
// functions are added to the queue after Wait is called, there is no guarantee that
// they will be executed.
func (c *ConcurrentQueue) Wait() error {
	c.waiting.Lock()
	c.wg.Wait()
	select {
	case err := <-c.errs:
		return err
	case <-c.ctx.Done():
		return nil
	}
}

func (c *ConcurrentQueue) run() {
	defer c.cancel()
	for {
		if c.cfg.concurrency == -1 || c.current < uint(c.cfg.concurrency) {
			if len(c.queue) > 0 {
				// Pop a function off the queue and run it.
				c.queueMux.Lock()
				c.current++
				f := c.queue[0]
				c.queue = c.queue[1:]
				c.wg.Add(1)
				go func() {
					defer c.wg.Done()
					defer func() {
						c.queueMux.Lock()
						// add some jitter before releasing the queue
						time.Sleep(time.Millisecond * 10)
						c.current--
						c.queueMux.Unlock()
					}()
					if err := f(); err != nil {
						c.cfg.LogVerbose(1, "queue: error running function: %v", err)
						c.errs <- err
					}
				}()
				c.queueMux.Unlock()
			} else {
				// No functions left to run, check if wait has been called.
				if !c.waiting.TryLock() {
					// Wait has been called, so send the cancel once all routines
					// have finished.
					c.wg.Wait()
					return
				}
				c.waiting.Unlock()
			}
		}
	}
}
