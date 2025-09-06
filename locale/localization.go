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
	HomePage   HomePageConfig   `yaml:"HomePage" json:"HomePage"`
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

type HomePageConfig struct {
	Header                 string `yaml:"Header" json:"Header"`
	DidYouKnowThat         string `yaml:"DidYouKnowThat" json:"DidYouKnowThat"`
	SidebarDescription     string `yaml:"SidebarDescription" json:"SidebarDescription"`
	HomePageDescription    string `yaml:"HomePageDescription" json:"HomePageDescription"`
	BlogSearchDescription  string `yaml:"BlogSearchDescription" json:"BlogSearchDescription"`
	MarkerMapDescription   string `yaml:"MarkerMapDescription" json:"MarkerMapDescription"`
	ThemeSwitchDescription string `yaml:"ThemeSwitchDescription" json:"ThemeSwitchDescription"`
	Hymn1                  string `yaml:"Hymn1" json:"Hymn1"`
	Hymn2                  string `yaml:"Hymn2" json:"Hymn2"`
	Hymn3                  string `yaml:"Hymn3" json:"Hymn3"`
	Hymn4                  string `yaml:"Hymn4" json:"Hymn4"`
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
