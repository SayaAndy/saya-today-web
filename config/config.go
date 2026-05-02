package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel           slog.Level                `json:"LogLevel" yaml:"logLevel" validate:"required"`
	BlogPages          BlogPagesConfig           `json:"BlogPages" yaml:"blogPages" validate:"required"`
	FactGiver          FactGiverConfig           `json:"FactGiver" yaml:"factGiver" validate:"required"`
	LocalePath         string                    `json:"LocalePath" yaml:"localePath" validate:"required,filepath"`
	AvailableLanguages []AvailableLanguageConfig `json:"AvailableLanguages" yaml:"availableLanguages" validate:"required"`
	Auth               AuthConfig                `json:"Auth" yaml:"auth" validate:"required"`
	Mail               MailConfig                `json:"Mail" yaml:"mail" validate:"required"`
	CanonicalEndpoint  string                    `json:"CanonicalEndpoint" yaml:"canonicalEndpoint" validate:"required"`
	Meta               MetaConfig                `json:"Meta" yaml:"meta"`
	PhotoStorage       PhotoStorageConfig        `json:"PhotoStorage" yaml:"photoStorage"`
	StaticStorage      PhotoTypeConfig           `json:"StaticStorage" yaml:"staticStorage"`
	AllowOrigins       []string                  `json:"AllowOrigins" yaml:"allowOrigins"`
}

type BlogPagesConfig struct {
	Storage StorageConfig `json:"Storage" yaml:"storage" validate:"required"`
}

type StorageConfig struct {
	Type   string `json:"type" yaml:"type,oneof=b2 s3"`
	Config any    `json:"Config" yaml:"config" validate:"required"`
}

func (sc *StorageConfig) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Type   string          `json:"Type"`
		Config json.RawMessage `json:"Config"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	sc.Type = tmp.Type

	switch tmp.Type {
	case "b2":
		var b2Config B2Config
		if err := json.Unmarshal(tmp.Config, &b2Config); err != nil {
			return fmt.Errorf("unmarshal B2Config: %w", err)
		}
		sc.Config = &b2Config
	case "s3":
		var s3Config S3Config
		if err := json.Unmarshal(tmp.Config, &s3Config); err != nil {
			return fmt.Errorf("unmarshal S3Config: %w", err)
		}
		sc.Config = &s3Config
	default:
		return fmt.Errorf("unsupported storage type: %s", tmp.Type)
	}

	return nil
}

func (sc *StorageConfig) UnmarshalYAML(value *yaml.Node) error {
	var tmp struct {
		Type   string    `yaml:"type"`
		Config yaml.Node `yaml:"config"`
	}

	if err := value.Decode(&tmp); err != nil {
		return err
	}

	sc.Type = tmp.Type

	switch tmp.Type {
	case "b2":
		var b2Config B2Config
		if err := tmp.Config.Decode(&b2Config); err != nil {
			return fmt.Errorf("unmarshal B2Config: %w", err)
		}
		sc.Config = &b2Config
	case "s3":
		var s3Config S3Config
		if err := tmp.Config.Decode(&s3Config); err != nil {
			return fmt.Errorf("unmarshal S3Config: %w", err)
		}
		sc.Config = &s3Config
	default:
		return fmt.Errorf("unsupported storage type: %s", tmp.Type)
	}

	return nil
}

type B2Config struct {
	BucketName     string `json:"BucketName" yaml:"bucketName" validate:"required,min=1"`
	Region         string `json:"Region" yaml:"region" validate:"required,min=1"`
	Prefix         string `json:"Prefix" yaml:"prefix"`
	KeyID          string `json:"KeyID" yaml:"keyID"`
	ApplicationKey string `json:"ApplicationKey" yaml:"applicationKey"`
}

type S3Config struct {
	BucketName      string `json:"BucketName" yaml:"bucketName" validate:"required,min=1"`
	Region          string `json:"Region" yaml:"region" validate:"required,min=1"`
	Prefix          string `json:"Prefix" yaml:"prefix"`
	Endpoint        string `json:"Endpoint" yaml:"endpoint" validate:"url"`
	UsePathStyle    bool   `json:"UsePathStyle"`
	AccessKeyID     string `json:"AccessKeyID" yaml:"accessKeyID"`
	SecretAccessKey string `json:"SecretAccessKey" yaml:"secretAccessKey"`
}

type FactGiverConfig struct {
	Storage       StorageConfig `json:"Storage" yaml:"storage" validate:"required"`
	FactsFileName string        `json:"FactsFileName" yaml:"factsFileName" validate:"required"`
}

type AvailableLanguageConfig struct {
	Name    string `json:"Name" yaml:"name" validate:"required"`
	Alt     string `json:"Alt" yaml:"alt"`
	Flag    string `json:"Flag" yaml:"flag" validate:"url"`
	LocFile string `json:"LocFile" yaml:"locFile" validate:"required,filepath"`
}

type AuthConfig struct {
	Salt string   `json:"Salt" yaml:"salt" validate:"required"`
	Db   DbConfig `json:"Db" yaml:"db" validate:"required"`
}

type DbConfig struct {
	Type string        `json:"Type" yaml:"type" validate:"required,oneof=sqlite3"`
	Cfg  Sqlite3Config `json:"Config" yaml:"config"`
}

type Sqlite3Config struct {
	DSN string `json:"DSN" yaml:"dsn" validate:"required"`
}

type MailConfig struct {
	ClientHost  string        `json:"ClientHost" yaml:"clientHost" validate:"required"`
	MailHost    string        `json:"MailHost" yaml:"mailHost" validate:"required"`
	PublicName  string        `json:"PublicName" yaml:"publicName" validate:"required"`
	MailAddress string        `json:"MailAddress" yaml:"mailAddress" validate:"required"`
	Username    string        `json:"Username" yaml:"username" validate:"required"`
	Password    string        `json:"Password" yaml:"password" validate:"required"`
	Salt        string        `json:"Salt" yaml:"salt" validate:"required"`
	Trigger     TriggerConfig `json:"Trigger" yaml:"trigger" validate:"required"`
}

type TriggerConfig struct {
	OnNewPost string `json:"OnNewPost" yaml:"onNewPost" validate:"cron,required"`
}

type MetaConfig struct {
	GoogleSiteVerification string `json:"GoogleSiteVerification" yaml:"googleSiteVerification"`
}

type PhotoStorageConfig struct {
	Full           PhotoTypeConfig    `json:"Full" yaml:"full" validate:"required"`
	Webp           PhotoTypeConfig    `json:"Webp" yaml:"webp"`
	Thumbnail1600p PhotoTypeConfig    `json:"Thumbnail1600p" yaml:"thumbnail1600p"`
	Thumbnail1200p PhotoTypeConfig    `json:"Thumbnail1200p" yaml:"thumbnail1200p"`
	Thumbnail800p  PhotoTypeConfig    `json:"Thumbnail800p" yaml:"thumbnail800p"`
	Thumbnail560p  PhotoTypeConfig    `json:"Thumbnail560p" yaml:"thumbnail560p"`
	Thumbnail320p  PhotoTypeConfig    `json:"Thumbnail320p" yaml:"thumbnail320p"`
	HomePageGifs   HomePageGifsConfig `json:"HomePageGifs" yaml:"homePageGifs"`
}

type PhotoTypeConfig struct {
	BaseUrl string `json:"BaseUrl" yaml:"baseUrl" validate:"url,required"`
}

type HomePageGifsConfig struct {
	BaseUrl string   `json:"BaseUrl" yaml:"baseUrl" validate:"url,required"`
	Indexes []string `json:"Indexes" yaml:"indexes"`
}

func LoadConfig(path string, config *Config) error {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	expandedFileBytes := []byte(os.ExpandEnv(string(fileBytes)))

	if err = yaml.Unmarshal(expandedFileBytes, config); err != nil {
		return err
	}

	return nil
}

func InitConfig(path string) (*Config, error) {
	config := &Config{}
	if err := LoadConfig(path, config); err != nil {
		return nil, err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return nil, err
	}

	return config, nil
}
