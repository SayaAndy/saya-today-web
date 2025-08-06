package b2

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Backblaze/blazer/b2"
	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/internal/frontmatter"
)

type B2Client struct {
	prefix string
	bucket *b2.Bucket
	b2cl   *b2.Client
}

func NewB2Client(cfg *config.B2Config) (*B2Client, error) {
	b2cl, err := b2.NewClient(context.Background(), cfg.KeyID, cfg.ApplicationKey)
	if err != nil {
		return nil, err
	}

	bucket, err := b2cl.Bucket(context.Background(), cfg.BucketName)
	if err != nil {
		return nil, err
	}

	return &B2Client{b2cl: b2cl, bucket: bucket, prefix: cfg.Prefix}, nil
}

type BlogPage struct {
	Link     string
	Metadata *frontmatter.Metadata
}

func (c *B2Client) Scan(prefix string) ([]*BlogPage, error) {
	filePaths := []*BlogPage{}

	iter := c.bucket.List(context.Background(), b2.ListPrefix(c.prefix+prefix))

	for iter.Next() {
		obj := iter.Object()
		if obj == nil {
			return nil, fmt.Errorf("failed to reference object in B2 bucket")
		}

		attrs, err := obj.Attrs(context.Background())
		if err != nil {
			return nil, fmt.Errorf("get attributes for object: %w", err)
		}

		if attrs.Status != b2.Uploaded {
			continue
		}

		if !strings.Contains(attrs.ContentType, "text/markdown") {
			continue
		}

		if _, ok := attrs.Info["title"]; !ok {
			continue
		}

		publishedTime, err := time.Parse(time.RFC3339, attrs.Info["published-time"])
		if err != nil {
			return nil, fmt.Errorf("failed to parse published time metadata field: %w", err)
		}

		linkParts := strings.Split(obj.Name(), ".")

		filePaths = append(filePaths, &BlogPage{
			Link: strings.Join(linkParts[:len(linkParts)-1], "."),
			Metadata: &frontmatter.Metadata{
				Title:            attrs.Info["title"],
				ShortDescription: attrs.Info["short-description"],
				ActionDate:       attrs.Info["action-date"],
				PublishedTime:    publishedTime,
				Thumbnail:        attrs.Info["thumbnail"],
				Tags:             strings.Split(attrs.Info["tags"], ","),
			},
		})
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("iterate over B2 objects: %w", err)
	}

	return filePaths, nil
}

func (c *B2Client) ReadAll(path string) ([]byte, error) {
	obj := c.bucket.Object(c.prefix + path)
	if obj == nil {
		return nil, fmt.Errorf("failed to reference object in B2 bucket")
	}
	attrs, err := obj.Attrs(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error getting attributes of an object: %w", err)
	}

	content := make([]byte, attrs.Size)
	reader := obj.NewReader(context.Background())

	if _, err = reader.Read(content); err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return content, nil
}

func (c *B2Client) ReadFrontmatter(path string) (metadata *frontmatter.Metadata, markdown []byte, err error) {
	contentBytes, err := c.ReadAll(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file for frontmatter parsing: %w", err)
	}

	return frontmatter.ParseFrontmatter(contentBytes)
}
