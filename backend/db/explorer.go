package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// class db metadata: uuid, createdat, lastmodified, namehistory
type ClassInfo struct {
	UUID         string    `json:"uuid"`
	CreatedAt    time.Time `json:"created_at"`
	LastModified time.Time `json:"last_modified"`
	NameHistory  []string  `json:"name_history"`
}

// an arbitrary learning session in the class database
type SessionInfo struct {
	ID             string          `json:"id"`
	UploadPrompt   string          `json:"upload_prompt"`
	ChatHistory    json.RawMessage `json:"chat_history"`
	StartTimestamp time.Time       `json:"start_timestamp"`
	EndTimestamp   *time.Time      `json:"end_timestamp"`
}

// an arbitrary entry in the AI history database for explorer
type ExplorerAIHistoryEntry struct {
	ID        int       `json:"id"`
	ClassUUID string    `json:"class_uuid"`
	SessionID string    `json:"session_id"`
	Prompt    string    `json:"prompt"`
	Response  string    `json:"response"`
	Tokens    int       `json:"tokens"`
	Cost      float64   `json:"cost"`
	Timestamp time.Time `json:"timestamp"`
}

type DatabaseExplorerData struct {
	ClassInfo ClassInfo        `json:"classInfo"`
	Sessions  []SessionInfo    `json:"sessions"`
	AIHistory AIHistoryEntries `json:"aiHistory"`
}

type AIHistoryEntries struct {
	Uploads []ExplorerAIHistoryEntry `json:"uploads"`
	Chats   []ExplorerAIHistoryEntry `json:"chats"`
	Parser  []ExplorerAIHistoryEntry `json:"parser"`
}

func GetDatabaseExplorerData(classUUID string) (*DatabaseExplorerData, error) {
	classInfo, err := getClassInfo(classUUID)
	if err != nil {
		return nil, fmt.Errorf("error getting class info: %w", err)
	}

	sessionData, err := GetClassSessions(classUUID)
	if err != nil {
		return nil, fmt.Errorf("error getting sessions: %w", err)
	}

	//in order to display the sessions in the explorer, we need to convert the SessionData to SessionInfo
	sessions := make([]SessionInfo, len(sessionData))
	for i, s := range sessionData {
		sessions[i] = SessionInfo{
			ID:             s.ID,
			UploadPrompt:   s.UploadPrompt,
			ChatHistory:    s.ChatHistory,
			StartTimestamp: s.StartTimestamp,
			EndTimestamp:   s.EndTimestamp,
		}
	}

	aiHistory, err := getAIHistoryEntries(classUUID)
	if err != nil {
		return nil, fmt.Errorf("error getting AI history: %w", err)
	}

	return &DatabaseExplorerData{
		ClassInfo: *classInfo,
		Sessions:  sessions,
		AIHistory: *aiHistory,
	}, nil
}

func getClassInfo(classUUID string) (*ClassInfo, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error getting home directory: %w", err)
	}

	classDBPath := filepath.Join(homedir, ".fluent", "class", classUUID+".db")
	if _, err := os.Stat(classDBPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("class database does not exist: %s", classDBPath)
	}

	classDB, err := sql.Open("sqlite3", classDBPath)
	if err != nil {
		return nil, fmt.Errorf("error opening class database: %w", err)
	}
	defer classDB.Close()

	var (
		uuid         string
		createdAt    time.Time
		lastModified time.Time
		nameHistory  string
	)//todo fix dis

	query := "SELECT uuid, created_at, last_modified, name_history FROM class_info LIMIT 1"
	err = classDB.QueryRow(query).Scan(&uuid, &createdAt, &lastModified, &nameHistory)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no class info found in database")
		}
		return nil, fmt.Errorf("error querying class info: %w", err)
	}

	var names []string
	if err := json.Unmarshal([]byte(nameHistory), &names); err != nil {
		names = []string{nameHistory}
	}

	return &ClassInfo{
		UUID:         uuid,
		CreatedAt:    createdAt,
		LastModified: lastModified,
		NameHistory:  names,
	}, nil
}

// get ai history for a particular class
func getAIHistoryEntries(classUUID string) (*AIHistoryEntries, error) {
	result := AIHistoryEntries{
		Uploads: []ExplorerAIHistoryEntry{},
		Chats:   []ExplorerAIHistoryEntry{},
		Parser:  []ExplorerAIHistoryEntry{},
	}

	historyTypes := []struct {
		dbName string
		dest   *[]ExplorerAIHistoryEntry
	}{
		{"uploads", &result.Uploads},
		{"chats", &result.Chats},
		{"parser", &result.Parser},
	}

	homedir, err := os.UserHomeDir()//todo: i wonder if this would be different on winodws
	if err != nil {
		return nil, fmt.Errorf("error getting home directory: %w", err)
	}

	for _, hType := range historyTypes {
		dbPath := filepath.Join(homedir, ".fluent", "aiHistory", hType.dbName+".db")
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			continue
		}

		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return nil, fmt.Errorf("error opening %s history database: %w", hType.dbName, err)
		}
		defer db.Close()

		query := `SELECT id, class_uuid, session_id, prompt, response, tokens, cost, timestamp 
				 FROM ai_history WHERE class_uuid = ? ORDER BY timestamp DESC`

		rows, err := db.Query(query, classUUID)
		if err != nil {
			return nil, fmt.Errorf("error querying %s history: %w", hType.dbName, err)
		}
		defer rows.Close()

		entries := []ExplorerAIHistoryEntry{}
		for rows.Next() {
			var entry ExplorerAIHistoryEntry
			var timestampStr string

			err := rows.Scan(
				&entry.ID,
				&entry.ClassUUID,
				&entry.SessionID,
				&entry.Prompt,
				&entry.Response,
				&entry.Tokens,
				&entry.Cost,
				&timestampStr,
			)
			if err != nil {
				return nil, fmt.Errorf("error scanning %s history entry: %w", hType.dbName, err)
			}

			entry.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
			if err != nil {
				//if parsing fails, use current time
				entry.Timestamp = time.Now()
			}

			entries = append(entries, entry)
		}

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating %s history rows: %w", hType.dbName, err)
		}

		*hType.dest = entries
	}

	return &result, nil
}