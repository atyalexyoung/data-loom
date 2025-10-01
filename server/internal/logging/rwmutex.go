package logging

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// DebugRWMutex wraps sync.RWMutex with logging
type DebugRWMutex struct {
	mu        sync.RWMutex
	component string
}

// NewDebugRWMutex creates a new instance with a component name for logs
func NewDebugRWMutex(component string) *DebugRWMutex {
	return &DebugRWMutex{component: component}
}

func (d *DebugRWMutex) RLock(method string) {
	start := time.Now()
	log.WithFields(log.Fields{
		"component": d.component,
		"method":    method,
		"lock_mode": "RLock",
	}).Trace("Acquiring read lock")

	d.mu.RLock()

	log.WithFields(log.Fields{
		"component": d.component,
		"method":    method,
		"lock_mode": "RLock",
		"wait_ms":   time.Since(start).Milliseconds(),
	}).Trace("Acquired read lock")
}

func (d *DebugRWMutex) RUnlock(method string) {
	d.mu.RUnlock()
	log.WithFields(log.Fields{
		"component": d.component,
		"method":    method,
		"lock_mode": "RUnlock",
	}).Trace("Released read lock")
}

func (d *DebugRWMutex) Lock(method string) {
	start := time.Now()
	log.WithFields(log.Fields{
		"component": d.component,
		"method":    method,
		"lock_mode": "Lock",
	}).Trace("Acquiring write lock")

	d.mu.Lock()

	log.WithFields(log.Fields{
		"component": d.component,
		"method":    method,
		"lock_mode": "Lock",
		"wait_ms":   time.Since(start).Milliseconds(),
	}).Trace("Acquired write lock")
}

func (d *DebugRWMutex) Unlock(method string) {
	d.mu.Unlock()
	log.WithFields(log.Fields{
		"component": d.component,
		"method":    method,
		"lock_mode": "Unlock",
	}).Trace("Released write lock")
}
