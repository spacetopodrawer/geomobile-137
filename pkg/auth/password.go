package auth

import (
	"log"

	"golang.org/x/crypto/bcrypt"
)

const defaultCost = bcrypt.DefaultCost

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), defaultCost)
	if err != nil {
		log.Printf("❌ Error hashing password: %v", err)
		return "", err
	}

	log.Printf("✅ Password hashed successfully")
	return string(hash), nil
}

func VerifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		log.Printf("❌ Password verification failed: %v", err)
		return false
	}

	log.Printf("✅ Password verified successfully")
	return true
}
