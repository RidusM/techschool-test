package cache

import (
	"time"
)

//go:generate mockgen -source=cache.go -destination=mock/cache.go -package=mock_cache -typed

type Cache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Put(key K, value V, ttl time.Duration)
	Has(key K) bool
	Len() int
	Capacity() int
	Purge()
	StartCleanup(interval time.Duration)
	StopCleanup()
	SetOnEvicted(onEvicted func(key K, value V))
}
