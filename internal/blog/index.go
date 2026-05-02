package blog

import "time"

const IndexFileName = "index.json"

const IndexSchemaVersion = 1

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

type IndexCategory struct {
	GeneratedAt time.Time    `json:"generatedAt"`
	Pages       []IndexEntry `json:"pages"`
}

type Index struct {
	SchemaVersion int                      `json:"schemaVersion"`
	GeneratedAt   time.Time                `json:"generatedAt"`
	Categories    map[string]IndexCategory `json:"categories"`
}
