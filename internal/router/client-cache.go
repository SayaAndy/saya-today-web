package router

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

type PageLike struct {
	PageRef string
	UserId  string
}

type ClientCache struct {
	hashMap      map[string]string
	mutexLikeMap map[string]*sync.Mutex
	mutexHashMap map[string]*sync.Mutex
	likePageMap  map[string]map[string]struct{}
	salt         []byte
	db           *sql.DB
}

var CCache *ClientCache

func NewClientCache(db *sql.DB, salt []byte) (*ClientCache, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("fail to init transaction with db to fill cache: %w", err)
	}

	rows, err := tx.Query("select * from blog_likes;")
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("fail to query db for blog_likes to fill cache: %w", err)
	}

	likePageMap := make(map[string]map[string]struct{})

	for rows.Next() {
		pageRef := make([]byte, 32)
		userId := make([]byte, 32)
		if err = rows.Scan(&pageRef, &userId); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("fail scanning blog_likes to fill cache: %w", err)
		}
		pageRefString := string(pageRef)
		userIdString := base64.RawStdEncoding.EncodeToString(userId)
		if _, ok := likePageMap[pageRefString]; !ok {
			likePageMap[pageRefString] = make(map[string]struct{})
		}
		likePageMap[pageRefString][userIdString] = struct{}{}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("fail to commit transaction in db: %w", err)
	}

	return &ClientCache{
		hashMap:      make(map[string]string),
		mutexLikeMap: make(map[string]*sync.Mutex),
		mutexHashMap: make(map[string]*sync.Mutex),
		likePageMap:  likePageMap,
		salt:         salt,
		db:           db,
	}, nil
}

func (c *ClientCache) Close() error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("fail to init transaction with db to dump cache: %w", err)
	}

	if _, err = tx.Exec("delete from blog_likes;"); err != nil {
		tx.Rollback()
		return fmt.Errorf("fail to truncate table blog_likes: %w", err)
	}

	userIdBytes := make(map[string][]byte)

	sqlStatement := fmt.Sprintf(`
    INSERT OR IGNORE INTO blog_likes (page_ref, user_id)
    VALUES %s(?, ?);
    `, strings.Repeat("(?, ?), ", 99))
	sqlStatementVars := make([]interface{}, 0, 200)

	for pageRef, userSet := range c.likePageMap {
		pageRefBytes := []byte(pageRef)

		for userId := range userSet {
			if _, ok := userIdBytes[userId]; !ok {
				userIdBytes[userId], err = base64.RawStdEncoding.DecodeString(userId)
				if err != nil {
					slog.Warn("couldn't parse one of user hashes into bytes back", slog.String("hash", userId), slog.String("error", err.Error()))
					continue
				}
			}

			sqlStatementVars = append(sqlStatementVars, interface{}(pageRefBytes), interface{}(userIdBytes[userId]))
			if len(sqlStatementVars) < 200 {
				continue
			}

			if _, err := tx.Exec(sqlStatement, sqlStatementVars...); err != nil {
				slog.Warn("couldn't insert blog like pairs into db", slog.String("error", err.Error()))
			}

			sqlStatementVars = make([]interface{}, 0, 200)
		}
	}

	if len(sqlStatementVars) > 0 {
		sqlStatement = fmt.Sprintf(`
		INSERT OR IGNORE INTO blog_likes (page_ref, user_id)
		VALUES %s(?, ?);
		`, strings.Repeat("(?, ?), ", len(sqlStatementVars)/2-1))

		if _, err := tx.Exec(sqlStatement, sqlStatementVars...); err != nil {
			slog.Warn("couldn't insert blog like pairs into db", slog.String("error", err.Error()))
		}
	}

	return tx.Commit()
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

func (c *ClientCache) GetLikeCount(page string) int {
	if userSet, ok := c.likePageMap[page]; ok {
		return len(userSet)
	}
	return 0
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
