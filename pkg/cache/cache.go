package cache

import (
	"time"
)

// This can't generate mock's for interface with generic's
// You can edit this interface, (for this service replace K=uuid.UUID, V = *entity.Order)
//go:generate mockgen -source=cache.go -destination=mock/cache.go -package=mock_cache

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
