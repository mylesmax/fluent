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
	CreatedAt                time.Time `json:"created_at"`
}

func CalculateNextReview(stability float64) time.Time {
	//TODO
	//comp. using stability for now, use FSRS later
	return time.Now()
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
		alternative_subjects_count, difficulty, examples, last_review, next_review, stability)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
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
		       alternative_subjects_count, difficulty, examples, last_review, next_review, stability, created_at
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
		       alternative_subjects_count, difficulty, examples, last_review, next_review, stability, created_at
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

	now := time.Now()
	_, err = classDB.Exec(`
		UPDATE factoids
		SET last_review = ?,
		    next_review = ?,
		    stability = ?
		WHERE id = ?;
	`, now, newNextReview, newStability, factoidID)

	if err != nil {
		return fmt.Errorf("failed to update factoid review: %v", err)
	}

	return nil
}
