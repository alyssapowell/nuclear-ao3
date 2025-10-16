package main

import (
	"database/sql"
	"fmt"
	"log"
)

// SchemaValidator ensures the database schema matches code expectations
type SchemaValidator struct {
	db *sql.DB
}

// ExpectedColumn represents what the code expects from the database
type ExpectedColumn struct {
	Name         string
	DataType     string
	IsNullable   bool
	DefaultValue *string
}

// WorksTableSchema defines the expected schema for the works table
var WorksTableSchema = []ExpectedColumn{
	{Name: "id", DataType: "uuid", IsNullable: false, DefaultValue: stringPtr("uuid_generate_v4()")},
	{Name: "title", DataType: "character varying", IsNullable: false, DefaultValue: nil},
	{Name: "summary", DataType: "text", IsNullable: true, DefaultValue: nil},
	{Name: "user_id", DataType: "uuid", IsNullable: false, DefaultValue: nil},
	{Name: "language", DataType: "character varying", IsNullable: false, DefaultValue: stringPtr("'en'::character varying")},
	{Name: "rating", DataType: "character varying", IsNullable: false, DefaultValue: stringPtr("'Not Rated'::character varying")},
	{Name: "word_count", DataType: "integer", IsNullable: false, DefaultValue: stringPtr("0")},
	{Name: "chapter_count", DataType: "integer", IsNullable: false, DefaultValue: stringPtr("1")},
	{Name: "is_complete", DataType: "boolean", IsNullable: true, DefaultValue: stringPtr("false")},
	{Name: "is_draft", DataType: "boolean", IsNullable: true, DefaultValue: stringPtr("true")},
	{Name: "restricted", DataType: "boolean", IsNullable: true, DefaultValue: stringPtr("false")},
	{Name: "published_at", DataType: "timestamp with time zone", IsNullable: true, DefaultValue: nil},
	{Name: "created_at", DataType: "timestamp with time zone", IsNullable: true, DefaultValue: stringPtr("now()")},
	{Name: "updated_at", DataType: "timestamp with time zone", IsNullable: true, DefaultValue: stringPtr("now()")},
}

func stringPtr(s string) *string {
	return &s
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator(db *sql.DB) *SchemaValidator {
	return &SchemaValidator{db: db}
}

// ValidateWorksTable validates that the works table matches expectations
func (sv *SchemaValidator) ValidateWorksTable() error {
	log.Println("🔍 Validating works table schema...")

	// Get actual schema from database
	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns 
		WHERE table_name = 'works' AND table_schema = 'public'
		ORDER BY ordinal_position
	`

	rows, err := sv.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query works table schema: %w", err)
	}
	defer rows.Close()

	actualColumns := make(map[string]ExpectedColumn)
	for rows.Next() {
		var name, dataType, isNullable string
		var defaultValue sql.NullString

		if err := rows.Scan(&name, &dataType, &isNullable, &defaultValue); err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}

		var defaultPtr *string
		if defaultValue.Valid {
			defaultPtr = &defaultValue.String
		}

		actualColumns[name] = ExpectedColumn{
			Name:         name,
			DataType:     dataType,
			IsNullable:   isNullable == "YES",
			DefaultValue: defaultPtr,
		}
	}

	// Validate required columns exist
	var errors []string
	for _, expected := range WorksTableSchema {
		actual, exists := actualColumns[expected.Name]
		if !exists {
			errors = append(errors, fmt.Sprintf("❌ Missing required column: %s", expected.Name))
			continue
		}

		// Check data type compatibility
		if !isCompatibleType(expected.DataType, actual.DataType) {
			errors = append(errors, fmt.Sprintf("❌ Column %s: expected type %s, got %s",
				expected.Name, expected.DataType, actual.DataType))
		}

		log.Printf("✅ Column %s: %s", expected.Name, actual.DataType)
	}

	// Check for deprecated columns that code might still reference
	deprecatedColumns := []string{"status", "restricted_to_users", "warnings"}
	for _, deprecated := range deprecatedColumns {
		if _, exists := actualColumns[deprecated]; exists {
			log.Printf("⚠️  Found deprecated column: %s (consider removing from schema)", deprecated)
		}
	}

	if len(errors) > 0 {
		errorMsg := "❌ Works table schema validation failed:\n"
		for _, err := range errors {
			errorMsg += "  " + err + "\n"
		}
		return fmt.Errorf(errorMsg)
	}

	log.Println("✅ Works table schema validation passed")
	return nil
}

// isCompatibleType checks if database type is compatible with expected type
func isCompatibleType(expected, actual string) bool {
	// Handle PostgreSQL type variations
	typeMap := map[string][]string{
		"character varying":        {"character varying", "varchar", "text"},
		"text":                     {"text", "character varying"},
		"uuid":                     {"uuid"},
		"integer":                  {"integer", "int", "int4"},
		"boolean":                  {"boolean", "bool"},
		"timestamp with time zone": {"timestamp with time zone", "timestamptz"},
	}

	compatibleTypes, exists := typeMap[expected]
	if !exists {
		return expected == actual
	}

	for _, compatible := range compatibleTypes {
		if actual == compatible {
			return true
		}
	}
	return false
}

// ValidateAllSchemas validates all critical table schemas
func (sv *SchemaValidator) ValidateAllSchemas() error {
	log.Println("🔧 Starting comprehensive schema validation...")

	// Validate works table (most critical)
	if err := sv.ValidateWorksTable(); err != nil {
		return fmt.Errorf("works table validation failed: %w", err)
	}

	// Validate tags table exists and has basic structure
	if err := sv.ValidateTableExists("tags"); err != nil {
		return fmt.Errorf("tags table validation failed: %w", err)
	}

	// Validate users table exists and has basic structure
	if err := sv.ValidateTableExists("users"); err != nil {
		return fmt.Errorf("users table validation failed: %w", err)
	}

	// Validate work_tags relationship table exists
	if err := sv.ValidateTableExists("work_tags"); err != nil {
		return fmt.Errorf("work_tags table validation failed: %w", err)
	}

	// Validate chapters table exists
	if err := sv.ValidateTableExists("chapters"); err != nil {
		return fmt.Errorf("chapters table validation failed: %w", err)
	}

	log.Println("🎉 All schema validations passed!")
	return nil
}

// ValidateTableExists checks if a critical table exists
func (sv *SchemaValidator) ValidateTableExists(tableName string) error {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = $1
		)`

	err := sv.db.QueryRow(query, tableName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if table %s exists: %w", tableName, err)
	}

	if !exists {
		return fmt.Errorf("❌ Critical table '%s' does not exist", tableName)
	}

	log.Printf("✅ Table %s exists", tableName)
	return nil
}
