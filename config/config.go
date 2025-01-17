package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-faster/errors"
	"github.com/joho/godotenv"
)

// Config holds all configuration values for the userbot
type Config struct {
	Phone         string
	AppID         int
	AppHash       string
	Debug         bool
	DataDir       string
	SessionDir    string
	LogChannel    int64
	CommandPrefix string

	OpenRouterAPIKey string
}

var configInstance *Config

// Instance returns the singleton config instance
func Instance() *Config {
	return configInstance
}

func sessionFolder(phone string) string {
	var out []rune
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			out = append(out, r)
		}
	}
	return "phone-" + string(out)
}

// Load creates a new Config instance from environment variables
func Load() (*Config, error) {
	if configInstance != nil {
		return configInstance, nil
	}

	// Load dotenv if available
	_ = godotenv.Load()

	phone := os.Getenv("TG_PHONE")
	if phone == "" {
		return nil, errors.New("TG_PHONE environment variable is required")
	}

	appID, err := strconv.Atoi(os.Getenv("APP_ID"))
	if err != nil {
		return nil, errors.Wrap(err, "invalid APP_ID")
	}

	appHash := os.Getenv("APP_HASH")
	if appHash == "" {
		return nil, errors.New("APP_HASH environment variable is required")
	}

	debug := os.Getenv("DEBUG") == "true"
	cmdPrefix := os.Getenv("COMMAND_PREFIX")
	if cmdPrefix == "" {
		cmdPrefix = "."
	}

	logChannel, err := strconv.ParseInt(strings.TrimPrefix(os.Getenv("LOG_CHANNEL"), "-100"), 10, 64)
	if err != nil {
		logChannel = 0
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "~/.config/macron"
	}

	sessionDir := filepath.Join(dataDir, sessionFolder(phone))
	err = os.MkdirAll(sessionDir, 0700)
	if err != nil {
		return nil, errors.Wrap(err, "create session dir")
	}

	configInstance = &Config{
		Phone:            phone,
		AppID:            appID,
		AppHash:          appHash,
		Debug:            debug,
		DataDir:          dataDir,
		SessionDir:       sessionDir,
		LogChannel:       logChannel,
		CommandPrefix:    cmdPrefix,
		OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
	}
	return configInstance, nil
}
