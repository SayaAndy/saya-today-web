package blog

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/SayaAndy/saya-today-web/internal/frontmatter"
)

const IndexFileName = "index.json"
const MedleysIndexFileName = "medleys.json"

const IndexSchemaVersion = 2

type IndexEntry struct {
	Link             string    `json:"link"`
	ModifiedTime     time.Time `json:"modifiedTime"`
	Title            string    `json:"title"`
	ShortDescription string    `json:"shortDescription"`
	ActionDate       string    `json:"actionDate"`
	PublishedTime    time.Time `json:"publishedTime"`
	Thumbnail        string    `json:"thumbnail"`
	Tags             []string  `json:"tags"`
	Geolocation      string    `json:"geolocation"`
	Medley           string    `json:"medley,omitempty"`
	MedleyPart       int       `json:"medleyPart,omitempty"`
}

func (e IndexEntry) Metadata() *frontmatter.Metadata {
	return &frontmatter.Metadata{
		Title:            e.Title,
		ShortDescription: e.ShortDescription,
		ActionDate:       e.ActionDate,
		PublishedTime:    e.PublishedTime,
		Thumbnail:        e.Thumbnail,
		Tags:             e.Tags,
		Geolocation:      e.Geolocation,
		Medley:           e.Medley,
		MedleyPart:       e.MedleyPart,
	}
}

type IndexV2Category struct {
	GeneratedAt time.Time             `json:"generatedAt"`
	Pages       map[string]IndexEntry `json:"pages"`
}

type IndexV1Category struct {
	GeneratedAt time.Time    `json:"generatedAt"`
	Pages       []IndexEntry `json:"pages"`
}

type Index struct {
	SchemaVersion int       `json:"schemaVersion"`
	GeneratedAt   time.Time `json:"generatedAt"`
	Categories    any       `json:"categories"`
}

func (idx *Index) UnmarshalJSON(data []byte) error {
	var tmp struct {
		SchemaVersion int             `json:"schemaVersion"`
		GeneratedAt   time.Time       `json:"generatedAt"`
		Categories    json.RawMessage `json:"categories"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	idx.SchemaVersion = tmp.SchemaVersion
	idx.GeneratedAt = tmp.GeneratedAt

	switch tmp.SchemaVersion {
	case 1:
		var categories map[string]*IndexV1Category
		if err := json.Unmarshal(tmp.Categories, &categories); err != nil {
			return fmt.Errorf("unmarshal map[string]*IndexV1Category: %w", err)
		}
		idx.Categories = &categories
	case 2:
		var categories map[string]*IndexV2Category
		if err := json.Unmarshal(tmp.Categories, &categories); err != nil {
			return fmt.Errorf("unmarshal map[string]*IndexV2Category: %w", err)
		}
		idx.Categories = &categories
	default:
		return fmt.Errorf("unsupported index version: %d", tmp.SchemaVersion)
	}

	return nil
}

type MedleyEntry struct {
	Codename string   `json:"codename"`
	Content  []string `json:"content"`
}

type MedleyPageEntry struct {
	Codename string `json:"codename"`
	Position int    `json:"position"`
}
