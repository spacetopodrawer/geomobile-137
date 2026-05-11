package auth

import (
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	jwt.RegisteredClaims
}

func GenerateToken(userID, email, secret string, duration time.Duration) (string, error) {
	claims := &TokenClaims{
		UserID:    userID,
		Email:     email,
		ExpiresAt: time.Now().Add(duration).Unix(),
		IssuedAt:  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Printf("❌ Error generating token: %v", err)
		return "", err
	}

	log.Printf("✅ Token generated for user: %s", userID)
	return tokenString, nil
}

func VerifyToken(tokenString, secret string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		log.Printf("❌ Error parsing token: %v", err)
		return nil, err
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		log.Printf("❌ Invalid token")
		return nil, jwt.ErrSignatureInvalid
	}

	log.Printf("✅ Token verified for user: %s", claims.UserID)
	return claims, nil
}

func RefreshToken(oldToken, secret string, duration time.Duration) (string, error) {
	claims, err := VerifyToken(oldToken, secret)
	if err != nil {
		return "", err
	}

	return GenerateToken(claims.UserID, claims.Email, secret, duration)
}
