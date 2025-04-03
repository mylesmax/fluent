package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	Topic          string          `json:"topic"`
	ChatHistory    json.RawMessage `json:"chat_history"`
	StartTimestamp time.Time       `json:"start_timestamp"`
	EndTimestamp   *time.Time      `json:"end_timestamp"`
	Status         string          `json:"status"`
	DropletCount   int             `json:"droplet_count"`
	TotalFactoids  int             `json:"total_factoids,omitempty"`
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
	Uploads   []ExplorerAIHistoryEntry `json:"uploads"`
	Chats     []ExplorerAIHistoryEntry `json:"chats"`
	Parser    []ExplorerAIHistoryEntry `json:"parser"`
	Extractor []ExplorerAIHistoryEntry `json:"extractor"`
}

type exploreSessionData struct {
	ID             string          `json:"id"`
	ClassUUID      string          `json:"classUUID"`
	UploadPrompt   string          `json:"uploadPrompt"`
	ChatHistory    json.RawMessage `json:"chatHistory"`
	StartTimestamp time.Time       `json:"startTimestamp"`
	EndTimestamp   *time.Time      `json:"endTimestamp"`
}

func getClassDB(classUUID string) (*sql.DB, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	classDBPath := filepath.Join(homeDir, ".fluent", "class", classUUID+".db")
	classDB, err := sql.Open("sqlite3", classDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open class database: %v", err)
	}
	return classDB, nil
}

func getDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Error getting home directory: %v", err)
		return ""
	}
	return filepath.Join(homeDir, ".fluent")
}

func GetDatabaseExplorerData(classUUID string) (*DatabaseExplorerData, error) {
	classInfo, err := getClassInfo(classUUID)
	if err != nil {
		return nil, fmt.Errorf("error getting class info: %w", err)
	}

	db, err := getClassDB(classUUID)
	if err != nil {
		return nil, fmt.Errorf("error connecting to class database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, upload_prompt, chat_history, start_timestamp, end_timestamp FROM sessions ORDER BY start_timestamp DESC`)
	if err != nil {
		log.Printf("error querying sessions from database: %v", err)
	}

	var sessionData []exploreSessionData
	if rows != nil {
		defer rows.Close()

		for rows.Next() {
			var s exploreSessionData
			err := rows.Scan(&s.ID, &s.UploadPrompt, &s.ChatHistory, &s.StartTimestamp, &s.EndTimestamp)
			if err != nil {
				log.Printf("Error scanning session row: %v", err)
				continue
			}
			s.ClassUUID = classUUID
			sessionData = append(sessionData, s)
		}
	}

	sessionsDir := filepath.Join(getDataDir(), "sessions")
	if dirInfo, err := os.Stat(sessionsDir); err == nil && dirInfo.IsDir() {
		files, err := os.ReadDir(sessionsDir)
		if err == nil {
			prefix := classUUID + "_session_"
			for _, file := range files {
				if strings.HasPrefix(file.Name(), prefix) {
					sessionID := strings.TrimPrefix(strings.TrimSuffix(file.Name(), ".json"), prefix)

					found := false
					for _, s := range sessionData {
						if s.ID == sessionID {
							found = true
							break
						}
					}

					if !found {
						sessionPath := filepath.Join(sessionsDir, file.Name())
						fileData, err := os.ReadFile(sessionPath)
						if err != nil {
							log.Printf("Error reading session file %s: %v", file.Name(), err)
							continue
						}

						var chatSession struct {
							ID             string          `json:"id"`
							ClassUUID      string          `json:"class_uuid"`
							UploadPrompt   string          `json:"upload_prompt"`
							ChatHistory    json.RawMessage `json:"chatHistory"`
							CreatedAt      time.Time       `json:"created_at"`
							LastActivity   time.Time       `json:"last_activity"`
							Status         string          `json:"status"`
							DropsAwarded   int             `json:"drops_awarded"`
							FactoidQueue   []string        `json:"factoid_queue"`
							CompletedQueue []string        `json:"completed_queue"`
							CurrentFactoid string          `json:"current_factoid"`
						}

						if err := json.Unmarshal(fileData, &chatSession); err != nil {
							log.Printf("Error parsing session file %s: %v", file.Name(), err)
							continue
						}

						sessionData = append(sessionData, exploreSessionData{
							ID:             sessionID,
							ClassUUID:      classUUID,
							UploadPrompt:   chatSession.UploadPrompt,
							ChatHistory:    chatSession.ChatHistory,
							StartTimestamp: chatSession.CreatedAt,
							EndTimestamp:   nil,
						})
					}
				}
			}
		}
	}

	log.Printf("Found %d sessions for class %s", len(sessionData), classUUID)

	var sessions []SessionInfo
	for _, s := range sessionData {
		sessionPath := filepath.Join(getDataDir(), "sessions", classUUID+"_session_"+s.ID+".json")

		sessionInfo := SessionInfo{
			ID:             s.ID,
			UploadPrompt:   s.UploadPrompt,
			ChatHistory:    s.ChatHistory,
			StartTimestamp: s.StartTimestamp,
			EndTimestamp:   s.EndTimestamp,
			Status:         "unknown",
			DropletCount:   0,
		}

		if fileData, err := os.ReadFile(sessionPath); err == nil {
			var chatSession struct {
				Status         string    `json:"status"`
				DropsAwarded   int       `json:"drops_awarded"`
				FactoidQueue   []string  `json:"factoid_queue"`
				CompletedQueue []string  `json:"completed_queue"`
				CurrentFactoid string    `json:"current_factoid"`
				LastActivity   time.Time `json:"last_activity"`
			}

			if err := json.Unmarshal(fileData, &chatSession); err == nil {
				sessionInfo.Status = "unknown"

				if chatSession.Status != "" {
					sessionInfo.Status = chatSession.Status
				}

				if len(chatSession.FactoidQueue) == 0 && chatSession.CurrentFactoid == "" {
					sessionInfo.Status = "completed"
				} else if sessionInfo.Status == "unknown" {
					sessionInfo.Status = "inactive"
				}

				if s.EndTimestamp != nil {
					if len(chatSession.FactoidQueue) == 0 && chatSession.CurrentFactoid == "" {
						sessionInfo.Status = "completed"
					} else {
						sessionInfo.Status = "inactive"
					}
				}

				if sessionInfo.Status == "active" && time.Since(chatSession.LastActivity) > 5*time.Minute {
					sessionInfo.Status = "inactive"
					log.Printf("Session %s marked as inactive due to inactivity (%s)", s.ID, time.Since(chatSession.LastActivity))
				}

				sessionInfo.DropletCount = chatSession.DropsAwarded
				sessionInfo.TotalFactoids = len(chatSession.FactoidQueue) +
					len(chatSession.CompletedQueue) +
					(map[bool]int{true: 1, false: 0})[chatSession.CurrentFactoid != ""]
			} else {
				log.Printf("error parsing session file %s: %v", sessionPath, err)
				sessionInfo.Status = "inactive"
			}
		} else {
			log.Printf("session file not found or error reading: %s", sessionPath)
			sessionInfo.Status = "inactive"
		}

		sessions = append(sessions, sessionInfo)
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
	) //todo fix dis

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
		Uploads:   []ExplorerAIHistoryEntry{},
		Chats:     []ExplorerAIHistoryEntry{},
		Parser:    []ExplorerAIHistoryEntry{},
		Extractor: []ExplorerAIHistoryEntry{},
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error getting home directory: %w", err)
	}

	aiHistoryDir := filepath.Join(homedir, ".fluent", "aiHistory")
	log.Printf("checking ai history in directory: %s", aiHistoryDir)

	files, err := os.ReadDir(aiHistoryDir)
	if err != nil {
		log.Printf("error reading aiHistory directory: %v", err)
		return &result, nil
	}

	for _, file := range files {
		log.Printf("found history database: %s", file.Name())
	}

	historyTypes := []struct {
		dbName string
		dest   *[]ExplorerAIHistoryEntry
	}{
		{"uploads", &result.Uploads},
		{"chats", &result.Chats},
		{"parser", &result.Parser},
	}

	for _, hType := range historyTypes {
		dbPath := filepath.Join(homedir, ".fluent", "aiHistory", hType.dbName+".db")
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			log.Printf("Database file not found: %s", dbPath)
			continue
		}

		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Printf("error opening %s history database: %v", hType.dbName, err)
			continue
		}
		defer db.Close()

		var tableExists bool
		tableCheckQuery := `SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='ai_history'`
		err = db.QueryRow(tableCheckQuery).Scan(&tableExists)
		if err != nil || !tableExists {
			log.Printf("ai_history table not found in %s database: %v", hType.dbName, err)
			continue
		}

		query := `SELECT id, class_uuid, session_id, prompt, response, tokens, cost, timestamp 
				 FROM ai_history WHERE class_uuid = ? ORDER BY timestamp DESC`

		rows, err := db.Query(query, classUUID)
		if err != nil {
			log.Printf("error querying %s history: %v", hType.dbName, err)
			continue
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
				log.Printf("error scanning %s history entry: %v", hType.dbName, err)
				continue
			}

			entry.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
			if err != nil {
				//if parsing fails, use current time
				entry.Timestamp = time.Now()
			}

			entries = append(entries, entry)
		}

		log.Printf("retrieved %d entries from %s database", len(entries), hType.dbName)

		if err := rows.Err(); err != nil {
			log.Printf("error iterating %s history rows: %v", hType.dbName, err)
		}

		*hType.dest = entries

		if hType.dbName == "parser" {
			result.Extractor = entries
			log.Printf("copied %d entries from parser to extractor field", len(entries))
		}
	}

	return &result, nil
}