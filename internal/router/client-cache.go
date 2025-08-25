package router

import (
	"log/slog"
	"sync"

	"golang.org/x/crypto/argon2"
)

type ClientCache struct {
	hashMap  map[string][]byte
	mutexMap map[string]*sync.Mutex
	salt     []byte
}

var CCache *ClientCache

func NewClientCache(salt []byte) *ClientCache {
	return &ClientCache{
		hashMap:  make(map[string][]byte),
		mutexMap: make(map[string]*sync.Mutex),
		salt:     salt,
	}
}

func (c *ClientCache) GetHash(id string) []byte {
	if val, ok := c.hashMap[id]; ok {
		slog.Debug("gave an old hash", slog.String("hash", string(val)))
		return val
	}

	if _, ok := c.mutexMap[id]; !ok {
		c.mutexMap[id] = &sync.Mutex{}
	}
	c.mutexMap[id].Lock()
	defer c.mutexMap[id].Unlock()

	if val, ok := c.hashMap[id]; ok {
		slog.Debug("gave a newly generated hash", slog.String("hash", string(val)))
		return val
	}

	c.hashMap[id] = argon2.IDKey([]byte(id), c.salt, 1, 64*1024, 4, 32)
	slog.Debug("generated hash", slog.String("hash", string(c.hashMap[id])))
	return c.hashMap[id]
}
