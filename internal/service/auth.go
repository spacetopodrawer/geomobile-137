package service

import (
	"cadastre_ia/internal/config"
	"cadastre_ia/internal/storage"
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo *storage.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo *storage.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

// Register creates a new user account with validation and password hashing
func (a *AuthService) Register(ctx context.Context, email, password, name string) (*storage.User, string, error) {
	log.Printf("📝 [AUTH] User registration started: email=%s", email)

	// Input validation
	if err := a.ValidateRegistration(email, password, name); err != nil {
		log.Printf("❌ [AUTH] Registration validation failed: email=%s error=%v", email, err)
		return nil, "", fmt.Errorf("registration validation failed: %w", err)
	}

	// Check if user already exists
	existing, err := a.userRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		log.Printf("⚠️  [AUTH] User already exists: email=%s", email)
		return nil, "", fmt.Errorf("user with this email already exists")
	}

	// Hash password with bcrypt (cost = 12)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Printf("❌ [AUTH] Password hashing failed: email=%s error=%v", email, err)
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user record
	user := &storage.User{
		ID:           uuid.New(),
		Email:        strings.ToLower(email),
		PasswordHash: string(passwordHash),
		Name:         strings.TrimSpace(name),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := a.userRepo.Create(ctx, user); err != nil {
		log.Printf("❌ [AUTH] User creation failed: email=%s error=%v", email, err)
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	// Generate JWT token
	token, err := a.GenerateToken(user.ID.String(), user.Email, 24*time.Hour)
	if err != nil {
		log.Printf("❌ [AUTH] Token generation failed: user_id=%s error=%v", user.ID, err)
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	log.Printf("✅ [AUTH] User registered successfully: user_id=%s email=%s", user.ID, user.Email)
	return user, token, nil
}

// Login authenticates user credentials and returns JWT token
func (a *AuthService) Login(ctx context.Context, email, password string) (*storage.User, string, error) {
	log.Printf("🔐 [AUTH] Login attempt: email=%s", email)

	// Validate inputs
	if email == "" || password == "" {
		log.Printf("❌ [AUTH] Login validation failed: empty credentials")
		return nil, "", fmt.Errorf("email and password required")
	}

	// Fetch user by email
	user, err := a.userRepo.GetByEmail(ctx, strings.ToLower(email))
	if err != nil {
		log.Printf("⚠️  [AUTH] User not found: email=%s", email)
		return nil, "", fmt.Errorf("invalid credentials")
	}

	if !a.VerifyPassword(user.PasswordHash, password) {
		log.Printf("⚠️  [AUTH] Invalid password: email=%s", email)
		return nil, "", fmt.Errorf("invalid credentials")
	}

	// Generate JWT token (valid for 24 hours)
	token, err := a.GenerateToken(user.ID.String(), user.Email, 24*time.Hour)
	if err != nil {
		log.Printf("❌ [AUTH] Token generation failed: user_id=%s error=%v", user.ID, err)
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	log.Printf("✅ [AUTH] User logged in: user_id=%s email=%s", user.ID, user.Email)
	return user, token, nil
}

// GenerateToken creates a JWT token with claims
func (a *AuthService) GenerateToken(userID, email string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"exp":   now.Add(duration).Unix(),
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(a.cfg.JWTSecret))
	if err != nil {
		log.Printf("❌ [AUTH] Token signing failed: user_id=%s error=%v", userID, err)
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	log.Printf("✅ [AUTH] Token generated: user_id=%s expires_in=%.0f hours", userID, duration.Hours())
	return tokenString, nil
}

// VerifyToken validates JWT token and returns user ID
func (a *AuthService) VerifyToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.cfg.JWTSecret), nil
	})

	if err != nil {
		log.Printf("❌ [AUTH] Token parsing failed: error=%v", err)
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		log.Printf("❌ [AUTH] Invalid token")
		return "", ErrUnauthorized
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Printf("❌ [AUTH] Invalid token claims")
		return "", ErrUnauthorized
	}

	// Extract user ID from claims
	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		log.Printf("❌ [AUTH] Invalid user ID in token")
		return "", ErrUnauthorized
	}

	log.Printf("✅ [AUTH] Token verified: user_id=%s", userID)
	return userID, nil
}

// RefreshToken generates a new token for an authenticated user
func (a *AuthService) RefreshToken(ctx context.Context, userID string) (string, error) {
	log.Printf("🔄 [AUTH] Refreshing token: user_id=%s", userID)

	// Validate user still exists
	user, err := a.userRepo.GetByID(ctx, userID)
	if err != nil {
		log.Printf("❌ [AUTH] User not found for token refresh: user_id=%s", userID)
		return "", ErrUserNotFound
	}

	// Generate new token
	token, err := a.GenerateToken(user.ID.String(), user.Email, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	log.Printf("✅ [AUTH] Token refreshed: user_id=%s", userID)
	return token, nil
}

// VerifyPassword compares password with bcrypt hash
func (a *AuthService) VerifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidateRegistration checks email, password, and name constraints
func (a *AuthService) ValidateRegistration(email, password, name string) error {
	// Email validation
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return fmt.Errorf("email is required")
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}

	// Password validation (minimum 8 chars, at least one number, one uppercase, one special char)
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	hasNumber := regexp.MustCompile(`\d`).MatchString(password)
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*]`).MatchString(password)

	if !hasNumber || !hasUpper || !hasSpecial {
		return fmt.Errorf("password must contain uppercase, number, and special character")
	}

	// Name validation
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}

	if len(name) < 2 || len(name) > 100 {
		return fmt.Errorf("name must be between 2 and 100 characters")
	}

	return nil
}
