package config

import (
	"encoding/json"
	"html/template"
	"os"
	"time"

	"git.kirsle.net/apps/barertc/pkg/log"
	"github.com/google/uuid"
	"github.com/pelletier/go-toml/v2"
)

// Version of the config format - when new fields are added, it will attempt
// to write the settings.toml to disk so new defaults populate.
var currentVersion = 17

// Config for your BareRTC app.
type Config struct {
	Version int `toml:"" comment:"Version of your config file (do not touch). When new features are added to BareRTC,\nthe Version is incremented and your settings.toml is written with sensible defaults added"` // will re-save your settings.toml on migrations

	JWT struct {
		Enabled        bool
		Strict         bool
		SecretKey      string
		LandingPageURL string
	} `toml:"" comment:"Use JWT tokens to log users into chat from your main website."`

	Title      string `toml:"" comment:"Your chat room title (plain text)"`
	Branding   string `toml:"" comment:"Your logo in the top-left corner of page. This can just be your Title again,\nOr you can use HTML here for custom style or image."`
	WebsiteURL string `toml:"" comment:"Your main website's base URL, for e.g. avatars and profile URLs to be relative to"`

	CORSHosts       []string
	AdminAPIKey     string
	PermitNSFW      bool
	BlockableAdmins bool

	UseXForwardedFor bool

	WebSocketReadLimit   int64
	WebSocketSendTimeout int
	MaxImageWidth        int
	PreviewImageWidth    int

	TURN TurnConfig `toml:"" comment:"Configure your TURN or STUN servers here.\n\nSTUN servers help WebRTC clients connect peer-to-peer for video, which is\npreferable as it saves on your bandwidth. You should list at least one, and\nthere are many public servers available such as Google's.\n\nTURN servers help WebRTC clients connect when a direct connection isn't\npossible. An open source server called 'coturn' can do both STUN and TURN."`

	PublicChannels []Channel `toml:"" comment:"Your pre-defined common public chat rooms.\n"`

	WebhookURLs []WebhookURL

	VIP VIP

	MessageFilters []*MessageFilter
	ModerationRule []*ModerationRule

	DirectMessageHistory DirectMessageHistory
	DMLinkSpamProtection DMLinkSpamProtection

	Strings Strings

	Logging Logging
}

type TurnConfig struct {
	URLs       []string
	Username   string
	Credential string
}

type VIP struct {
	Name           string
	Branding       string
	Icon           string
	MutuallySecret bool
}

type DirectMessageHistory struct {
	Enabled           bool
	SQLiteDatabase    string
	RetentionDays     int
	DisclaimerMessage string
}

type DMLinkSpamProtection struct {
	Enabled    bool          `toml:"" comment:"Spam detection in case a user pastes the same URL into several DM threads in a short timeframe,\ne.g. trying to take your entire userbase to a competing chat room."`
	MaxThreads int           `toml:"" comment:"The max number of DM threads the user can paste the same link into"`
	TimeLimit  time.Duration `toml:"" comment:"Rate limit (in seconds), if the user spams MaxThreads within the rate limit they are kicked or banned from chat."`
	BanHours   time.Duration `toml:"" comment:"The user will be kicked the first time, if they log back in and spam once more they are banned for this many hours."`
}

// GetChannels returns a JavaScript safe array of the default PublicChannels.
func (c Config) GetChannels() template.JS {
	data, _ := json.Marshal(c.PublicChannels)
	return template.JS(data)
}

// GetChannel looks up and returns a channel by ID.
func (c Config) GetChannel(id string) (Channel, bool) {
	for _, ch := range c.PublicChannels {
		if ch.ID == id {
			return ch, true
		}
	}
	return Channel{}, false
}

// Channel config for a default public room.
type Channel struct {
	ID           string // Like "lobby"
	Name         string // Like "Main Chat Room"
	Icon         string `toml:",omitempty"` // CSS class names for room icon (optional)
	VIP          bool   // For VIP users only
	PermitPhotos bool   // photos are allowed to be shared

	// ChatServer messages to send to the user immediately upon connecting.
	WelcomeMessages []string

	EchoMessagesOnJoin int
}

// WebhookURL allows tighter integration with your website.
type WebhookURL struct {
	Name    string
	Enabled bool
	URL     string
}

// Strings config for customizing certain user-facing messaging around the app.
type Strings struct {
	ModRuleErrorCameraAlwaysNSFW string
	ModRuleErrorNoBroadcast      string
	ModRuleErrorNoVideo          string
	ModRuleErrorNoImage          string
}

// Logging configs to monitor channels or usernames.
type Logging struct {
	Enabled   bool
	Directory string
	Channels  []string
	Usernames []string
}

// ModerationRule applies certain rules to moderate specific users.
type ModerationRule struct {
	Username         string
	CameraAlwaysNSFW bool
	NoBroadcast      bool
	NoVideo          bool
	NoImage          bool
	NoDarkVideo      bool
}

// Current loaded configuration.
var Current = DefaultConfig()

// DefaultConfig returns sensible defaults and will write the initial
// settings.toml file to disk.
func DefaultConfig() Config {
	var c = Config{
		Title:       "BareRTC",
		Branding:    "BareRTC",
		WebsiteURL:  "https://www.example.com",
		AdminAPIKey: uuid.New().String(),
		CORSHosts: []string{
			"https://www.example.com",
		},
		WebSocketReadLimit:   1024 * 1024 * 40, // 40 MB.
		WebSocketSendTimeout: 10,               // seconds
		MaxImageWidth:        1280,
		PreviewImageWidth:    360,
		PublicChannels: []Channel{
			{
				ID:   "lobby",
				Name: "Accueil",
				WelcomeMessages: []string{
					"Bienvenue sur le serveur de chat !",
					"Merci de respecter les règles de base :\n\n1. Amusez-vous\n2. Soyez gentil",
				},
				EchoMessagesOnJoin: 10,
			},
			{
				ID:   "offtopic",
				Name: "Hors sujet",
				WelcomeMessages: []string{
					"Bienvenue dans le canal Hors sujet !",
				},
				PermitPhotos: true,
			},
			{
				ID:           "vip",
				Name:         "Réservé aux VIP",
				VIP:          true,
				PermitPhotos: true,
				WelcomeMessages: []string{
					"Ce canal est réservé uniquement aux opérateurs et aux VIP.",
				},
			},
		},
		TURN: TurnConfig{
			URLs: []string{
				"stun:stun.l.google.com:19302",
			},
		},
		WebhookURLs: []WebhookURL{
			{
				Name: "report",
				URL:  "https://example.com/barertc/report",
			},
			{
				Name: "profile",
				URL:  "https://example.com/barertc/user-profile",
			},
		},
		VIP: VIP{
			Name:     "VIP",
			Branding: "<em>VIP Members</em>",
			Icon:     "fa fa-circle",
		},
		MessageFilters: []*MessageFilter{
			{
				PublicChannels:  true,
				PrivateChannels: true,
				KeywordPhrases: []string{
					`\bswear words\b`,
					`\b(swearing|cursing)\b`,
					`suck my ([^\s]+)`,
				},
				CensorMessage:      true,
				ChatServerResponse: "Watch your language.",
			},
		},
		ModerationRule: []*ModerationRule{
			{
				Username: "example",
			},
		},
		Strings: Strings{
			ModRuleErrorCameraAlwaysNSFW: "Une règle de modération du serveur de chat vous oblige actuellement à garder votre caméra marquée comme Explicite. Veuillez contacter un modérateur si vous avez des questions à ce sujet.",
			ModRuleErrorNoBroadcast:      "Une règle de modération du serveur de chat restreint actuellement votre capacité à partager votre webcam. Veuillez contacter un opérateur pour plus d'informations.",
			ModRuleErrorNoVideo:          "Une règle de modération du serveur de chat restreint actuellement votre capacité à regarder des webcams. Veuillez contacter un opérateur pour plus d'informations.",
			ModRuleErrorNoImage:          "Une règle de modération du serveur de chat restreint actuellement votre capacité à partager des images. Veuillez contacter un opérateur pour plus d'informations.",
		},
		DirectMessageHistory: DirectMessageHistory{
			Enabled:           false,
			SQLiteDatabase:    "database.sqlite",
			RetentionDays:     90,
			DisclaimerMessage: `<i class="fa fa-info-circle mr-1"></i> <strong>Rappel :</strong> merci d'adopter un comportement honorable dans les messages privés.`,
		},
		DMLinkSpamProtection: DMLinkSpamProtection{
			Enabled:    true,
			MaxThreads: 5,
			TimeLimit:  60 * 60,
			BanHours:   24,
		},
		Logging: Logging{
			Directory: "./logs",
			Channels:  []string{"lobby", "offtopic"},
			Usernames: []string{},
		},
	}
	c.JWT.Strict = true
	return c
}

// LoadSettings reads a settings.toml from disk if available.
func LoadSettings() error {
	data, err := os.ReadFile("./settings.toml")
	if err != nil {
		// Settings file didn't exist, create the default one.
		if os.IsNotExist(err) {
			WriteSettings()
			return nil
		}

		return err
	}

	if err = toml.Unmarshal(data, &Current); err != nil {
		return err
	}

	// Have we added new config fields? Save the settings.toml.
	if Current.Version != currentVersion {
		log.Warn("New options are available for your settings.toml file. Your settings will be re-saved now.")
		Current.Version = currentVersion
		if err := WriteSettings(); err != nil {
			log.Error("Couldn't write your settings.toml file: %s", err)
		}
	}

	return err
}

// WriteSettings will commit the settings.toml to disk.
func WriteSettings() error {
	log.Error("Note: initial settings.toml was written to disk.")
	buf, err := toml.Marshal(Current)
	if err != nil {
		return err
	}

	return os.WriteFile("./settings.toml", buf, 0644)
}

// GetModerationRule returns a matching ModerationRule for the given user, or nil if no rule is found.
func (c Config) GetModerationRule(username string) *ModerationRule {
	for _, rule := range c.ModerationRule {
		if rule.Username == username {
			return rule
		}
	}
	return nil
}
