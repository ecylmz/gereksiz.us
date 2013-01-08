package cache

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"net/http"
)

type Cache struct {
	Cached []byte
}

/*
 * This function will save the data in the memcache and the datastore
 */
func AddCache(r *http.Request, identifier string, data []byte) {
	c := appengine.NewContext(r)

	// Add it to the memcache
	item := &memcache.Item{
		Key:   identifier,
		Value: data,
	}
	memcache.Add(c, item)

	// Add it to the Datastore
	cache := Cache{
		Cached: data,
	}
	datastore.Put(c, datastore.NewKey(c, "Cache", identifier, 0, nil), &cache)
}

/*
 * This function will get a specified entity from the memcache, and if not available from the datastore
 */
func GetCache(r *http.Request, identifier string) ([]byte, bool) {
	c := appengine.NewContext(r)

	// Check if the item is cached in the memcache
	if item, err := memcache.Get(c, identifier); err == nil {
		return item.Value, true
	}

	// Check if the item is cached in the datastore
	key := datastore.NewKey(c, "Cache", identifier, 0, nil)
	var cache Cache
	datastore.Get(c, key, &cache)
	if string(cache.Cached) != "" {
		// Add it to the memcache
		AddCache(r, identifier, cache.Cached)

		// Return the cached data
		return cache.Cached, true
	}

	// Not in cache :(
	return nil, false
}

/*
 * This function will clear a specified entity from the memcache and the datastore
 */
func DeleteCache(r *http.Request, identifier string) {
	c := appengine.NewContext(r)

	// Delete from memcache
	memcache.Delete(c, identifier)

	// Delete from the datastore
	key := datastore.NewKey(c, "Cache", identifier, 0, nil)
	var cache Cache
	datastore.Get(c, key, &cache)
	datastore.Delete(c, key)
}
