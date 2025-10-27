package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Generate fresh hash for password123
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Fresh hash: %s\n", string(hash))

	// Test that it works
	err = bcrypt.CompareHashAndPassword(hash, []byte("password123"))
	if err == nil {
		fmt.Printf("✅ Hash verification successful!\n")
	} else {
		fmt.Printf("❌ Hash verification failed: %v\n", err)
	}
}
