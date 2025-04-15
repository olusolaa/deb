package config

import (
	"log"
	"strings"

	"github.com/spf13/viper" // Import viper
)

type Config struct {
	Port               string
	CorsAllowedOrigin  string
	OpenRouterAPIKey   string
	OpenRouterBaseURL  string
	LLMModelName       string
	MongoDBURI         string // Added for MongoDB connection
	GoogleClientID     string // Added for Google OAuth
	GoogleClientSecret string // Added for Google OAuth
	GoogleRedirectURL  string // Added for Google OAuth Callback
	JWTSecret          string // Added for signing our application's JWTs
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

	// Enable Viper to read Environment Variables
	vip