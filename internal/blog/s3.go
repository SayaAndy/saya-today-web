package blog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/internal/frontmatter"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const s3ScanConcurrency = 32

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

	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(s3cfg.Region),
	}
	if s3cfg.AccessKeyID != "" && s3cfg.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(s3cfg.AccessKeyID, s3cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	var s3Opts []func(*s3.Options)
	if s3cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(s3cfg.Endpoint)
		})
	}
	s3Opts = append(s3Opts, func(o *s3.Options) {
		o.UsePathStyle = s3cfg.UsePathStyle
		o.DisableLogOutputChecksumValidationSkipped = true
	})

	s3cl := s3.NewFromConfig(awsCfg, s3Opts...)

	return &S3Client{s3cfg.Prefix, s3cfg.BucketName, s3cl}, nil
}

func (c *S3Client) Scan(prefix string) ([]*Page, error) {
	pages, err := c.scanFromIndex(prefix)
	if err == nil {
		return pages, nil
	}

	var nsk *s3types.NoSuchKey
	if !errors.As(err, &nsk) {
		return nil, err
	}

	slog.Warn("index.json missing, falling back to listing", slog.String("prefix", c.prefix))
	return c.scanByListing(prefix)
}

func (c *S3Client) scanFromIndex(prefix string) ([]*Page, error) {
	out, err := c.s3cl.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(IndexFileName),
	})
	if err != nil {
		return nil, fmt.Errorf("get index.json: %w", err)
	}
	defer out.Body.Close()

	raw, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("read index.json: %w", err)
	}

	var idx Index
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, fmt.Errorf("unmarshal index.json: %w", err)
	}

	wantLang := ""
	if i := strings.Index(prefix, "/"); i > 0 {
		wantLang = prefix[:i]
	}

	fullPrefix := c.prefix + prefix
	pages := make([]*Page, 0)
	for catKey, cat := range idx.Categories {
		lang, ok := strings.CutPrefix(catKey, c.prefix)
		if !ok {
			continue
		}
		if wantLang != "" && wantLang != lang {
			continue
		}
		for _, e := range cat.Pages {
			if !strings.HasPrefix(e.Link, fullPrefix) {
				continue
			}
			linkParts := strings.Split(e.Link, "/")
			nameParts := strings.Split(linkParts[len(linkParts)-1], ".")
			fileName := strings.Join(nameParts[:len(nameParts)-1], ".")
			pages = append(pages, &Page{
				Link:         e.Link,
				FileName:     fileName,
				Lang:         lang,
				ModifiedTime: e.ModifiedTime,
				Metadata: &frontmatter.Metadata{
					Title:            e.Title,
					ShortDescription: e.ShortDescription,
					ActionDate:       e.ActionDate,
					PublishedTime:    e.PublishedTime,
					Thumbnail:        e.Thumbnail,
					Tags:             e.Tags,
					Geolocation:      e.Geolocation,
					Medley:           e.Medley,
					MedleyPart:       e.MedleyPart,
				},
			})
		}
	}
	return pages, nil
}

func (c *S3Client) scanByListing(prefix string) ([]*Page, error) {
	fullPrefix := c.prefix + prefix

	type candidate struct {
		key          string
		lastModified time.Time
	}
	var candidates []candidate

	paginator := s3.NewListObjectsV2Paginator(c.s3cl, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucketName),
		Prefix: aws.String(fullPrefix),
	})
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
			candidates = append(candidates, candidate{key, aws.ToTime(obj.LastModified)})
		}
	}

	pages := make([]*Page, len(candidates))
	sem := make(chan struct{}, s3ScanConcurrency)
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex

	for i, cand := range candidates {
		sem <- struct{}{}
		wg.Go(func() {
			defer func() { <-sem }()

			head, err := c.s3cl.HeadObject(context.Background(), &s3.HeadObjectInput{
				Bucket: aws.String(c.bucketName),
				Key:    aws.String(cand.key),
			})
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("head S3 object %s: %w", cand.key, err)
				}
				errMu.Unlock()
				return
			}

			if head.ContentType == nil || !strings.Contains(*head.ContentType, "text/markdown") {
				return
			}
			meta := head.Metadata
			if meta["title"] == "" {
				return
			}

			publishedTime, err := time.Parse(time.RFC3339, meta["published-time"])
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("failed to parse published time metadata field: %w", err)
				}
				errMu.Unlock()
				return
			}

			linkParts := strings.Split(cand.key, "/")
			nameParts := strings.Split(linkParts[len(linkParts)-1], ".")
			fileName := strings.Join(nameParts[:len(nameParts)-1], ".")
			lang, _ := strings.CutPrefix(linkParts[0], c.prefix)

			tags := strings.Split(meta["tags"], ",")
			slices.Sort(tags)

			title, _ := url.QueryUnescape(meta["title"])
			shortDescription, _ := url.QueryUnescape(meta["short-description"])
			thumbnail, _ := url.QueryUnescape(meta["thumbnail"])

			pages[i] = &Page{
				Link:         cand.key,
				FileName:     fileName,
				Lang:         lang,
				ModifiedTime: cand.lastModified,
				Metadata: &frontmatter.Metadata{
					Title:            title,
					ShortDescription: shortDescription,
					ActionDate:       meta["action-date"],
					PublishedTime:    publishedTime,
					Thumbnail:        thumbnail,
					Tags:             tags,
					Geolocation:      meta["geolocation"],
				},
			}
		})
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	out := pages[:0]
	for _, p := range pages {
		if p != nil {
			out = append(out, p)
		}
	}
	return out, nil
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
