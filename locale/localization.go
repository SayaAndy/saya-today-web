package locale

import (
	"os"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

type LocaleConfig struct {
	TagsLabel       string                `yaml:"TagsLabel" json:"TagsLabel"`
	BlogSearch      BlogSearchConfig      `yaml:"BlogSearch" json:"BlogSearch"`
	GlobalMap       GlobalMapConfig       `yaml:"GlobalMap" json:"GlobalMap"`
	HomePage        HomePageConfig        `yaml:"HomePage" json:"HomePage"`
	UnsubscribePage UnsubscribePageConfig `yaml:"UnsubscribePage" json:"UnsubscribePage"`
	Metadata        MetadataConfig        `yaml:"Metadata" json:"Metadata"`
	Mail            MailConfig            `yaml:"Mail" json:"Mail"`
	UserProfile     UserProfileConfig     `yaml:"UserProfile" json:"UserProfile"`
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
	Contacts               string `yaml:"Contacts" json:"Contacts"`
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

type UnsubscribePageConfig struct {
	Header        string `yaml:"Header" json:"Header"`
	UnsetCode     string `yaml:"UnsetCode" json:"UnsetCode"`
	InvalidCode   string `yaml:"InvalidCode" json:"InvalidCode"`
	OnServerError string `yaml:"OnServerError" json:"OnServerError"`
	Success       string `yaml:"Success" json:"Success"`
}

type MetadataConfig struct {
	Action    string `yaml:"Action" json:"Action"`
	Published string `yaml:"Published" json:"Published"`
}

type MailConfig struct {
	UnsubscribeFooter string            `yaml:"UnsubscribeFooter" json:"UnsubscribeFooter"`
	VerifyEmail       VerifyEmailConfig `yaml:"VerifyEmail" json:"VerifyEmail"`
	NewPost           NewPostConfig     `yaml:"NewPost" json:"NewPost"`
}

type VerifyEmailConfig struct {
	Subject   string `yaml:"Subject" json:"Subject"`
	Welcome   string `yaml:"Welcome" json:"Welcome"`
	Intro     string `yaml:"Intro" json:"Intro"`
	GotoLink  string `yaml:"GotoLink" json:"GotoLink"`
	InputCode string `yaml:"InputCode" json:"InputCode"`
	IfRandom  string `yaml:"IfRandom" json:"IfRandom"`
}

type NewPostConfig struct {
	Subject    string `yaml:"Subject" json:"Subject"`
	CapturedOn string `yaml:"CapturedOn" json:"CapturedOn"`
	Intro      string `yaml:"Intro" json:"Intro"`
}

type UserProfileConfig struct {
	Header                       string `yaml:"Header" json:"Header"`
	EmailHeader                  string `yaml:"EmailHeader" json:"EmailHeader"`
	VerificationCodeHeader       string `yaml:"VerificationCodeHeader" json:"VerificationCodeHeader"`
	TypeEmail                    string `yaml:"TypeEmail" json:"TypeEmail"`
	SendCodeButton               string `yaml:"SendCodeButton" json:"SendCodeButton"`
	VerifyButton                 string `yaml:"VerifyButton" json:"VerifyButton"`
	VerificationCodeSent         string `yaml:"VerificationCodeSent" json:"VerificationCodeSent"`
	DelayTilVerification         string `yaml:"DelayTilVerification" json:"DelayTilVerification"`
	FailedEmailRender            string `yaml:"FailedEmailRender" json:"FailedEmailRender"`
	VerificationCodeSendingError string `yaml:"VerificationCodeSendingError" json:"VerificationCodeSendingError"`
	EmailAlreadyValidated        string `yaml:"EmailAlreadyValidated" json:"EmailAlreadyValidated"`
	EmailTaken                   string `yaml:"EmailTaken" json:"EmailTaken"`
	EmailEmpty                   string `yaml:"EmailEmpty" json:"EmailEmpty"`
	VerificationSuccess          string `yaml:"VerificationSuccess" json:"VerificationSuccess"`
	VerificationFailed           string `yaml:"VerificationFailed" json:"VerificationFailed"`
	VerificationEmpty            string `yaml:"VerificationEmpty" json:"VerificationEmpty"`
	SubscriptionHeader           string `yaml:"SubscriptionHeader" json:"SubscriptionHeader"`
	TagsNone                     string `yaml:"TagsNone" json:"TagsNone"`
	TagsAll                      string `yaml:"TagsAll" json:"TagsAll"`
	TagsSpecific                 string `yaml:"TagsSpecific" json:"TagsSpecific"`
	TagDoesNotExist              string `yaml:"TagDoesNotExist" json:"TagDoesNotExist"`
	TagAlreadyAdded              string `yaml:"TagAlreadyAdded" json:"TagAlreadyAdded"`
	SaveButton                   string `yaml:"SaveButton" json:"SaveButton"`
	SubscribeInvalidType         string `yaml:"SubscribeInvalidType" json:"SubscribeInvalidType"`
	FailedToSubscribe            string `yaml:"FailedToSubscribe" json:"FailedToSubscribe"`
	SubscribedSuccessfully       string `yaml:"SubscribedSuccessfully" json:"SubscribedSuccessfully"`
	RefreshPage                  string `yaml:"RefreshPage" json:"RefreshPage"`
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
