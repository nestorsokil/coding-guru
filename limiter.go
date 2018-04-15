package main

import (
	"sync"
	"fmt"
	"time"
)

type UnLimiter func()

type Limiter interface {
	Allow(operation, user string, withTimeoutMillis int64) (bool, UnLimiter)
}

func NewLimiter() Limiter {
	t := limiterImpl{}
	t.inProgress = syncSet{
		RWMutex:sync.RWMutex{},
		elements:make(map[string]struct{}),
	}
	return &t
}

type limiterImpl struct {
	inProgress syncSet
}

func (l *limiterImpl) Allow(operation, user string, withTimeoutMillis int64) (bool, UnLimiter) {
	key := fmt.Sprintf("%s.%s", operation, user)
	if l.inProgress.has(key) {
		return false, nil
	}
	l.inProgress.put(key)
	cancelled := make(chan struct{})
	timeout := time.After(time.Duration(withTimeoutMillis) * time.Millisecond)
	cancelFunc := func() {
		cancelled <- struct{}{}
	}
	go func() {
		select {
		case <-cancelled: l.inProgress.rm(key)
		case <-timeout: l.inProgress.rm(key)
		}
	}()
	return true, cancelFunc
}

type syncSet struct {
	sync.RWMutex
	elements map[string]struct{}
}
func (s syncSet) put(elem string) {
	s.Lock()
	s.elements[elem] = struct{}{}
	s.Unlock()
}
func (s syncSet) has(elem string) bool {
	s.RLock()
	_, ok := s.elements[elem]
	s.RUnlock()
	return ok
}
func (s syncSet) rm(elem string) {
	s.Lock()
	delete(s.elements, elem)
	s.Unlock()
}
