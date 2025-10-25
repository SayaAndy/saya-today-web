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

type ClientCache struct {
	hashMap      map[string]string
	hashMapMutex sync.RWMutex

	likePageMap       map[string]map[string]struct{}
	viewPageMap       map[string]map[string]struct{}
	pageMutexMap      map[string]*sync.RWMutex
	pageMutexMapMutex sync.Mutex

	salt []byte
	db   *sql.DB
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
	viewPageMap := make(map[string]map[string]struct{})
	pageMutexMap := make(map[string]*sync.RWMutex)

	for rows.Next() {
		var pageRef string
		var userId []byte
		if err = rows.Scan(&pageRef, &userId); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("fail scanning blog_likes to fill cache: %w", err)
		}
		userIdString := base64.RawStdEncoding.EncodeToString(userId)
		if _, ok := likePageMap[pageRef]; !ok {
			likePageMap[pageRef] = make(map[string]struct{})
			viewPageMap[pageRef] = make(map[string]struct{})
			pageMutexMap[pageRef] = &sync.RWMutex{}
		}
		likePageMap[pageRef][userIdString] = struct{}{}
		viewPageMap[pageRef][userIdString] = struct{}{}
	}

	rows, err = tx.Query("select * from blog_views;")
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("fail to query db for blog_views to fill cache: %w", err)
	}

	for rows.Next() {
		var pageRef string
		var userId []byte
		if err = rows.Scan(&pageRef, &userId); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("fail scanning blog_views to fill cache: %w", err)
		}
		userIdString := base64.RawStdEncoding.EncodeToString(userId)
		if _, ok := viewPageMap[pageRef]; !ok {
			viewPageMap[pageRef] = make(map[string]struct{})
			pageMutexMap[pageRef] = &sync.RWMutex{}
		}
		viewPageMap[pageRef][userIdString] = struct{}{}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("fail to commit transaction in db: %w", err)
	}

	return &ClientCache{
		hashMap:      make(map[string]string),
		likePageMap:  likePageMap,
		viewPageMap:  viewPageMap,
		pageMutexMap: pageMutexMap,
		salt:         salt,
		db:           db,
	}, nil
}

func (c *ClientCache) Close() error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("fail to init transaction with db to dump cache: %w", err)
	}

	if err = batchSave(tx, "blog_likes", c.likePageMap); err != nil {
		tx.Rollback()
		return fmt.Errorf("fail to save blog_likes: %s", err)
	}

	if err = batchSave(tx, "blog_views", c.viewPageMap); err != nil {
		tx.Rollback()
		return fmt.Errorf("fail to save blog_views: %s", err)
	}

	if err = tx.Commit(); err != nil {
		tx.Rollback()
		return fmt.Errorf("fail to commit all the changes related to cache: %s", err)
	}
	return nil
}

func (c *ClientCache) GetHash(id string) string {
	c.hashMapMutex.RLock()
	if val, ok := c.hashMap[id]; ok {
		c.hashMapMutex.RUnlock()
		slog.Debug("gave an old hash", slog.String("hash", val))
		return val
	}
	c.hashMapMutex.RUnlock()

	c.hashMapMutex.Lock()
	defer c.hashMapMutex.Unlock()

	if val, ok := c.hashMap[id]; ok {
		slog.Debug("gave a newly generated hash", slog.String("hash", val))
		return val
	}

	c.hashMap[id] = base64.RawStdEncoding.EncodeToString(argon2.IDKey([]byte(id), c.salt, 1, 64*1024, 4, 32))
	slog.Debug("generated hash", slog.String("hash", c.hashMap[id]))
	return c.hashMap[id]
}

func (c *ClientCache) getPageMutex(page string) *sync.RWMutex {
	c.pageMutexMapMutex.Lock()
	defer c.pageMutexMapMutex.Unlock()

	if mutex, ok := c.pageMutexMap[page]; ok {
		return mutex
	}

	c.pageMutexMap[page] = &sync.RWMutex{}
	return c.pageMutexMap[page]
}

func (c *ClientCache) GetLikeStatus(id string, page string) bool {
	page = strings.Clone(page)

	mutex := c.getPageMutex(page)
	mutex.RLock()
	defer mutex.RUnlock()

	if _, ok := c.likePageMap[page]; !ok {
		return false
	}
	_, ok := c.likePageMap[page][c.GetHash(id)]
	return ok
}

func (c *ClientCache) GetLikeCount(page string) int {
	page = strings.Clone(page)

	mutex := c.getPageMutex(page)
	mutex.RLock()
	defer mutex.RUnlock()

	if userSet, ok := c.likePageMap[page]; ok {
		return len(userSet)
	}
	return 0
}

func (c *ClientCache) LikeOn(id string, page string) (alreadyLiked bool) {
	page = strings.Clone(page)
	hash := c.GetHash(id)

	mutex := c.getPageMutex(page)
	mutex.Lock()
	defer mutex.Unlock()

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
	page = strings.Clone(page)
	hash := c.GetHash(id)

	mutex := c.getPageMutex(page)
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := c.likePageMap[page]; !ok {
		return true
	}
	if _, ok := c.likePageMap[page][hash]; !ok {
		return true
	}

	delete(c.likePageMap[page], hash)
	return false
}

func (c *ClientCache) GetViewStatus(id string, page string) bool {
	page = strings.Clone(page)

	mutex := c.getPageMutex(page)
	mutex.RLock()
	defer mutex.RUnlock()

	if _, ok := c.viewPageMap[page]; !ok {
		return false
	}
	_, ok := c.viewPageMap[page][c.GetHash(id)]
	return ok
}

func (c *ClientCache) GetViewCount(page string) int {
	page = strings.Clone(page)

	mutex := c.getPageMutex(page)
	mutex.RLock()
	defer mutex.RUnlock()

	if userSet, ok := c.viewPageMap[page]; ok {
		return len(userSet)
	}
	return 0
}

func (c *ClientCache) View(id string, page string) {
	page = strings.Clone(page)
	hash := c.GetHash(id)

	mutex := c.getPageMutex(page)
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := c.viewPageMap[page]; !ok {
		c.viewPageMap[page] = make(map[string]struct{})
	}
	c.viewPageMap[page][hash] = struct{}{}
}

func batchSave(tx *sql.Tx, table string, pageMap map[string]map[string]struct{}) (err error) {
	if _, err = tx.Exec(fmt.Sprintf("delete from %s;", table)); err != nil {
		return fmt.Errorf("fail to truncate table %s: %w", table, err)
	}

	userIdBytes := make(map[string][]byte)

	sqlStatement := fmt.Sprintf(`
    INSERT OR IGNORE INTO %s (page_ref, user_id)
    VALUES %s(?, ?);
    `, table, strings.Repeat("(?, ?), ", 99))
	sqlStatementVars := make([]any, 0, 200)

	for pageRef, userSet := range pageMap {
		for userId := range userSet {
			if _, ok := userIdBytes[userId]; !ok {
				userIdBytes[userId], err = base64.RawStdEncoding.DecodeString(userId)
				if err != nil {
					slog.Warn("couldn't parse one of user hashes into bytes back", slog.String("hash", userId), slog.String("error", err.Error()))
					continue
				}
			}

			sqlStatementVars = append(sqlStatementVars, any(pageRef), any(userIdBytes[userId]))
			if len(sqlStatementVars) < 200 {
				continue
			}

			if _, err := tx.Exec(sqlStatement, sqlStatementVars...); err != nil {
				slog.Warn("couldn't insert blog stat pairs into db", slog.String("table", table), slog.String("error", err.Error()))
			}

			sqlStatementVars = make([]any, 0, 200)
		}
	}

	if len(sqlStatementVars) > 0 {
		sqlStatement = fmt.Sprintf(`
		INSERT OR IGNORE INTO %s (page_ref, user_id)
		VALUES %s(?, ?);
		`, table, strings.Repeat("(?, ?), ", len(sqlStatementVars)/2-1))

		if _, err := tx.Exec(sqlStatement, sqlStatementVars...); err != nil {
			slog.Warn("couldn't insert blog stat pairs into db", slog.String("table", table), slog.String("error", err.Error()))
		}
	}

	return nil
}
