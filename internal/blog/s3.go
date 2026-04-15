package blog

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/internal/frontmatter"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	prefix     string
	bucketName string
	s3cl       *s3.Client
}

func NewS3Client(cfg *config.StorageConfig) (Client, error) {
	if cfg.Type != "s3" {
		return nil, fmt.Errorf("invalid storage type for S3Client")
	}
	s3cfg := cfg.Config.(*config.S3Config)

	s3cl := s3.New(s3.Options{
		Region:       s3cfg.Region,
		BaseEndpoint: aws.String(s3cfg.Endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(s3cfg.AccessKeyID, s3cfg.SecretAccessKey, ""),
	})

	return &S3Client{s3cfg.Prefix, s3cfg.BucketName, s3cl}, nil
}

func (c *S3Client) Scan(prefix string) ([]*Page, error) {
	pages := []*Page{}

	fullPrefix := c.prefix + prefix
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucketName),
		Prefix: aws.String(fullPrefix),
	}

	paginator := s3.NewListObjectsV2Paginator(c.s3cl, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.Background())
		if err != nil {
			return nil, fmt.Errorf("list S3 objects: %w", err)
		}

		for _, obj := range output.Contents {
			key := aws.ToString(obj.Key)

			if !strings.HasSuffix(key, ".md") {
				continue
			}

			head, err := c.s3cl.HeadObject(context.Background(), &s3.HeadObjectInput{
				Bucket: aws.String(c.bucketName),
				Key:    aws.String(key),
			})
			if err != nil {
				return nil, fmt.Errorf("head S3 object %s: %w", key, err)
			}

			if head.ContentType == nil || !strings.Contains(*head.ContentType, "text/markdown") {
				continue
			}

			meta := head.Metadata
			if meta["title"] == "" {
				continue
			}

			publishedTime, err := time.Parse(time.RFC3339, meta["published-time"])
			if err != nil {
				return nil, fmt.Errorf("failed to parse published time metadata field: %w", err)
			}

			linkParts := strings.Split(key, "/")
			nameParts := strings.Split(linkParts[len(linkParts)-1], ".")
			fileName := strings.Join(nameParts[:len(nameParts)-1], ".")

			tags := strings.Split(meta["tags"], ",")
			slices.Sort(tags)

			lang, _ := strings.CutPrefix(linkParts[0], c.prefix)

			pages = append(pages, &Page{
				Link:         key,
				FileName:     fileName,
				Lang:         lang,
				ModifiedTime: aws.ToTime(obj.LastModified),
				Metadata: &frontmatter.Metadata{
					Title:            meta["title"],
					ShortDescription: meta["short-description"],
					ActionDate:       meta["action-date"],
					PublishedTime:    publishedTime,
					Thumbnail:        meta["thumbnail"],
					Tags:             tags,
					Geolocation:      meta["geolocation"],
				},
			})
		}
	}

	return pages, nil
}

func (c *S3Client) ReadAll(path string) ([]byte, error) {
	output, err := c.s3cl.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(c.prefix + path),
	})
	if err != nil {
		return nil, fmt.Errorf("get S3 object: %w", err)
	}
	defer output.Body.Close()

	content, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("read S3 object body: %w", err)
	}

	return content, nil
}

func (c *S3Client) ReadFrontmatter(path string) (metadata *frontmatter.Metadata, markdown []byte, err error) {
	contentBytes, err := c.ReadAll(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file for frontmatter parsing: %w", err)
	}

	return frontmatter.ParseFrontmatter(contentBytes)
}
