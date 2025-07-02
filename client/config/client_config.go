package config

import (
	"bytes"
	"os"

	"git.kirsle.net/apps/barertc/pkg/log"
	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
)

// Version of the config format - when new fields are added, it will attempt
// to write the chatbot.toml to disk so new defaults populate.
var currentVersion = -1

// Config for your BareBot robot.
type Config struct {
	Version int // will re-save your chatbot.toml on migrations

	// Chat server config
	BareRTC BareRTC

	// Profile settings for their chat username
	Profile Profile

	WebSocketReadLimit int64
}

type BareRTC struct {
	AdminAPIKey string
	URL         string
}

type Profile struct {
	Username   string
	Nickname   string
	ProfileURL string
	AvatarURL  string
	Emoji      string
	Gender     string
	IsAdmin    bool
}

// Current loaded configuration.
var Current = DefaultConfig()

// DefaultConfig returns sensible defaults and will write the initial
// chatbot.toml file to disk.
func DefaultConfig() Config {
	var c = Config{
		BareRTC: BareRTC{
			AdminAPIKey: uuid.New().String(),
			URL:         "http://localhost:9000",
		},
		Profile: Profile{
			Username: "barebot",
			Nickname: "BareBOT",
			Emoji:    "🤖",
		},
		WebSocketReadLimit: 1024 * 1024 * 40, // 40 MB.
	}
	return c
}

// LoadSettings reads a chatbot.toml from disk if available.
func LoadSettings() error {
	data, err := os.ReadFile("./chatbot.toml")
	if err != nil {
		// Settings file didn't exist, create the default one.
		if os.IsNotExist(err) {
			WriteSettings()
			return nil
		}

		return err
	}

	_, err = toml.Decode(string(data), &Current)
	if err != nil {
		return err
	}

	// Have we added new config fields? Save the chatbot.toml.
	if Current.Version != currentVersion {
		log.Warn("New options are available for your chatbot.toml file. Your settings will be re-saved now.")
		Current.Version = currentVersion
		if err := WriteSettings(); err != nil {
			log.Error("Couldn't write your chatbot.toml file: %s", err)
		}
	}

	return err
}

// WriteSettings will commit the chatbot.toml to disk.
func WriteSettings() error {
	if Current.Version == 0 {
		Current.Version = currentVersion
	}

	log.Error("Note: initial chatbot.toml was written to disk.")
	var buf = new(bytes.Buffer)
	err := toml.NewEncoder(buf).Encode(Current)
	if err != nil {
		return err
	}
	return os.WriteFile("./chatbot.toml", buf.Bytes(), 0644)
}
