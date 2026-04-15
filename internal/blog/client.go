package blog

import (
	"time"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/internal/frontmatter"
)

type Page struct {
	Link         string
	FileName     string
	Lang         string
	ModifiedTime time.Time
	Metadata     *frontmatter.Metadata
}

type Client interface {
	Scan(prefix string) ([]*Page, error)
	ReadAll(path string) ([]byte, error)
	ReadFrontmatter(path string) (metadata *frontmatter.Metadata, markdown []byte, err error)
}

var NewClientMap = map[string]func(*config.StorageConfig) (Client, error){
	"b2": NewB2Client,
	"s3": NewS3Client,
}
