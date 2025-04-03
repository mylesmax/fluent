package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var (
	db   *sql.DB
	once sync.Once
)

// id,name,emoji,glow, uuid, etc., tbd
// cur drops
type Profile struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Emoji        string   `json:"emoji"`
	GlowColor    string   `json:"glowColor"`
	CurrentDrops int      `json:"currentDrops"`
	ClassDBPath  string   `json:"classDBPath"`
	ClassUUID    string   `json:"classUUID"`
	IsAddNew     bool     `json:"isAddNew"`
	Active       bool     `json:"active"`
	NameHistory  []string `json:"nameHistory"`
}

//for logging prices + ai interactions
type AIHistoryEntry struct {
	ID        int       `json:"id"`
	ClassUUID string    `json:"classUUID"`
	SessionID string    `json:"sessionID"`
	Prompt    string    `json:"prompt"`
	Response  string    `json:"response"`
	Tokens    int       `json:"tokens"`
	Cost      float64   `json:"cost"`
	Timestamp time.Time `json:"timestamp"`
}

type SessionData struct {
	ID             string          `json:"id"`
	ClassUUID      string          `json:"classUUID"`
	UploadPrompt   string          `json:"uploadPrompt"`
	ChatHistory    json.RawMessage `json:"chatHistory"`
	StartTimestamp time.Time       `json:"startTimestamp"`
	EndTimestamp   *time.Time      `json:"endTimestamp"`
}

// sep into user data/styling, class dbs for droplets, trash can
func ensureDataDirs() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	dirs := []string{
		filepath.Join(homeDir, ".fluent"),
		filepath.Join(homeDir, ".fluent/user"),
		filepath.Join(homeDir, ".fluent/class"),
		filepath.Join(homeDir, ".fluent/trash"),
		filepath.Join(homeDir, ".fluent/aiHistory"), // New directory for AI history
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}
	return nil
}

func createClassDB(classDBPath string, classUUID string, initialName string) error {
	classDB, err := sql.Open("sqlite3", classDBPath)
	if err != nil {
		return err
	}
	defer classDB.Close()

	_, err = classDB.Exec(`
		CREATE TABLE IF NOT EXISTS class_info (
			uuid TEXT PRIMARY KEY,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			name_history TEXT NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	_, err = classDB.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			upload_prompt TEXT NOT NULL,
			chat_history TEXT NOT NULL,
			start_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			end_timestamp TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create sessions table: %v", err)
	}

	_, err = classDB.Exec(`
		CREATE TABLE IF NOT EXISTS factoids (
			id TEXT PRIMARY KEY,
			session_id TEXT,
			question TEXT NOT NULL,
			answer TEXT NOT NULL,
			type TEXT,
			verbatim TEXT,
			context TEXT,
			requires_clarification INTEGER,
			alternative_subjects_count INTEGER,
			difficulty INTEGER,
			examples TEXT,
			last_review TIMESTAMP,
			next_review TIMESTAMP,
			stability REAL DEFAULT 1.0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create factoids table: %v", err)
	}

	nameHistory := []string{initialName}
	nameHistoryJSON, err := json.Marshal(nameHistory)
	if err != nil {
		return err
	}

	_, err = classDB.Exec(`
		INSERT OR REPLACE INTO class_info (uuid, name_history)
		VALUES (?, ?)
	`, classUUID, string(nameHistoryJSON))
	return err
}

func FixExistingClassDatabases() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	classDir := filepath.Join(homeDir, ".fluent", "class")
	files, err := os.ReadDir(classDir)
	if err != nil {
		return fmt.Errorf("failed to read class directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".db" {
			dbPath := filepath.Join(classDir, file.Name())
			log.Printf("Checking class database: %s", dbPath)

			classDB, err := sql.Open("sqlite3", dbPath)
			if err != nil {
				log.Printf("Error opening class database %s: %v", dbPath, err)
				continue
			}

			var tableName string
			err = classDB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='sessions'").Scan(&tableName)
			if err != nil {
				log.Printf("Creating missing sessions table in %s", dbPath)

				_, err = classDB.Exec(`
					CREATE TABLE IF NOT EXISTS sessions (
						id TEXT PRIMARY KEY,
						upload_prompt TEXT NOT NULL,
						chat_history TEXT NOT NULL,
						start_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						end_timestamp TIMESTAMP
					);
				`)
				if err != nil {
					log.Printf("Error creating sessions table for class %s: %v", file.Name(), err)
				} else {
					log.Printf("Successfully created sessions table in %s", dbPath)
				}
			}

			classDB.Close()
		}
	}

	return nil
}

func InitDB() (*sql.DB, error) {
	if err := ensureDataDirs(); err != nil {
		return nil, fmt.Errorf("failed to ensure data directories: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	dbPath := filepath.Join(homeDir, ".fluent", "profiles.db")
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			emoji TEXT,
			glow_color TEXT,
			current_drops INTEGER DEFAULT 0,
			class_db_path TEXT,
			class_uuid TEXT,
			is_add_new INTEGER DEFAULT 0,
			active INTEGER DEFAULT 1
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create profiles table: %v", err)
	}
	initAIHistoryDatabases()

	if err := initClassDatabases(); err != nil {
		log.Printf("Warning: Error initializing some class databases: %v", err)
	}

	if err := FixExistingClassDatabases(); err != nil {
		log.Printf("warning: error fixing some class databases: %v", err)
	}

	return db, err
}

func initClassDatabases() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	rows, err := db.Query("SELECT class_uuid FROM profiles WHERE class_uuid IS NOT NULL")
	if err != nil {
		return fmt.Errorf("failed to query class UUIDs: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var classUUID string
		if err := rows.Scan(&classUUID); err != nil {
			log.Printf("Error scanning class UUID: %v", err)
			continue
		}

		classDBPath := filepath.Join(homeDir, ".fluent", "class", classUUID+".db")
		classDB, err := sql.Open("sqlite3", classDBPath)
		if err != nil {
			log.Printf("Error opening class database %s: %v", classDBPath, err)
			continue
		}
		defer classDB.Close()

		_, err = classDB.Exec(`
			CREATE TABLE IF NOT EXISTS sessions (
				id TEXT PRIMARY KEY,
				upload_prompt TEXT NOT NULL,
				chat_history TEXT NOT NULL,
				start_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				end_timestamp TIMESTAMP
			);
		`)
		if err != nil {
			log.Printf("Error creating sessions table for class %s: %v", classUUID, err)
		}

		_, err = classDB.Exec(`
			CREATE TABLE IF NOT EXISTS factoids (
				id TEXT PRIMARY KEY,
				session_id TEXT,
				question TEXT NOT NULL,
				answer TEXT NOT NULL,
				type TEXT,
				verbatim TEXT,
				context TEXT,
				requires_clarification INTEGER,
				alternative_subjects_count INTEGER,
				difficulty INTEGER,
				examples TEXT,
				last_review TIMESTAMP,
				next_review TIMESTAMP,
				stability REAL DEFAULT 1.0,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)
		if err != nil {
			log.Printf("Error creating factoids table for class %s: %v", classUUID, err)
		}

		_, err = classDB.Exec(`
			CREATE TABLE IF NOT EXISTS class_info (
				uuid TEXT PRIMARY KEY,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				last_modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				name_history TEXT DEFAULT '[]'
			);
		`)
		if err != nil {
			log.Printf("Error creating class_info table for class %s: %v", classUUID, err)
		}
	}

	return nil
}

func initAIHistoryDatabases() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get home directory: %v", err)
		return
	}

	aiHistoryDBs := []string{
		filepath.Join(homeDir, ".fluent/aiHistory/uploads.db"),
		filepath.Join(homeDir, ".fluent/aiHistory/chats.db"),
		filepath.Join(homeDir, ".fluent/aiHistory/parser.db"),
	}

	for _, dbPath := range aiHistoryDBs {
		aiHistoryDB, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Printf("Failed to open AI history database %s: %v", dbPath, err)
			continue
		}

		_, err = aiHistoryDB.Exec(`
			CREATE TABLE IF NOT EXISTS ai_history (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				class_uuid TEXT NOT NULL,
				session_id TEXT NOT NULL,
				prompt TEXT NOT NULL,
				response TEXT,
				tokens INTEGER DEFAULT 0,
				cost REAL DEFAULT 0.0,
				timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)
		if err != nil {
			log.Printf("Failed to create AI history table in %s: %v", dbPath, err)
		}

		aiHistoryDB.Close()
	}
}

func GetProfiles() ([]Profile, error) {
	rows, err := db.Query(`
		SELECT id, name, emoji, glow_color, current_drops, class_db_path, class_uuid, is_add_new, active
		FROM profiles 
		WHERE is_add_new = 0 AND active = 1
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var p Profile
		err := rows.Scan(&p.ID, &p.Name, &p.Emoji, &p.GlowColor, &p.CurrentDrops, &p.ClassDBPath, &p.ClassUUID, &p.IsAddNew, &p.Active)
		if err != nil {
			return nil, err
		}

		if p.ClassDBPath != "" {
			classDB, err := sql.Open("sqlite3", p.ClassDBPath)
			if err == nil {
				var nameHistoryJSON string
				err = classDB.QueryRow("SELECT name_history FROM class_info WHERE uuid = ?", p.ClassUUID).Scan(&nameHistoryJSON)
				if err == nil {
					json.Unmarshal([]byte(nameHistoryJSON), &p.NameHistory)
				}
				classDB.Close()
			}
		}

		profiles = append(profiles, p)
	}
	return profiles, nil
}

func AddProfile(profile Profile) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	classUUID := uuid.New().String()
	classDBPath := filepath.Join(homeDir, ".fluent", "class", classUUID+".db")

	if err := createClassDB(classDBPath, classUUID, profile.Name); err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO profiles (name, emoji, glow_color, current_drops, class_db_path, class_uuid, is_add_new, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1)
	`, profile.Name, profile.Emoji, profile.GlowColor, profile.CurrentDrops, classDBPath, classUUID, profile.IsAddNew)
	return err
}

func UpdateProfile(profile Profile) error {
	_, err := db.Exec(`
		UPDATE profiles 
		SET name = ?, emoji = ?, glow_color = ?, current_drops = ?
		WHERE id = ?
	`, profile.Name, profile.Emoji, profile.GlowColor, profile.CurrentDrops, profile.ID)
	return err
}

func UpdateProfileName(oldName, newName string) error {
	var classDBPath, classUUID string
	err := db.QueryRow("SELECT class_db_path, class_uuid FROM profiles WHERE name = ?", oldName).Scan(&classDBPath, &classUUID)
	if err != nil {
		return err
	}

	if classDBPath != "" {
		classDB, err := sql.Open("sqlite3", classDBPath)
		if err == nil {
			defer classDB.Close()

			var nameHistoryJSON string
			err = classDB.QueryRow("SELECT name_history FROM class_info WHERE uuid = ?", classUUID).Scan(&nameHistoryJSON)
			if err == nil {
				var nameHistory []string
				json.Unmarshal([]byte(nameHistoryJSON), &nameHistory)
				nameHistory = append(nameHistory, newName)
				newNameHistoryJSON, _ := json.Marshal(nameHistory)

				_, err = classDB.Exec(`
					UPDATE class_info 
					SET name_history = ?, last_modified = CURRENT_TIMESTAMP
					WHERE uuid = ?
				`, string(newNameHistoryJSON), classUUID)
			}
		}
	}

	_, err = db.Exec(`
		UPDATE profiles 
		SET name = ?
		WHERE name = ?
	`, newName, oldName)
	return err
}

func DeleteProfile(name string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	var classDBPath, classUUID string
	err = db.QueryRow("SELECT class_db_path, class_uuid FROM profiles WHERE name = ?", name).Scan(&classDBPath, &classUUID)
	if err != nil {
		return err
	}

	if classDBPath != "" && classUUID != "" {
		trashPath := filepath.Join(homeDir, ".fluent", "trash", classUUID+".db")
		err = os.Rename(classDBPath, trashPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to move class DB to trash: %v", err)
		}
	}

	_, err = db.Exec(`
		UPDATE profiles 
		SET active = 0
		WHERE name = ?
	`, name)
	return err
}

func UpdateDrops(name string, drops int) error {
	_, err := db.Exec(`
		UPDATE profiles 
		SET current_drops = ?
		WHERE name = ?
	`, drops, name)
	return err
}

// try this lter, idgaf it works or not
func RestoreProfile(classUUID string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	trashPath := filepath.Join(homeDir, ".fluent", "trash", classUUID+".db")
	classDBPath := filepath.Join(homeDir, ".fluent", "class", classUUID+".db")

	err = os.Rename(trashPath, classDBPath)
	if err != nil {
		return fmt.Errorf("failed to restore class DB from trash: %v", err)
	}

	_, err = db.Exec(`
		UPDATE profiles 
		SET active = 1
		WHERE class_uuid = ?
	`, classUUID)
	return err
}

func UpdateProfileGlowColor(name string, newGlowColor string) error {
	_, err := db.Exec(`
		UPDATE profiles 
		SET glow_color = ?
		WHERE name = ?
	`, newGlowColor, name)
	return err
}

func CreateLearnSession(classUUID, uploadPrompt string, chatHistory json.RawMessage) (string, error) {
	sessionID := uuid.New().String()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %v", err)
	}

	classDBPath := filepath.Join(homeDir, ".fluent", "class", classUUID+".db")
	classDB, err := sql.Open("sqlite3", classDBPath)
	if err != nil {
		return "", fmt.Errorf("failed to open class database: %v", err)
	}
	defer classDB.Close()

	_, err = classDB.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			upload_prompt TEXT NOT NULL,
			chat_history TEXT NOT NULL,
			start_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			end_timestamp TIMESTAMP
		);
	`)
	if err != nil {
		return "", fmt.Errorf("failed to create sessions table: %v", err)
	}

	_, err = classDB.Exec(`
		INSERT INTO sessions (id, upload_prompt, chat_history)
		VALUES (?, ?, ?);
	`, sessionID, uploadPrompt, chatHistory)
	if err != nil {
		return "", fmt.Errorf("failed to insert session: %v", err)
	}

	err = RecordAIUploadHistory(classUUID, sessionID, uploadPrompt, "", 0, 0.0)
	if err != nil {
		log.Printf("Warning: Failed to record upload AI history: %v", err)
	}

	return sessionID, nil
}

func RecordAIUploadHistory(classUUID, sessionID, prompt, response string, tokens int, cost float64) error {
	return recordAIHistory("uploads.db", classUUID, sessionID, prompt, response, tokens, cost)
}

func RecordAIChatHistory(classUUID, sessionID, prompt, response string, tokens int, cost float64) error {
	return recordAIHistory("chats.db", classUUID, sessionID, prompt, response, tokens, cost)
}

func RecordAIParserHistory(classUUID, sessionID, prompt, response string, tokens int, cost float64) error {
	return recordAIHistory("parser.db", classUUID, sessionID, prompt, response, tokens, cost)
}

func recordAIHistory(dbName, classUUID, sessionID, prompt, response string, tokens int, cost float64) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	dbPath := filepath.Join(homeDir, ".fluent/aiHistory", dbName)
	aiHistoryDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open AI history database: %v", err)
	}
	defer aiHistoryDB.Close()

	_, err = aiHistoryDB.Exec(`
		INSERT INTO ai_history (class_uuid, session_id, prompt, response, tokens, cost)
		VALUES (?, ?, ?, ?, ?, ?);
	`, classUUID, sessionID, prompt, response, tokens, cost)

	return err
}

func GetSessionData(classUUID, sessionID string) (*SessionData, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	classDBPath := filepath.Join(homeDir, ".fluent", "class", classUUID+".db")
	classDB, err := sql.Open("sqlite3", classDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open class database: %v", err)
	}
	defer classDB.Close()

	var session SessionData
	var startTimestamp string
	var endTimestamp sql.NullString
	var chatHistoryStr string

	err = classDB.QueryRow(`
		SELECT id, upload_prompt, chat_history, start_timestamp, end_timestamp
		FROM sessions
		WHERE id = ?
	`, sessionID).Scan(&session.ID, &session.UploadPrompt, &chatHistoryStr, &startTimestamp, &endTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session: %v", err)
	}

	session.ClassUUID = classUUID
	session.ChatHistory = json.RawMessage(chatHistoryStr)
	session.StartTimestamp, _ = time.Parse(time.RFC3339, startTimestamp)

	if endTimestamp.Valid {
		endTime, _ := time.Parse(time.RFC3339, endTimestamp.String)
		session.EndTimestamp = &endTime
	}

	return &session, nil
}

func GetClassSessions(classUUID string) ([]SessionData, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	classDBPath := filepath.Join(homeDir, ".fluent", "class", classUUID+".db")
	classDB, err := sql.Open("sqlite3", classDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open class database: %v", err)
	}
	defer classDB.Close()

	var tableExists bool
	err = classDB.QueryRow(`
		SELECT COUNT(*) > 0 
		FROM sqlite_master 
		WHERE type='table' AND name='sessions'
	`).Scan(&tableExists)
	if err != nil {
		return nil, fmt.Errorf("failed to check if sessions table exists: %v", err)
	}

	if !tableExists {
		return []SessionData{}, nil
	}

	rows, err := classDB.Query(`
		SELECT id, upload_prompt, chat_history, start_timestamp, end_timestamp
		FROM sessions
		ORDER BY start_timestamp DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %v", err)
	}
	defer rows.Close()

	var sessions []SessionData
	for rows.Next() {
		var session SessionData
		var startTimestamp string
		var endTimestamp sql.NullString
		var chatHistoryStr string

		err := rows.Scan(&session.ID, &session.UploadPrompt, &chatHistoryStr, &startTimestamp, &endTimestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session row: %v", err)
		}

		session.ClassUUID = classUUID
		session.ChatHistory = json.RawMessage(chatHistoryStr)
		session.StartTimestamp, _ = time.Parse(time.RFC3339, startTimestamp)

		if endTimestamp.Valid {
			endTime, _ := time.Parse(time.RFC3339, endTimestamp.String)
			session.EndTimestamp = &endTime
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

func UpdateSessionChatHistory(classUUID, sessionID string, chatHistory json.RawMessage) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	classDBPath := filepath.Join(homeDir, ".fluent", "class", classUUID+".db")
	classDB, err := sql.Open("sqlite3", classDBPath)
	if err != nil {
		return fmt.Errorf("failed to open class database: %v", err)
	}
	defer classDB.Close()

	_, err = classDB.Exec(`
		UPDATE sessions
		SET chat_history = ?
		WHERE id = ?
	`, chatHistory, sessionID)

	return err
}

func EndSession(classUUID, sessionID string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	classDBPath := filepath.Join(homeDir, ".fluent", "class", classUUID+".db")
	classDB, err := sql.Open("sqlite3", classDBPath)
	if err != nil {
		return fmt.Errorf("failed to open class database: %v", err)
	}
	defer classDB.Close()

	_, err = classDB.Exec(`
		UPDATE sessions
		SET end_timestamp = CURRENT_TIMESTAMP
		WHERE id = ?
	`, sessionID)

	return err
}

func GetFactoid(classUUID, factoidID string) (FactoidData, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return FactoidData{}, fmt.Errorf("failed to get home directory: %v", err)
	}

	classDBPath := filepath.Join(homeDir, ".fluent", "class", classUUID+".db")
	classDB, err := sql.Open("sqlite3", classDBPath)
	if err != nil {
		return FactoidData{}, fmt.Errorf("failed to open class database: %v", err)
	}
	defer classDB.Close()

	var factoid FactoidData
	var examplesJSON string

	err = classDB.QueryRow(`
		SELECT id, question, answer, type, verbatim, context, requires_clarification, 
		alternative_subjects_count, difficulty, examples, stability, next_review
		FROM factoids WHERE id = ?
	`, factoidID).Scan(
		&factoid.ID,
		&factoid.Question,
		&factoid.Answer,
		&factoid.Type,
		&factoid.Verbatim,
		&factoid.Context,
		&factoid.RequiresClarification,
		&factoid.AlternativeSubjectsCount,
		&factoid.Difficulty,
		&examplesJSON,
		&factoid.Stability,
		&factoid.NextReview,
	)
	if err != nil {
		return FactoidData{}, fmt.Errorf("failed to retrieve factoid: %v", err)
	}

	if examplesJSON != "" {
		err = json.Unmarshal([]byte(examplesJSON), &factoid.Examples)
		if err != nil {
			factoid.Examples = []string{}
		}
	} else {
		factoid.Examples = []string{}
	}

	return factoid, nil
}
