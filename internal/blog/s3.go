package blog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/internal/frontmatter"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	out, err := c.s3cl.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(IndexFileName),
	})
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", IndexFileName, err)
	}
	defer out.Body.Close()

	raw, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", IndexFileName, err)
	}

	var idx Index
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", IndexFileName, err)
	}

	wantLang := ""
	if i := strings.Index(prefix, "/"); i > 0 {
		wantLang = prefix[:i]
	}

	fullPrefix := c.prefix + prefix
	pages := make([]*Page, 0)

	switch idx.SchemaVersion {
	case 1:
		for catKey, cat := range *idx.Categories.(*map[string]*IndexV1Category) {
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
				fileName := e.Link[strings.LastIndex(e.Link, "/")+1 : strings.LastIndex(e.Link, ".")]
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
	case 2:
		for catKey, cat := range *idx.Categories.(*map[string]*IndexV2Category) {
			lang, ok := strings.CutPrefix(catKey, c.prefix)
			if !ok {
				continue
			}
			if wantLang != "" && wantLang != lang {
				continue
			}
			for codename, e := range cat.Pages {
				if !strings.HasPrefix(e.Link, fullPrefix) {
					continue
				}
				pages = append(pages, &Page{
					Link:         e.Link,
					FileName:     codename,
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
	}

	return pages, nil
}

func (c *S3Client) ReadAll(path string) ([]byte, error) {
	return c.readAll(c.prefix + path)
}

func (c *S3Client) readAll(path string) ([]byte, error) {
	output, err := c.s3cl.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(path),
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
	idxRaw, err := c.readAll(IndexFileName)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", IndexFileName, err)
	}

	var idx Index
	if err := json.Unmarshal(idxRaw, &idx); err != nil {
		return nil, nil, fmt.Errorf("unmarshal %s: %w", IndexFileName, err)
	}

	contentBytes, err := c.ReadAll(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file for frontmatter parsing: %w", err)
	}

	switch idx.SchemaVersion {
	case 1:
		return frontmatter.ParseFrontmatter(contentBytes)
	case 2:
		fullPath := c.prefix + path
		page := (*idx.Categories.(*map[string]*IndexV2Category))[fullPath[:strings.LastIndex(fullPath, "/")]].Pages[fullPath[strings.LastIndex(fullPath, "/")+1:strings.LastIndex(fullPath, ".")]]
		metadata = page.Metadata()

		if !bytes.HasPrefix(contentBytes, []byte("---\n")) {
			return metadata, contentBytes, nil
		}

		end := bytes.Index(contentBytes[4:], []byte("\n---\n"))
		if end == -1 {
			return metadata, contentBytes, nil
		}

		return metadata, contentBytes[end+9:], nil
	}

	return frontmatter.ParseFrontmatter(contentBytes)
}
