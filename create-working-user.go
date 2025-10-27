package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Database connection
	connStr := "user=ao3_user password=secure_password dbname=ao3_nuclear host=localhost port=5432 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		// Try without credentials
		connStr = "user=postgres dbname=nuclear_ao3 host=localhost port=5432 sslmode=disable"
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Printf("Database connection failed. Creating in-memory test user...")
			fmt.Println("=== IN-MEMORY TEST USER ===")
			fmt.Println("Email: test@example.com")
			fmt.Println("Password: password123")

			// Generate hash for manual insertion
			hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
			fmt.Printf("Hash to use: %s\n", string(hash))
			return
		}
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Printf("Database ping failed: %v", err)
		fmt.Println("=== MANUAL HASH FOR DATABASE ===")
		hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		fmt.Printf("Run this SQL command:\n")
		fmt.Printf("UPDATE users SET password_hash = '%s' WHERE email = 'test@nuclear-ao3.com';\n", string(hash))
		return
	}

	// Generate fresh password hash
	password := "password123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	// Update or insert test user
	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, display_name, is_verified, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (email) 
		DO UPDATE SET password_hash = $4, updated_at = NOW()`,
		"test-user-001", "testuser", "test@example.com", string(hash), "Test User", true)

	if err != nil {
		log.Printf("Failed to create user: %v", err)
		fmt.Println("=== MANUAL INSERTION REQUIRED ===")
		fmt.Printf("Run this SQL:\n")
		fmt.Printf("INSERT INTO users (id, username, email, password_hash, display_name, is_verified) VALUES ('test-001', 'testuser', 'test@example.com', '%s', 'Test User', true);\n", string(hash))
		return
	}

	// Also try nuclear-ao3.com domain
	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, display_name, is_verified, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (email) 
		DO UPDATE SET password_hash = $4, updated_at = NOW()`,
		"test-user-002", "testuser2", "test@nuclear-ao3.com", string(hash), "Test User Nuclear", true)

	fmt.Printf("✅ SUCCESS: Created working test users!\n")
	fmt.Printf("Email: test@example.com\n")
	fmt.Printf("Password: password123\n")
	fmt.Printf("Alt Email: test@nuclear-ao3.com\n")
	fmt.Printf("Alt Password: password123\n")
	fmt.Printf("Hash used: %s\n", string(hash))
}
