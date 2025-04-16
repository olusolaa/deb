package config

import (
	"log"
	"strings"

	"github.com/spf13/viper" // Import viper
)

type Config struct {
	Port                  string
	CorsAllowedOrigin     string
	OpenRouterAPIKey      string
	OpenRouterBaseURL     string
	LLMModelName          string
	MongoDBURI            string // Added for MongoDB connection
	GoogleClientID        string // Added for Google OAuth
	GoogleClientSecret    string // Added for Google OAuth
	GoogleRedirectURL     string // Added for Google OAuth Callback
	JWTSecret             string // Added for signing our application's JWTs
	BibleDBPath           string // Path to the Bible SQLite database
	ChatRateLimitEnabled  bool   // Whether chat rate limiting is enabled
	ChatRateLimitPerDay   int    // Maximum number of chat requests per user per day
	YearlyTheme           string // Theme of the year for Bible reading plans
	DefaultTargetAudience string // Default target audience for Bible reading plans
}

// Load uses Viper to load configuration from .env file and environment variables.
func Load() *Config {
	// --- Viper Setup ---

	// Set the name of the config file (without extension)
	viper.SetConfigName(".env")
	// Set the type of the config file (viper needs this for proper parsing)
	viper.SetConfigType("env")
	// Add the path to look for the config file.
	// "." means look in the current directory (where the Go program is run from).
	// If you run `go run ./cmd/server/main.go` from the `backend` directory,
	// viper will look for `backend/.env`.
	viper.AddConfigPath(".")

	// Optional: Set default values
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("CORS_ALLOWED_ORIGIN", "http://localhost:3000")
	viper.SetDefault("OPENROUTER_BASE_URL", "https://openrouter.ai")
	viper.SetDefault("LLM_MODEL_NAME", "openai/gpt-3.5-turbo")
	viper.SetDefault("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback") // Default callback URL
	viper.SetDefault("BIBLE_DB_PATH", "./data/bible.db")                                  // Default Bible database path
	viper.SetDefault("YEARLY_THEME", "Faith and Perseverance")                            // Default yearly theme
	viper.SetDefault("DEFAULT_TARGET_AUDIENCE", "adult believer")                         // Default target audience

	// Enable Viper to read Environment Variables
	viper.AutomaticEnv()

	// Make environment variables case-insensitive and replace dots with underscores
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("WARN: Config file not found: %v\n", err)
	}

	// Initialize Config
	// Set default values for rate limiting if not provided
	if !viper.IsSet("CHAT_RATE_LIMIT_ENABLED") {
		viper.SetDefault("CHAT_RATE_LIMIT_ENABLED", "true")
	}
	if !viper.IsSet("CHAT_RATE_LIMIT_PER_DAY") {
		viper.SetDefault("CHAT_RATE_LIMIT_PER_DAY", "5")
	}

	return &Config{
		Port:                  viper.GetString("PORT"),
		CorsAllowedOrigin:     viper.GetString("CORS_ALLOWED_ORIGIN"),
		OpenRouterAPIKey:      viper.GetString("OPENROUTER_API_KEY"),
		OpenRouterBaseURL:     viper.GetString("OPENROUTER_BASE_URL"),
		LLMModelName:          viper.GetString("LLM_MODEL_NAME"),
		MongoDBURI:            viper.GetString("MONGODB_URI"),
		GoogleClientID:        viper.GetString("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:    viper.GetString("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:     viper.GetString("GOOGLE_REDIRECT_URL"),
		JWTSecret:             viper.GetString("JWT_SECRET"),
		BibleDBPath:           viper.GetString("BIBLE_DB_PATH"),
		ChatRateLimitEnabled:  strings.ToLower(viper.GetString("CHAT_RATE_LIMIT_ENABLED")) == "true",
		ChatRateLimitPerDay:   viper.GetInt("CHAT_RATE_LIMIT_PER_DAY"),
		YearlyTheme:           viper.GetString("YEARLY_THEME"),
		DefaultTargetAudience: viper.GetString("DEFAULT_TARGET_AUDIENCE"),
	}
}
