package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	hash := "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/VLg7xGx8VYKuOCVla"
	password := "password123"

	fmt.Printf("Testing hash: %s\n", hash)
	fmt.Printf("Against password: %s\n", password)
	fmt.Printf("=================================\n")

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		fmt.Printf("✅ SUCCESS: password123 matches the hash\n")
	} else {
		fmt.Printf("❌ FAILED: password123 does NOT match the hash\nError: %v\n", err)

		// Generate a fresh hash for verification
		fmt.Printf("\nGenerating fresh hash for password123:\n")
		freshHash, genErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if genErr == nil {
			fmt.Printf("Fresh hash: %s\n", string(freshHash))

			// Test the fresh hash
			testErr := bcrypt.CompareHashAndPassword(freshHash, []byte(password))
			if testErr == nil {
				fmt.Printf("✅ Fresh hash validates correctly\n")
				fmt.Printf("\nUse this hash in your database:\n%s\n", string(freshHash))
			} else {
				fmt.Printf("❌ Fresh hash validation failed: %v\n", testErr)
			}
		}
	}
}
