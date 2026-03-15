package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	storedHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

	passwords := []string{"admin123", "admin", "password", "123456"}

	for _, pwd := range passwords {
		err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(pwd))
		if err == nil {
			fmt.Printf("Password '%s' matches!\n", pwd)
		} else {
			fmt.Printf("Password '%s' does NOT match\n", pwd)
		}
	}

	// Generate new hash for admin123
	newHash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), 10)
	fmt.Printf("\nNew hash for 'admin123': %s\n", string(newHash))
}
