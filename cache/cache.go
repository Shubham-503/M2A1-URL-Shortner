package cache

import (
	"M2A1-URL-Shortner/models"
	"encoding/json"
	"time"

	"github.com/allegro/bigcache"
)

// URLShortener represents the data we want to cache.
// type ShortURL struct {
// 	OriginalURL string `json:"original_url"`
// 	// You can add more fields if needed.
// }

// URLCache defines the interface for our cache.
type URLCache interface {
	Set(key string, value models.URLShortener) error
	Get(key string) (models.URLShortener, error)
	Delete(key string) error
	Close() error
}

// BigCacheStore is an implementation of URLCache using BigCache.
type BigCacheStore struct {
	cache *bigcache.BigCache
}

// NewBigCacheStore initializes a new BigCacheStore.
func NewBigCacheStore() (*BigCacheStore, error) {
	config := bigcache.Config{
		Shards:           1024,
		LifeWindow:       10 * time.Minute,
		CleanWindow:      5 * time.Minute,
		MaxEntrySize:     500,
		HardMaxCacheSize: 8192,
		Verbose:          false,
	}
	bc, err := bigcache.NewBigCache(config)
	if err != nil {
		return nil, err
	}
	return &BigCacheStore{
		cache: bc,
	}, nil
}

// Set stores a value in the cache.
func (b *BigCacheStore) Set(key string, value models.URLShortener) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return b.cache.Set(key, data)
}

// Get retrieves a value from the cache.
func (b *BigCacheStore) Get(key string) (models.URLShortener, error) {
	data, err := b.cache.Get(key)
	if err != nil {
		return models.URLShortener{}, err
	}
	var value models.URLShortener
	err = json.Unmarshal(data, &value)
	if err != nil {
		return models.URLShortener{}, err
	}
	return value, nil
}

// Delete removes a value from the cache.
func (b *BigCacheStore) Delete(key string) error {
	return b.cache.Delete(key)
}

// Close stops the cache (BigCache doesn't need explicit closing, so we return nil).
func (b *BigCacheStore) Close() error {
	return nil
}
