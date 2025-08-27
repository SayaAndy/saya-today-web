package locale

import (
	"os"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

type LocaleConfig struct {
	TagsLabel  string           `yaml:"TagsLabel" json:"TagsLabel"`
	BlogSearch BlogSearchConfig `yaml:"BlogSearch" json:"BlogSearch"`
	GlobalMap  GlobalMapConfig  `yaml:"GlobalMap" json:"GlobalMap"`
	LikeButton string           `yaml:"LikeButton" json:"LikeButton"`
}

type BlogSearchConfig struct {
	Header                 string `yaml:"Header" json:"Header"`
	TagsHeader             string `yaml:"TagsHeader" json:"TagsHeader"`
	OrderByHeader          string `yaml:"OrderByHeader" json:"OrderByHeader"`
	TitleOrdered           string `yaml:"TitleOrdered" json:"TitleOrdered"`
	ActionDateOrdered      string `yaml:"ActionDateOrdered" json:"ActionDateOrdered"`
	PublicationDateOrdered string `yaml:"PublicationDateOrdered" json:"PublicationDateOrdered"`
	ChooseAllTags          string `yaml:"ChooseAllTags" json:"ChooseAllTags"`
}

type GlobalMapConfig struct {
	Header string `yaml:"Header" json:"Header"`
}

func LoadConfig(path string, config *LocaleConfig) error {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err = yaml.Unmarshal(fileBytes, config); err != nil {
		return err
	}

	return nil
}

func InitConfig(path string) (*LocaleConfig, error) {
	config := &LocaleConfig{}
	if err := LoadConfig(path, config); err != nil {
		return nil, err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return nil, err
	}

	return config, nil
}
