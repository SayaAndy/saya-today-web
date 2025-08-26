package router

import (
	"encoding/base64"
	"log/slog"
	"sync"

	"golang.org/x/crypto/argon2"
)

type ClientCache struct {
	hashMap      map[string]string
	mutexLikeMap map[string]*sync.Mutex
	mutexHashMap map[string]*sync.Mutex
	likePageMap  map[string]map[string]struct{}
	salt         []byte
}

var CCache *ClientCache

func NewClientCache(salt []byte) *ClientCache {
	return &ClientCache{
		hashMap:      make(map[string]string),
		mutexLikeMap: make(map[string]*sync.Mutex),
		mutexHashMap: make(map[string]*sync.Mutex),
		likePageMap:  make(map[string]map[string]struct{}),
		salt:         salt,
	}
}

func (c *ClientCache) GetHash(id string) string {
	if val, ok := c.hashMap[id]; ok {
		slog.Debug("gave an old hash", slog.String("hash", val))
		return val
	}

	if _, ok := c.mutexHashMap[id]; !ok {
		c.mutexHashMap[id] = &sync.Mutex{}
	}
	c.mutexHashMap[id].Lock()
	defer c.mutexHashMap[id].Unlock()

	if val, ok := c.hashMap[id]; ok {
		slog.Debug("gave a newly generated hash", slog.String("hash", val))
		return val
	}

	c.hashMap[id] = base64.RawStdEncoding.EncodeToString(argon2.IDKey([]byte(id), c.salt, 1, 64*1024, 4, 32))
	slog.Debug("generated hash", slog.String("hash", c.hashMap[id]))
	return c.hashMap[id]
}

func (c *ClientCache) GetLikeStatus(id string, page string) bool {
	if _, ok := c.likePageMap[page]; !ok {
		return false
	}
	_, ok := c.likePageMap[page][c.GetHash(id)]
	return ok
}

func (c *ClientCache) LikeOn(id string, page string) (alreadyLiked bool) {
	if _, ok := c.mutexLikeMap[id]; !ok {
		c.mutexLikeMap[id] = &sync.Mutex{}
	}
	c.mutexLikeMap[id].Lock()
	defer c.mutexLikeMap[id].Unlock()

	hash := c.GetHash(id)

	if userSet, ok := c.likePageMap[page]; ok {
		_, alreadyLiked = userSet[hash]
		c.likePageMap[page][hash] = struct{}{}
		return
	}

	c.likePageMap[page] = make(map[string]struct{})
	c.likePageMap[page][hash] = struct{}{}
	return
}

func (c *ClientCache) LikeOff(id string, page string) (alreadyUnliked bool) {
	if _, ok := c.mutexLikeMap[id]; !ok {
		c.mutexLikeMap[id] = &sync.Mutex{}
	}
	c.mutexLikeMap[id].Lock()
	defer c.mutexLikeMap[id].Unlock()

	if _, ok := c.likePageMap[page]; !ok {
		return true
	}
	hash := c.GetHash(id)
	if _, ok := c.likePageMap[page][hash]; !ok {
		return true
	}

	delete(c.likePageMap[page], hash)
	return false
}
