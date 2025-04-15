package service

import (
	"bibleapp/backend/internal/config"
	"bibleapp/backend/internal/domain"
	"bibleapp/backend/internal/repository"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleUserInfo holds the relevant fields from Google's userinfo endpoint
type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// AuthService handles authentication logic (OAuth, JWT)
type AuthService struct {
	googleOAuthConfig *oauth2.Config
	userRepo          repository.UserRepository
	jwtSecret         []byte
}

// NewAuthService creates a new AuthService
func NewAuthService(googleOAuthConfig *oauth2.Config, userRepo repository.UserRepository, jwtSecret string) *AuthService {
	if len(jwtSecret) == 0 {
		log.Fatal("FATAL: JWT Secret cannot be empty in AuthService")
	}
	return &AuthService{
		googleOAuthConfig: googleOAuthConfig,
		userRepo:          userRepo,
		jwtSecret:         []byte(jwtSecret),
	}
}

// SetupGoogleOAuthConfig initializes the OAuth2 config from the main application config
func SetupGoogleOAuthConfig(cfg *config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}
}

// GenerateOauthState generates a secure random state string for OAuth flow
func GenerateOauthState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate oauth state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetGoogleLoginURL generates the URL the user should be redirected to for Google login
func (s *AuthService) GetGoogleLoginURL(state string) string {
	return s.googleOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline) // Use offline for potential refresh token later
}

// HandleGoogleCallback processes the callback from Google
func (s *AuthService) HandleGoogleCallback(ctx context.Context, code string) (*domain.User, string, error) {
	// 1. Exchange code for token
	token, err := s.googleOAuthConfig.Exchange(ctx, code)
	if err != nil {
		log.Printf("ERROR: Failed to exchange Google auth code for token: %v", err)
		return nil, "", fmt.Errorf("code exchange failed: %w", err)
	}

	// Check if token is valid (basic check)
	if !token.Valid() {
		log.Printf("ERROR: Invalid token received from Google exchange")
		return nil, "", errors.New("invalid token received")
	}

	// 2. Get user info from Google
	userInfo, err := s.fetchGoogleUserInfo(ctx, token)
	if err != nil {
		log.Printf("ERROR: Failed to fetch user info from Google: %v", err)
		return nil, "", fmt.Errorf("failed to get user info: %w", err)
	}

	// 3. Find or Create User in DB
	user, err := s.userRepo.FindByGoogleID(ctx, userInfo.ID)
	if err != nil {
		// Handle potential DB errors during find
		log.Printf("ERROR: Database error finding user by Google ID %s: %v", userInfo.ID, err)
		return nil, "", fmt.Errorf("database error finding user: %w", err)
	}

	if user == nil { // User not found, create them
		newUser := &domain.User{
			GoogleID: userInfo.ID,
			Email:    userInfo.Email,
			Name:     userInfo.Name,
			Picture:  userInfo.Picture,
			// CreatedAt/UpdatedAt set by repository
		}
		user, err = s.userRepo.Create(ctx, newUser)
		if err != nil {
			log.Printf("ERROR: Failed to create user %s (%s): %v", userInfo.Email, userInfo.ID, err)
			return nil, "", fmt.Errorf("failed to create user: %w", err)
		}
		log.Printf("INFO: Created new user: ID=%s, Email=%s", user.ID, user.Email)
	} else {
		log.Printf("INFO: Found existing user: ID=%s, Email=%s", user.ID, user.Email)
		// Optional: Update user info if it changed (Name, Picture)
		// Requires an Update method in the repository
		// if user.Name != userInfo.Name || user.Picture != userInfo.Picture { ... s.userRepo.Update(ctx, user) ... }
	}

	// 4. Generate JWT for our application
	appToken, err := s.generateJWT(user)
	if err != nil {
		log.Printf("ERROR: Failed to generate JWT for user %s: %v", user.ID, err)
		return nil, "", fmt.Errorf("failed to generate session token: %w", err)
	}

	return user, appToken, nil
}

// fetchGoogleUserInfo uses the Google token to get user details
func (s *AuthService) fetchGoogleUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	client := s.googleOAuthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed requesting user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google userinfo request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading user info response body: %w", err)
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed unmarshalling user info JSON: %w", err)
	}

	return &userInfo, nil
}

// generateJWT creates a signed JWT for the given user
func (s *AuthService) generateJWT(user *domain.User) (string, error) {
	// Define JWT claims
	claims := jwt.MapClaims{
		"sub": user.ID,                                   // Subject (standard claim) - our internal user ID
		"gid": user.GoogleID,                             // Google ID (custom claim)
		"eml": user.Email,                                // Email (custom claim)
		"nam": user.Name,                                 // Name (custom claim)
		"pic": user.Picture,                              // Picture URL (custom claim)
		"iss": "bibleapp",                                // Issuer (standard claim)
		"aud": "bibleapp_users",                          // Audience (standard claim)
		"exp": time.Now().Add(time.Hour * 24 * 7).Unix(), // Expiration time (e.g., 7 days)
		"iat": time.Now().Unix(),                         // Issued at (standard claim)
		"nbf": time.Now().Unix(),                         // Not before (standard claim)
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret
	signedToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signedToken, nil
}

// ValidateJWT verifies a JWT string and returns the claims if valid
func (s *AuthService) ValidateJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Return the secret key
		return s.jwtSecret, nil
	})

	if err != nil {
		log.Printf("WARN: JWT parsing/validation failed: %v", err)
		return nil, err // Returns specific errors like TokenExpiredError, etc.
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Optional: Perform additional checks on claims (e.g., issuer, audience)
		// Manually check issuer and audience since MapClaims doesn't have these verification methods
		if iss, ok := claims["iss"].(string); !ok || iss != "bibleapp" {
			return nil, errors.New("invalid issuer")
		}
		if aud, ok := claims["aud"].(string); !ok || aud != "bibleapp_users" {
			return nil, errors.New("invalid audience")
		}
		return claims, nil
	} else {
		return nil, errors.New("invalid JWT token")
	}
}
