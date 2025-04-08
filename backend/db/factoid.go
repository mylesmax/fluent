package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type FactoidData struct {
	ID                       string    `json:"id"`
	ClassUUID                string    `json:"class_uuid"`
	Question                 string    `json:"question"`
	Answer                   string    `json:"answer"`
	Type                     string    `json:"type"`
	Verbatim                 string    `json:"verbatim"`
	Context                  string    `json:"context"`
	RequiresClarification    bool      `json:"requires_clarification"`
	AlternativeSubjectsCount int       `json:"alternative_subjects_count"`
	Difficulty               int       `json:"difficulty"`
	Examples                 []string  `json:"examples"`
	LastReview               time.Time `json:"last_review"`
	NextReview               time.Time `json:"next_review"`
	Stability                float64   `json:"stability"`
	Repetitions              int       `json:"repetitions"`
	EaseFactor               float64   `json:"ease_factor"`
	Interval                 int       `json:"interval"`
	CreatedAt                time.Time `json:"created_at"`
}

func CalculateNextReview(stability float64) time.Time {
	daysToAdd := int(stability)
	if daysToAdd < 1 {
		daysToAdd = 1
	}

	return time.Now().AddDate(0, 0, daysToAdd)
}

// SuperMemo 2 algorithm for spaced repetition
func SM2Algorithm(quality int, factoid *FactoidData) {
	if factoid.EaseFactor == 0 {
		factoid.EaseFactor = 2.5
	}

	if quality >= 3 {
		if factoid.Repetitions == 0 {
			factoid.Interval = 1
		} else if factoid.Repetitions == 1 {
			factoid.Interval = 6
		} else {
			factoid.Interval = int(float64(factoid.Interval) * factoid.EaseFactor)
		}

		factoid.Repetitions++

		factoid.EaseFactor = factoid.EaseFactor + (0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02))
	} else {
		factoid.Repetitions = 0
		factoid.Interval = 1
	}

	if factoid.EaseFactor < 1.3 {
		factoid.EaseFactor = 1.3
	}

	factoid.NextReview = time.Now().AddDate(0, 0, factoid.Interval)
	factoid.LastReview = time.Now()
	factoid.Stability = factoid.EaseFactor
}

// save an array of factoids to the db
func StoreFactoids(classUUID string, sessionID string, factoidsJSON string) error {
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
		CREATE TABLE IF NOT EXISTS factoids (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			question TEXT NOT NULL,
			answer TEXT NOT NULL,
			type TEXT NOT NULL,
			verbatim TEXT NOT NULL,
			context TEXT NOT NULL,
			requires_clarification BOOLEAN NOT NULL,
			alternative_subjects_count INTEGER NOT NULL,
			difficulty INTEGER NOT NULL,
			examples TEXT NOT NULL,
			last_review TIMESTAMP NOT NULL,
			next_review TIMESTAMP NOT NULL,
			stability REAL NOT NULL,
			repetitions INTEGER NOT NULL DEFAULT 0,
			ease_factor REAL NOT NULL DEFAULT 2.5,
			interval INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create factoids table: %v", err)
	}

	var factoids []FactoidData
	err = json.Unmarshal([]byte(factoidsJSON), &factoids)
	if err != nil {
		return fmt.Errorf("failed to unmarshal factoids JSON: %v", err)
	}

	tx, err := classDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	stmt, err := tx.Prepare(`
		INSERT INTO factoids 
		(id, session_id, question, answer, type, verbatim, context, requires_clarification, 
		alternative_subjects_count, difficulty, examples, last_review, next_review, stability,
		repetitions, ease_factor, interval)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, f := range factoids {
		if f.ID == "" {
			f.ID = uuid.New().String()
		}
		if f.LastReview.IsZero() {
			f.LastReview = now
		}
		if f.NextReview.IsZero() {
			f.NextReview = now.Add(24 * time.Hour)
		}
		if f.Stability == 0 {
			f.Stability = 1.0//default stability
		}
		if f.EaseFactor == 0 {
			f.EaseFactor = 2.5 //default ease factor
		}
		if f.Difficulty == 0 {
			f.Difficulty = 3//middle difficulty
		}

		examplesJSON, err := json.Marshal(f.Examples)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to marshal examples: %v", err)
		}

		_, err = stmt.Exec(
			f.ID,
			sessionID,
			f.Question,
			f.Answer,
			f.Type,
			f.Verbatim,
			f.Context,
			f.RequiresClarification,
			f.AlternativeSubjectsCount,
			f.Difficulty,
			string(examplesJSON),
			f.LastReview,
			f.NextReview,
			f.Stability,
			f.Repetitions,
			f.EaseFactor,
			f.Interval,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert factoid: %v", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// get factoids for a specific class
func GetFactoids(classUUID string) ([]FactoidData, error) {
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

	rows, err := classDB.Query(`
		SELECT id, session_id, question, answer, type, verbatim, context, requires_clarification,
		       alternative_subjects_count, difficulty, examples, last_review, next_review, stability, 
		       repetitions, ease_factor, interval, created_at
		FROM factoids
		ORDER BY created_at DESC;
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query factoids: %v", err)
	}
	defer rows.Close()

	var factoids []FactoidData
	for rows.Next() {
		var f FactoidData
		var sessionID string
		var examplesJSON string

		err := rows.Scan(
			&f.ID,
			&sessionID,
			&f.Question,
			&f.Answer,
			&f.Type,
			&f.Verbatim,
			&f.Context,
			&f.RequiresClarification,
			&f.AlternativeSubjectsCount,
			&f.Difficulty,
			&examplesJSON,
			&f.LastReview,
			&f.NextReview,
			&f.Stability,
			&f.Repetitions,
			&f.EaseFactor,
			&f.Interval,
			&f.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan factoid row: %v", err)
		}

		if err := json.Unmarshal([]byte(examplesJSON), &f.Examples); err != nil {
			return nil, fmt.Errorf("failed to unmarshal examples: %v", err)
		}

		f.ClassUUID = classUUID
		factoids = append(factoids, f)
	}

	return factoids, nil
}

// get factoids due for review
func GetDueFactoids(classUUID string) ([]FactoidData, error) {
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

	now := time.Now()
	rows, err := classDB.Query(`
		SELECT id, session_id, question, answer, type, verbatim, context, requires_clarification,
		       alternative_subjects_count, difficulty, examples, last_review, next_review, stability,
		       repetitions, ease_factor, interval, created_at
		FROM factoids
		WHERE next_review <= ?
		ORDER BY next_review;
	`, now)
	if err != nil {
		return nil, fmt.Errorf("failed to query due factoids: %v", err)
	}
	defer rows.Close()

	var factoids []FactoidData
	for rows.Next() {
		var f FactoidData
		var sessionID string
		var examplesJSON string

		err := rows.Scan(
			&f.ID,
			&sessionID,
			&f.Question,
			&f.Answer,
			&f.Type,
			&f.Verbatim,
			&f.Context,
			&f.RequiresClarification,
			&f.AlternativeSubjectsCount,
			&f.Difficulty,
			&examplesJSON,
			&f.LastReview,
			&f.NextReview,
			&f.Stability,
			&f.Repetitions,
			&f.EaseFactor,
			&f.Interval,
			&f.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan factoid row: %v", err)
		}

		if err := json.Unmarshal([]byte(examplesJSON), &f.Examples); err != nil {
			return nil, fmt.Errorf("failed to unmarshal examples: %v", err)
		}

		f.ClassUUID = classUUID
		factoids = append(factoids, f)
	}

	return factoids, nil
}

// after review, update the factoid
func UpdateFactoidReview(factoidID string, classUUID string, rating int, newStability float64, newNextReview time.Time) error {
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

	var factoid FactoidData
	var sessionID string
	var examplesJSON string

	err = classDB.QueryRow(`
		SELECT id, session_id, question, answer, type, verbatim, context, requires_clarification,
		       alternative_subjects_count, difficulty, examples, last_review, next_review, stability,
		       repetitions, ease_factor, interval, created_at
		FROM factoids
		WHERE id = ?
	`, factoidID).Scan(
		&factoid.ID,
		&sessionID,
		&factoid.Question,
		&factoid.Answer,
		&factoid.Type,
		&factoid.Verbatim,
		&factoid.Context,
		&factoid.RequiresClarification,
		&factoid.AlternativeSubjectsCount,
		&factoid.Difficulty,
		&examplesJSON,
		&factoid.LastReview,
		&factoid.NextReview,
		&factoid.Stability,
		&factoid.Repetitions,
		&factoid.EaseFactor,
		&factoid.Interval,
		&factoid.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to retrieve factoid for review update: %v", err)
	}

	SM2Algorithm(rating, &factoid)

	_, err = classDB.Exec(`
		UPDATE factoids
		SET last_review = ?,
		    next_review = ?,
		    stability = ?,
		    repetitions = ?,
		    ease_factor = ?,
		    interval = ?
		WHERE id = ?;
	`, factoid.LastReview, factoid.NextReview, factoid.Stability, factoid.Repetitions, factoid.EaseFactor, factoid.Interval, factoidID)

	if err != nil {
		return fmt.Errorf("failed to update factoid review: %v", err)
	}

	return nil
}

// save a single factoid to the database and get id
func StoreFactoid(classUUID string, factoid FactoidData) (string, error) {
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
		CREATE TABLE IF NOT EXISTS factoids (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			question TEXT NOT NULL,
			answer TEXT NOT NULL,
			type TEXT NOT NULL,
			verbatim TEXT NOT NULL,
			context TEXT NOT NULL,
			requires_clarification BOOLEAN NOT NULL,
			alternative_subjects_count INTEGER NOT NULL,
			difficulty INTEGER NOT NULL,
			examples TEXT NOT NULL,
			last_review TIMESTAMP NOT NULL,
			next_review TIMESTAMP NOT NULL,
			stability REAL NOT NULL,
			repetitions INTEGER NOT NULL DEFAULT 0,
			ease_factor REAL NOT NULL DEFAULT 2.5,
			interval INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return "", fmt.Errorf("failed to create factoids table: %v", err)
	}

	factoid.ID = uuid.New().String()

	now := time.Now()
	sessionID := "default_session"

	if factoid.LastReview.IsZero() {
		factoid.LastReview = now
	}

	if factoid.NextReview.IsZero() {
		factoid.NextReview = now.Add(24 * time.Hour)
	}

	if factoid.Stability == 0 {
		factoid.Stability = 1.0 // default stability
	}

	if factoid.EaseFactor == 0 {
		factoid.EaseFactor = 2.5 // default ease factor
	}

	if factoid.Difficulty == 0 {
		factoid.Difficulty = 3 // middle difficulty
	}

	examplesJSON, err := json.Marshal(factoid.Examples)
	if err != nil {
		return "", fmt.Errorf("failed to marshal examples: %v", err)
	}

	// factoid addition logic
	_, err = classDB.Exec(`
		INSERT INTO factoids 
		(id, session_id, question, answer, type, verbatim, context, requires_clarification, 
		alternative_subjects_count, difficulty, examples, last_review, next_review, stability,
		repetitions, ease_factor, interval)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`,
		factoid.ID,
		sessionID,
		factoid.Question,
		factoid.Answer,
		factoid.Type,
		factoid.Verbatim,
		factoid.Context,
		factoid.RequiresClarification,
		factoid.AlternativeSubjectsCount,
		factoid.Difficulty,
		string(examplesJSON),
		factoid.LastReview,
		factoid.NextReview,
		factoid.Stability,
		factoid.Repetitions,
		factoid.EaseFactor,
		factoid.Interval,
	)
	if err != nil {
		return "", fmt.Errorf("failed to insert factoid: %v", err)
	}

	return factoid.ID, nil
}