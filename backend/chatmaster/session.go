package chatmaster

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"fluent/backend/db"
)

func (cm *ChatMaster) SaveFactoidsToSession(factoids []db.FactoidData, classUUID string) (*ChatSession, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.activeSessions[classUUID]

	if !exists || session.Status != "active" {
		sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

		session = &ChatSession{
			ID:           sessionID,
			ClassUUID:    classUUID,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
			FactoidQueue: make([]string, 0, len(factoids)),
			Status:       "active",
		}
		
		for _, f := range factoids {
			session.FactoidQueue = append(session.FactoidQueue, f.ID)
		}

		if len(session.FactoidQueue) > 0 {
			session.CurrentFactoid = session.FactoidQueue[0]
			session.FactoidQueue = session.FactoidQueue[1:]
		} else {
			session.Status = "completed"
		}
		
		cm.activeSessions[classUUID] = session
		//ppppppppppersist
		if err := cm.persistSession(session); err != nil {
			return nil, err
		}

		return session, nil
	}

	for _, f := range factoids {
		session.FactoidQueue = append(session.FactoidQueue, f.ID)
	}

	if session.CurrentFactoid == "" && len(session.FactoidQueue) > 0 {
		session.CurrentFactoid = session.FactoidQueue[0]
		session.FactoidQueue = session.FactoidQueue[1:]
	}

	if err := cm.persistSession(session); err != nil {
		return nil, fmt.Errorf("failed to persist session: %v", err)
	}

	return session, nil
}

func (cm *ChatMaster) ListSessions(classUUID string) ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	sessionsDir := filepath.Join(homeDir, ".fluent", "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read sessions directory: %v", err)
	}

	var sessionIDs []string
	prefix := classUUID + "_"
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" && len(file.Name()) > len(prefix) && file.Name()[:len(prefix)] == prefix {
			sessionID := file.Name()[len(prefix) : len(file.Name())-5]
			sessionIDs = append(sessionIDs, sessionID)
		}
	}

	return sessionIDs, nil
}

func (cm *ChatMaster) ExportSession(classUUID, sessionID, exportPath string) error {
	session, err := cm.GetSession(classUUID, sessionID)
	if err != nil {
		return err
	}

	exportData := struct {
		Session    *ChatSession `json:"session"`
		ExportTime time.Time    `json:"export_time"`
	}{
		Session:    session,
		ExportTime: time.Now(),
	}

	exportJSON, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %v", err)
	}

	err = os.WriteFile(exportPath, exportJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write export file: %v", err)
	}

	return nil
}

func (cm *ChatMaster) GetSessionStatistics(sessionID, classUUID string) (map[string]interface{}, error) {
	session, err := cm.GetSession(classUUID, sessionID)
	if err != nil {
		return nil, err
	}

	factoids, err := db.GetFactoids(classUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get factoids: %v", err)
	}

	factoidMap := make(map[string]db.FactoidData)
	for _, f := range factoids {
		factoidMap[f.ID] = f
	}

	stats := map[string]interface{}{
		"total_factoids":     len(session.CompletedQueue) + len(session.FactoidQueue) + (map[bool]int{true: 1, false: 0})[session.CurrentFactoid != ""],
		"completed_factoids": len(session.CompletedQueue),
		"remaining_factoids": len(session.FactoidQueue) + (map[bool]int{true: 1, false: 0})[session.CurrentFactoid != ""],
		"drops_awarded":      session.DropsAwarded,
		"session_duration":   session.LastActivity.Sub(session.CreatedAt).Minutes(),
		"status":             session.Status,
	}

	successCount := 0
	postponeCount := 0
	for _, exchange := range session.ChatHistory {
		if exchange.Outcome == OutcomeSuccess {
			successCount++
		} else if exchange.Outcome == OutcomePostpone {
			postponeCount++
		}
	}

	totalOutcomes := successCount + postponeCount
	if totalOutcomes > 0 {
		stats["success_rate"] = float64(successCount) / float64(totalOutcomes)
	} else {
		stats["success_rate"] = 0.0
	}

	difficultyCount := make(map[int]int)
	for _, id := range session.CompletedQueue {
		if factoid, exists := factoidMap[id]; exists {
			difficultyCount[factoid.Difficulty]++
		}
	}
	for _, id := range session.FactoidQueue {
		if factoid, exists := factoidMap[id]; exists {
			difficultyCount[factoid.Difficulty]++
		}
	}
	if session.CurrentFactoid != "" {
		if factoid, exists := factoidMap[session.CurrentFactoid]; exists {
			difficultyCount[factoid.Difficulty]++
		}
	}

	stats["difficulty_distribution"] = difficultyCount

	return stats, nil
}

func (cm *ChatMaster) ListSessionsByActivity(classUUID string) ([]string, error) {
	sessionIDs, err := cm.ListSessions(classUUID)
	if err != nil {
		return nil, err
	}

	type sessionInfo struct {
		ID           string
		LastActivity time.Time
	}

	var sessions []sessionInfo
	for _, id := range sessionIDs {
		session, err := cm.GetSession(classUUID, id)
		if err != nil {
			continue
		}
		sessions = append(sessions, sessionInfo{
			ID:           id,
			LastActivity: session.LastActivity,
		})
	}
	
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastActivity.After(sessions[j].LastActivity)
	})

	result := make([]string, len(sessions))
	for i, s := range sessions {
		result[i] = s.ID
	}

	return result, nil
}

func (cm *ChatMaster) PauseSession(classUUID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.activeSessions[classUUID]
	if !exists || session.Status != "active" {
		return fmt.Errorf("no active session for class %s", classUUID)
	}

	session.Status = "paused"
	if err := cm.persistSession(session); err != nil {
		return fmt.Errorf("failed to persist session: %v", err)
	}

	return nil
}

func (cm *ChatMaster) ResumeSession(classUUID, sessionID string) error {
	session, err := cm.GetSession(classUUID, sessionID)
	if err != nil {
		return err
	}

	if session.Status != "paused" {
		return fmt.Errorf("session is not paused: %s", sessionID)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	session.Status = "active"
	cm.activeSessions[classUUID] = session

	if err := cm.persistSession(session); err != nil {
		return fmt.Errorf("failed to persist session: %v", err)
	}

	return nil
}

func (cm *ChatMaster) ClearActiveSessions() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.activeSessions = make(map[string]*ChatSession)
}