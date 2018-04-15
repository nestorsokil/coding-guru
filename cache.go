package main

import (
	"github.com/bluele/gcache"
	"time"
)


type QueryCache interface {
	Put(key, value string, expireSeconds int64) (error)
	Get(key string) (string, bool)
}

func NewCache(size int) QueryCache {
	return &lfuCacheImpl{ data:gcache.New(size).LFU().Build() }
}

type lfuCacheImpl struct {
	data gcache.Cache
}

func (cache *lfuCacheImpl) Put(key, value string, expireSeconds int64) (error) {
	err := cache.data.SetWithExpire(key, value, time.Duration(expireSeconds) * time.Second)
	if err != nil {
		return err
	}
	return nil
}

func (cache *lfuCacheImpl) Get(key string) (string, bool) {
	entry, err := cache.data.Get(key)
	if entry == nil || err != nil {
		return "", false
	}
	return entry.(string), true
}

