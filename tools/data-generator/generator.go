package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Configuration for data generation
type Config struct {
	Scale         string
	Users         int
	Works         int
	Tags          int
	Fandoms       int
	Characters    int
	Relationships int
	Comments      int
	Kudos         int
	Bookmarks     int
	DatabaseURL   string
	Output        string
	Realistic     bool
	Verbose       bool
}

// Predefined data for realistic generation
var (
	popularFandoms = []FandomData{
		{"Marvel Cinematic Universe", "movie", "Superhero franchise"},
		{"Harry Potter - J. K. Rowling", "book", "Wizarding world series"},
		{"Sherlock Holmes & Related Fandoms", "tv", "Detective stories"},
		{"방탄소년단 | Bangtan Boys | BTS", "music", "K-pop group"},
		{"僕のヒーローアカデミア | Boku no Hero Academia | My Hero Academia", "anime", "Superhero academy anime"},
		{"Naruto", "anime", "Ninja anime series"},
		{"Teen Wolf (TV)", "tv", "Supernatural teen drama"},
		{"Supernatural", "tv", "Brothers hunting monsters"},
		{"The Witcher (TV)", "tv", "Fantasy monster hunter"},
		{"原神 | Genshin Impact (Video Game)", "game", "Open-world RPG"},

		// Secondary tier
		{"Star Wars", "movie", "Space opera franchise"},
		{"Attack on Titan", "anime", "Titan-fighting anime"},
		{"Dream SMP", "other", "Minecraft roleplay server"},
		{"Haikyuu!!", "anime", "Volleyball sports anime"},
		{"Good Omens", "tv", "Angel and demon friendship"},
		{"The Mandalorian", "tv", "Star Wars bounty hunter series"},
		{"Minecraft (Video Game)", "game", "Block building game"},
		{"Hermitcraft SMP", "other", "Minecraft server"},
		{"Critical Role (Web Series)", "podcast", "D&D actual play"},
		{"Hollow Knight (Video Game)", "game", "Indie metroidvania"},
	}

	characterNames = []string{
		"Tony Stark", "Steve Rogers", "Natasha Romanoff", "Clint Barton", "Bruce Banner", "Thor",
		"Harry Potter", "Hermione Granger", "Ron Weasley", "Draco Malfoy", "Severus Snape",
		"Sherlock Holmes", "John Watson", "Mycroft Holmes", "Jim Moriarty",
		"김남준 | Kim Namjoon | RM", "김석진 | Kim Seokjin | Jin", "민윤기 | Min Yoongi | Suga",
		"정호석 | Jung Hoseok | J-Hope", "박지민 | Park Jimin", "김태형 | Kim Taehyung | V", "전정국 | Jeon Jungkook",
		"緑谷出久 | Midoriya Izuku", "爆豪勝己 | Bakugou Katsuki", "轟焦凍 | Todoroki Shouto",
		"うずまきナルト | Uzumaki Naruto", "うちはサスケ | Uchiha Sasuke", "春野サクラ | Haruno Sakura",
		"Derek Hale", "Stiles Stilinski", "Scott McCall", "Lydia Martin",
		"Dean Winchester", "Sam Winchester", "Castiel", "Jack Kline",
		"Geralt of Rivia", "Jaskier | Dandelion", "Yennefer of Vengerberg", "Ciri",
	}

	workTitles = []string{
		"The {adjective} {noun}",
		"{Character} and the {noun} of {noun}",
		"A {adjective} {noun}",
		"{number} Ways to {verb}",
		"In the {noun} of the {noun}",
		"When {character} {verb}",
		"The {noun} Chronicles",
		"{adjective} {noun}, {adjective} {noun}",
		"Beyond the {noun}",
		"Tales from {place}",
	}

	adjectives = []string{"Bright", "Dark", "Hidden", "Lost", "Secret", "Ancient", "Forbidden", "Golden", "Silver", "Crimson", "Shadow", "Crystal", "Eternal", "Broken", "Sacred", "Wild", "Silent", "Distant", "Frozen", "Burning"}
	nouns      = []string{"Heart", "Star", "Moon", "Storm", "Fire", "Ice", "Dream", "Memory", "Path", "Journey", "Battle", "War", "Peace", "Love", "Hope", "Fear", "Light", "Darkness", "Magic", "Power"}
	verbs      = []string{"Love", "Fight", "Dance", "Sing", "Dream", "Hope", "Fear", "Run", "Hide", "Seek", "Find", "Save", "Protect", "Destroy", "Create", "Build", "Break", "Heal", "Hurt", "Remember"}
	places     = []string{"Hogwarts", "Asgard", "Wakanda", "Gotham", "Metropolis", "Starfleet", "Middle Earth", "Narnia", "Wonderland", "Neverland"}

	tagTypes   = []string{"fandom", "character", "relationship", "freeform"}
	ratings    = []string{"General Audiences", "Teen And Up Audiences", "Mature", "Explicit", "Not Rated"}
	categories = []string{"Gen", "F/M", "M/M", "F/F", "Multi", "Other"}
	warnings   = []string{"Creator Chose Not To Use Archive Warnings", "No Archive Warnings Apply", "Graphic Depictions Of Violence", "Major Character Death", "Underage", "Rape/Non-Con"}
)

type FandomData struct {
	Name        string
	Type        string
	Description string
}

func main() {
	var config Config

	// Command line flags
	flag.StringVar(&config.Scale, "scale", "small", "Data scale: small, medium, large, stress")
	flag.IntVar(&config.Users, "users", 0, "Number of users (overrides scale)")
	flag.IntVar(&config.Works, "works", 0, "Number of works (overrides scale)")
	flag.IntVar(&config.Tags, "tags", 0, "Number of tags (overrides scale)")
	flag.StringVar(&config.DatabaseURL, "db", "postgres://ao3_user:ao3_password@localhost:5432/ao3_nuclear?sslmode=disable", "Database connection string")
	flag.StringVar(&config.Output, "output", "", "Output SQL file (if specified, writes to file instead of database)")
	flag.BoolVar(&config.Realistic, "realistic", true, "Use realistic data patterns")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose output")
	flag.Parse()

	// Set scale-based defaults
	setScaleDefaults(&config)

	if config.Verbose {
		fmt.Printf("Generating data with config: %+v\n", config)
	}

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Generate data
	generator := &DataGenerator{
		config:  config,
		users:   make([]uuid.UUID, 0),
		fandoms: make([]uuid.UUID, 0),
		tags:    make([]uuid.UUID, 0),
		works:   make([]uuid.UUID, 0),
	}

	var db *sql.DB
	var err error
	var outputFile *os.File

	if config.Output != "" {
		outputFile, err = os.Create(config.Output)
		if err != nil {
			log.Fatal("Failed to create output file:", err)
		}
		defer outputFile.Close()
		generator.output = outputFile
	} else {
		db, err = sql.Open("postgres", config.DatabaseURL)
		if err != nil {
			log.Fatal("Failed to connect to database:", err)
		}
		defer db.Close()
		generator.db = db
	}

	// Generate all data
	fmt.Println("🚀 Starting Nuclear AO3 data generation...")

	if err := generator.generateUsers(); err != nil {
		log.Fatal("Failed to generate users:", err)
	}

	if err := generator.generateTags(); err != nil {
		log.Fatal("Failed to generate tags:", err)
	}

	if err := generator.generateWorks(); err != nil {
		log.Fatal("Failed to generate works:", err)
	}

	if err := generator.generateInteractions(); err != nil {
		log.Fatal("Failed to generate interactions:", err)
	}

	fmt.Printf("✅ Successfully generated:\n")
	fmt.Printf("   👥 %d users\n", len(generator.users))
	fmt.Printf("   📚 %d works\n", len(generator.works))
	fmt.Printf("   🏷️  %d tags\n", len(generator.tags))
	fmt.Printf("   🌟 %d fandoms\n", len(generator.fandoms))

	if config.Output != "" {
		fmt.Printf("   💾 Output written to: %s\n", config.Output)
	} else {
		fmt.Printf("   🗄️  Data inserted into database\n")
	}
}

type DataGenerator struct {
	config  Config
	db      *sql.DB
	output  *os.File
	users   []uuid.UUID
	fandoms []uuid.UUID
	tags    []uuid.UUID
	works   []uuid.UUID
}

func setScaleDefaults(config *Config) {
	if config.Users == 0 || config.Works == 0 || config.Tags == 0 {
		switch config.Scale {
		case "small":
			if config.Users == 0 {
				config.Users = 100
			}
			if config.Works == 0 {
				config.Works = 1000
			}
			if config.Tags == 0 {
				config.Tags = 500
			}
		case "medium":
			if config.Users == 0 {
				config.Users = 1000
			}
			if config.Works == 0 {
				config.Works = 10000
			}
			if config.Tags == 0 {
				config.Tags = 2000
			}
		case "large":
			if config.Users == 0 {
				config.Users = 10000
			}
			if config.Works == 0 {
				config.Works = 100000
			}
			if config.Tags == 0 {
				config.Tags = 10000
			}
		case "stress":
			if config.Users == 0 {
				config.Users = 100000
			}
			if config.Works == 0 {
				config.Works = 1000000
			}
			if config.Tags == 0 {
				config.Tags = 50000
			}
		}
	}

	// Set derived values
	config.Fandoms = min(len(popularFandoms), config.Tags/10)
	config.Characters = config.Tags / 4
	config.Relationships = config.Tags / 5
	config.Comments = config.Works * 3  // Average 3 comments per work
	config.Kudos = config.Works * 5     // Average 5 kudos per work
	config.Bookmarks = config.Works / 2 // Half of works bookmarked
}

func (g *DataGenerator) execute(query string, args ...interface{}) error {
	if g.output != nil {
		// Write to file
		if len(args) > 0 {
			// Format the query with arguments (simplified)
			_, err := fmt.Fprintf(g.output, "%s;\n", query)
			return err
		} else {
			_, err := fmt.Fprintf(g.output, "%s;\n", query)
			return err
		}
	} else {
		// Execute on database
		_, err := g.db.Exec(query, args...)
		return err
	}
}

func (g *DataGenerator) generateUsers() error {
	fmt.Printf("👥 Generating %d users...\n", g.config.Users)

	usernames := generateUsernames(g.config.Users)

	for i := 0; i < g.config.Users; i++ {
		userID := uuid.New()
		g.users = append(g.users, userID)

		username := usernames[i]
		email := fmt.Sprintf("%s@example.com", username)
		displayName := generateDisplayName(username)

		query := `INSERT INTO users (id, username, email, display_name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`
		err := g.execute(query, userID, username, email, displayName, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert user %s: %w", username, err)
		}
	}

	return nil
}

func (g *DataGenerator) generateTags() error {
	fmt.Printf("🏷️  Generating %d tags...\n", g.config.Tags)

	// Generate fandoms first
	for i, fandom := range popularFandoms {
		if i >= g.config.Fandoms {
			break
		}

		tagID := uuid.New()
		g.fandoms = append(g.fandoms, tagID)
		g.tags = append(g.tags, tagID)

		query := `INSERT INTO tags (id, name, type, canonical, is_canonical, is_filterable, description, use_count, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
		err := g.execute(query, tagID, fandom.Name, "fandom", true, true, true, fandom.Description, rand.Intn(1000)+100, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert fandom %s: %w", fandom.Name, err)
		}
	}

	// Generate characters
	charactersToGenerate := min(g.config.Characters, len(characterNames))
	for i := 0; i < charactersToGenerate; i++ {
		tagID := uuid.New()
		g.tags = append(g.tags, tagID)

		name := characterNames[i%len(characterNames)]
		if i >= len(characterNames) {
			name = fmt.Sprintf("%s %d", name, i/len(characterNames))
		}

		query := `INSERT INTO tags (id, name, type, canonical, is_canonical, is_filterable, use_count, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
		err := g.execute(query, tagID, name, "character", true, true, true, rand.Intn(500)+50, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert character %s: %w", name, err)
		}
	}

	// Generate relationships
	for i := 0; i < g.config.Relationships; i++ {
		tagID := uuid.New()
		g.tags = append(g.tags, tagID)

		char1 := characterNames[rand.Intn(len(characterNames))]
		char2 := characterNames[rand.Intn(len(characterNames))]
		relationshipType := "/"
		if rand.Float32() < 0.3 {
			relationshipType = " & "
		}
		name := fmt.Sprintf("%s%s%s", char1, relationshipType, char2)

		query := `INSERT INTO tags (id, name, type, canonical, is_canonical, is_filterable, use_count, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
		err := g.execute(query, tagID, name, "relationship", true, true, true, rand.Intn(300)+25, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert relationship %s: %w", name, err)
		}
	}

	// Generate freeform tags
	freeformTags := []string{
		"Alternate Universe", "Fluff", "Angst", "Hurt/Comfort", "Friends to Lovers", "Enemies to Lovers",
		"Slow Burn", "First Kiss", "Domestic Fluff", "Alternate Universe - Modern Setting",
		"Alternate Universe - Coffee Shops & Cafés", "Alternate Universe - College/University",
		"Protective", "Jealousy", "Mutual Pining", "Getting Together", "Established Relationship",
		"Family", "Friendship", "Romance", "Adventure", "Mystery", "Horror", "Humor",
		"Drama", "Action/Adventure", "Slice of Life", "Fix-It", "Time Travel", "Soulmates",
		"Fake/Pretend Relationship", "Only One Bed", "Sharing a Bed", "Cuddling & Snuggling",
		"Hand-Holding", "Kissing", "Making Out", "Sexual Content", "Explicit Sexual Content",
		"BDSM", "Dom/sub", "Alpha/Beta/Omega Dynamics", "Mpreg", "Trans Character",
		"LGBTQ Themes", "Coming Out", "Identity Issues", "Mental Health Issues",
		"Depression", "Anxiety", "PTSD", "Therapy", "Healing", "Recovery",
	}

	remainingTags := g.config.Tags - len(g.tags)
	for i := 0; i < remainingTags; i++ {
		tagID := uuid.New()
		g.tags = append(g.tags, tagID)

		name := freeformTags[i%len(freeformTags)]
		if i >= len(freeformTags) {
			name = fmt.Sprintf("%s %d", name, i/len(freeformTags)+1)
		}

		query := `INSERT INTO tags (id, name, type, canonical, is_canonical, is_filterable, use_count, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
		err := g.execute(query, tagID, name, "freeform", rand.Float32() < 0.7, true, true, rand.Intn(200)+10, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert freeform tag %s: %w", name, err)
		}
	}

	return nil
}

func (g *DataGenerator) generateWorks() error {
	fmt.Printf("📚 Generating %d works...\n", g.config.Works)

	for i := 0; i < g.config.Works; i++ {
		workID := uuid.New()
		g.works = append(g.works, workID)

		title := generateWorkTitle()
		summary := generateSummary()
		authorID := g.users[rand.Intn(len(g.users))]

		// Realistic distributions
		rating := ratings[rand.Intn(len(ratings))]
		category := categories[rand.Intn(len(categories))]
		warning := warnings[rand.Intn(len(warnings))]

		// Word count following realistic distribution (most works are short)
		wordCount := generateRealisticWordCount()

		// Chapter count
		chapterCount := 1
		if wordCount > 5000 {
			chapterCount = rand.Intn(20) + 1
		}

		isComplete := rand.Float32() < 0.6 // 60% of works are complete
		isDraft := rand.Float32() < 0.05   // 5% are drafts

		var publishedAt *time.Time
		if !isDraft {
			pub := time.Now().AddDate(0, 0, -rand.Intn(365*3)) // Published in last 3 years
			publishedAt = &pub
		}

		query := `INSERT INTO works (id, title, summary, user_id, rating, category, warnings, word_count, chapter_count, is_complete, is_draft, published_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
		err := g.execute(query, workID, title, summary, authorID, rating, category, warning, wordCount, chapterCount, isComplete, isDraft, publishedAt, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert work %s: %w", title, err)
		}

		// Add work statistics
		statsQuery := `INSERT INTO work_statistics (work_id, hits, kudos, comments, bookmarks) VALUES ($1, $2, $3, $4, $5)`
		hits := rand.Intn(1000) + 10
		kudos := rand.Intn(hits / 2)
		comments := rand.Intn(kudos / 2)
		bookmarks := rand.Intn(kudos / 3)

		err = g.execute(statsQuery, workID, hits, kudos, comments, bookmarks)
		if err != nil {
			return fmt.Errorf("failed to insert work statistics for %s: %w", title, err)
		}

		// Add tags (2-8 tags per work)
		numTags := rand.Intn(7) + 2
		usedTags := make(map[uuid.UUID]bool)

		// Always add a fandom
		if len(g.fandoms) > 0 {
			fandomID := g.fandoms[rand.Intn(len(g.fandoms))]
			tagQuery := `INSERT INTO work_tags (work_id, tag_id) VALUES ($1, $2)`
			err = g.execute(tagQuery, workID, fandomID)
			if err != nil {
				return fmt.Errorf("failed to insert work tag: %w", err)
			}
			usedTags[fandomID] = true
		}

		// Add other tags
		for len(usedTags) < numTags && len(usedTags) < len(g.tags) {
			tagID := g.tags[rand.Intn(len(g.tags))]
			if !usedTags[tagID] {
				tagQuery := `INSERT INTO work_tags (work_id, tag_id) VALUES ($1, $2)`
				err = g.execute(tagQuery, workID, tagID)
				if err != nil {
					return fmt.Errorf("failed to insert work tag: %w", err)
				}
				usedTags[tagID] = true
			}
		}
	}

	return nil
}

func (g *DataGenerator) generateInteractions() error {
	fmt.Printf("💬 Generating interactions (kudos, comments, bookmarks)...\n")

	// Generate some kudos
	numKudos := min(g.config.Kudos, len(g.users)*len(g.works)/10)
	for i := 0; i < numKudos; i++ {
		workID := g.works[rand.Intn(len(g.works))]
		userID := g.users[rand.Intn(len(g.users))]

		query := `INSERT INTO kudos (work_id, user_id, created_at) VALUES ($1, $2, $3) ON CONFLICT (work_id, user_id) DO NOTHING`
		err := g.execute(query, workID, userID, time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert kudos: %w", err)
		}
	}

	// Generate some comments
	numComments := min(g.config.Comments, len(g.users)*len(g.works)/20)
	for i := 0; i < numComments; i++ {
		workID := g.works[rand.Intn(len(g.works))]
		userID := g.users[rand.Intn(len(g.users))]
		content := generateComment()

		query := `INSERT INTO comments (work_id, user_id, content, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`
		err := g.execute(query, workID, userID, content, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert comment: %w", err)
		}
	}

	// Generate some bookmarks
	numBookmarks := min(g.config.Bookmarks, len(g.users)*len(g.works)/30)
	for i := 0; i < numBookmarks; i++ {
		workID := g.works[rand.Intn(len(g.works))]
		userID := g.users[rand.Intn(len(g.users))]

		query := `INSERT INTO bookmarks (work_id, user_id, created_at, updated_at) VALUES ($1, $2, $3, $4) ON CONFLICT (work_id, user_id) DO NOTHING`
		err := g.execute(query, workID, userID, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert bookmark: %w", err)
		}
	}

	return nil
}

// Helper functions
func generateUsernames(count int) []string {
	prefixes := []string{"fan", "reader", "writer", "dreamer", "star", "moon", "sun", "fire", "ice", "storm", "shadow", "light", "dark", "bright", "swift", "wild", "free", "brave", "kind", "wise"}
	suffixes := []string{"lover", "writer", "reader", "dreamer", "seeker", "finder", "keeper", "guardian", "warrior", "mage", "scholar", "artist", "poet", "dancer", "singer", "storyteller", "wanderer", "explorer", "hero", "legend"}

	usernames := make([]string, count)
	for i := 0; i < count; i++ {
		prefix := prefixes[rand.Intn(len(prefixes))]
		suffix := suffixes[rand.Intn(len(suffixes))]
		number := rand.Intn(9999) + 1
		usernames[i] = fmt.Sprintf("%s%s%d", prefix, suffix, number)
	}
	return usernames
}

func generateDisplayName(username string) string {
	if rand.Float32() < 0.7 {
		return username
	}
	// Sometimes use a different display name
	adjective := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	return fmt.Sprintf("%s %s", adjective, noun)
}

func generateWorkTitle() string {
	template := workTitles[rand.Intn(len(workTitles))]

	// Replace placeholders
	title := strings.ReplaceAll(template, "{adjective}", adjectives[rand.Intn(len(adjectives))])
	title = strings.ReplaceAll(title, "{noun}", nouns[rand.Intn(len(nouns))])
	title = strings.ReplaceAll(title, "{verb}", verbs[rand.Intn(len(verbs))])
	title = strings.ReplaceAll(title, "{character}", characterNames[rand.Intn(len(characterNames))])
	title = strings.ReplaceAll(title, "{place}", places[rand.Intn(len(places))])
	title = strings.ReplaceAll(title, "{number}", fmt.Sprintf("%d", rand.Intn(10)+1))

	return title
}

func generateSummary() string {
	templates := []string{
		"A story about love, loss, and finding hope in the darkness.",
		"When {character} meets {character}, nothing will ever be the same.",
		"Five times they almost kissed, and one time they did.",
		"An exploration of {adjective} relationships and {adjective} emotions.",
		"What happens when everything you thought you knew turns out to be wrong?",
		"A {adjective} tale of {noun} and {noun}.",
		"Sometimes the best things come from the most unexpected places.",
		"A collection of moments that matter.",
	}

	template := templates[rand.Intn(len(templates))]
	summary := strings.ReplaceAll(template, "{character}", characterNames[rand.Intn(len(characterNames))])
	summary = strings.ReplaceAll(summary, "{adjective}", adjectives[rand.Intn(len(adjectives))])
	summary = strings.ReplaceAll(summary, "{noun}", nouns[rand.Intn(len(nouns))])

	return summary
}

func generateRealisticWordCount() int {
	// Realistic distribution: most works are short
	r := rand.Float32()
	if r < 0.4 {
		return rand.Intn(2000) + 500 // Short works: 500-2500 words
	} else if r < 0.7 {
		return rand.Intn(8000) + 2500 // Medium works: 2.5-10K words
	} else if r < 0.9 {
		return rand.Intn(25000) + 10000 // Long works: 10-35K words
	} else {
		return rand.Intn(200000) + 50000 // Epic works: 50-250K words
	}
}

func generateComment() string {
	comments := []string{
		"This is amazing! I love how you wrote this.",
		"Please update soon! I'm dying to know what happens next.",
		"This made me cry happy tears.",
		"The characterization is spot on.",
		"I stayed up way too late reading this.",
		"This is now my headcanon.",
		"I love the way you write dialogue.",
		"This hit me right in the feels.",
		"I need more of this in my life.",
		"This is exactly what I needed today.",
		"The plot twist got me good!",
		"I'm obsessed with this fic.",
		"Your writing style is beautiful.",
		"This deserves so much more attention.",
		"I'm going to reread this immediately.",
	}
	return comments[rand.Intn(len(comments))]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
