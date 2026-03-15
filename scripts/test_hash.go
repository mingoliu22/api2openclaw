package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := "admin123"

	// Generate hash with cost 10
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Generated hash: %s\n", string(hash))

	// Test with the stored hash
	storedHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err == nil {
		fmt.Println("Stored hash matches password!")
	} else {
		fmt.Printf("Stored hash does NOT match: %v\n", err)
	}
}
