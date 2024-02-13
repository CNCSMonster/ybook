package pkg

import lru "github.com/hashicorp/golang-lru"

// === cache ===

type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Remove(key string)
	RemoveAll()
}
type cache struct {
	arc *lru.ARCCache
}

func (c cache) Get(key string) (interface{}, bool) {
	return c.arc.Get(key)
}
func (c cache) Set(key string, value interface{}) {
	c.arc.Add(key, value)
}
func (c cache) Remove(key string) {
	c.arc.Remove(key)
}
func (c cache) RemoveAll() {
	c.arc.Purge()
}

func NewCache(size int) Cache {
	arc, err := lru.NewARC(size)
	if err != nil {
		panic(err)
	}
	return cache{arc: arc}
}
