package config

import (
	"log"
	"strings"

	"github.com/spf13/viper" // Import viper
)

type Config struct {
	Port              string
	CorsAllowedOrigin string
	OpenRouterAPIKey  string
	OpenRouterBaseURL string
	LLMModelName      string
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

	// Enable Viper to read Environment Variables
	viper.AutomaticEnv()
	// Optional: Configure environment variable prefix and replacer if needed
	// viper.SetEnvPrefix("APP") // e.g., APP_PORT, APP_OPENROUTER_API_KEY
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // Replace dots with underscores if using nested keys

	// --- Read Configuration ---

	// Attempt to read the config file
	err := viper.ReadInConfig()
	if err != nil {
		// If the config file is not found, Viper will proceed using defaults/env vars.
		// Only panic if there's an error *parsing* an existing config file.
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("INFO: .env file not found, using environment variables and defaults.")
		} else {
			// Config file was found but another error was produced
			log.Fatalf("FATAL: Error reading config file: %v", err)
		}
	} else {
		log.Printf("INFO: Using configuration from: %s", viper.ConfigFileUsed())
	}

	// --- Retrieve Values and Build Config Struct ---

	// Retrieve values using Viper getters
	cfg := &Config{
		Port:              viper.GetString("PORT"),
		CorsAllowedOrigin: viper.GetString("CORS_ALLOWED_ORIGIN"),
		OpenRouterAPIKey:  viper.GetString("OPENROUTER_API_KEY"), // Reads from .env OR environment variable
		OpenRouterBaseURL: viper.GetString("OPENROUTER_BASE_URL"),
		LLMModelName:      viper.GetString("LLM_MODEL_NAME"),
	}

	// --- Validation (moved after loading) ---
	if cfg.OpenRouterAPIKey == "" {
		// This check now happens *after* Viper has tried loading from both .env and environment.
		log.Fatal("FATAL: OPENROUTER_API_KEY is not set in .env file or as an environment variable.")
	}

	// Log the effective configuration (excluding secrets)
	log.Printf("INFO: Effective configuration:")
	log.Printf("  Port: %s", cfg.Port)
	log.Printf("  CORS Allowed Origin: %s", cfg.CorsAllowedOrigin)
	log.Printf("  OpenRouter Base URL: %s", cfg.OpenRouterBaseURL)
	log.Printf("  LLM Model Name: %s", cfg.LLMModelName)
	// Avoid logging the API key itself: log.Printf("  OpenRouter API Key: %s", cfg.OpenRouterAPIKey)

	return cfg
}
