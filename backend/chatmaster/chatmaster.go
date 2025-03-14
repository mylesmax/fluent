package chatmaster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fluent/backend/db"
	"fluent/backend/openai"

	gopenai "github.com/sashabaranov/go-openai"
)

type ChatOutcome string

const (
	OutcomeSuccess  ChatOutcome = "success"
	OutcomePostpone ChatOutcome = "postpone"
	OutcomeOngoing  ChatOutcome = "ongoing"
)

type ChatSession struct {
	ID             string         `json:"id"`
	ClassUUID      string         `json:"class_uuid"`
	CreatedAt      time.Time      `json:"created_at"`
	LastActivity   time.Time      `json:"last_activity"`
	FactoidQueue   []string       `json:"factoid_queue"`
	CompletedQueue []string       `json:"completed_queue"`
	CurrentFactoid string         `json:"current_factoid"`
	ChatHistory    []ChatExchange `json:"chat_history"`
	DropsAwarded   int            `json:"drops_awarded"`
	Status         string         `json:"status"`//active, completed, paused
}

type ChatExchange struct {
	UserMessage   string      `json:"user_message"`
	SystemMessage string      `json:"system_message"`
	FactoidID     string      `json:"factoid_id"`
	Timestamp     time.Time   `json:"timestamp"`
	Outcome       ChatOutcome `json:"outcome"`
}

type ChatMaster struct {
	client         *openai.OpenAIClient
	activeSessions map[string]*ChatSession
	mu             sync.Mutex
}

func NewChatMaster(client *openai.OpenAIClient) *ChatMaster {
	return &ChatMaster{
		client:         client,
		activeSessions: make(map[string]*ChatSession),
	}
}

func (cm *ChatMaster) InitSession(classUUID string, factoids []db.FactoidData) (*ChatSession, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if session, exists := cm.activeSessions[classUUID]; exists && session.Status == "active" {
		return session, nil
	}

	session := &ChatSession{
		ID:           createSessionID(),
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

	if err := cm.persistSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (cm *ChatMaster) ProcessUserMessage(ctx context.Context, classUUID, userMessage string) (string, ChatOutcome, error) {
	cm.mu.Lock()
	session, exists := cm.activeSessions[classUUID]
	if !exists || session.Status != "active" {
		var err error
		session, err = cm.loadSession(classUUID)
		if err != nil {
			cm.mu.Unlock()
			return "", OutcomeOngoing, fmt.Errorf("no active session for class %s", classUUID)
		}
		cm.activeSessions[classUUID] = session
	}

	if session.CurrentFactoid == "" {
		if len(session.FactoidQueue) > 0 {
			session.CurrentFactoid = session.FactoidQueue[0]
			session.FactoidQueue = session.FactoidQueue[1:]
		} else {
			session.Status = "completed"
			cm.mu.Unlock()
			return "no more factoids to rev", OutcomeOngoing, nil
		}
	}
	cm.mu.Unlock()

	factoids, err := db.GetFactoids(classUUID)
	if err != nil {
		return "", OutcomeOngoing, fmt.Errorf("failed to get factoids: %v", err)
	}

	var currentFactoid db.FactoidData
	found := false
	for _, f := range factoids {
		if f.ID == session.CurrentFactoid {
			currentFactoid = f
			found = true
			break
		}
	}

	if !found {
		return "", OutcomeOngoing, fmt.Errorf("factoid not found: %s", session.CurrentFactoid)
	}

	response, outcome, err := cm.chatWithFactoid(ctx, currentFactoid, userMessage)
	if err != nil {
		return "", OutcomeOngoing, fmt.Errorf("chat failed: %v", err)
	}

	cm.mu.Lock()//loc
	defer cm.mu.Unlock()

	exchange := ChatExchange{
		UserMessage:   userMessage,
		SystemMessage: response,
		FactoidID:     session.CurrentFactoid,
		Timestamp:     time.Now(),
		Outcome:       outcome,
	}
	session.ChatHistory = append(session.ChatHistory, exchange)
	session.LastActivity = time.Now()

	if outcome == OutcomeSuccess {
		session.DropsAwarded += calculateDrops(currentFactoid)
		session.CompletedQueue = append(session.CompletedQueue, session.CurrentFactoid)

		newStability := currentFactoid.Stability * 1.5//todo: change this laater
		newNextReview := time.Now().Add(time.Hour * 24 * time.Duration(int(newStability)))
		err = db.UpdateFactoidReview(currentFactoid.ID, classUUID, 5, newStability, newNextReview)
		if err != nil {
			fmt.Printf("err updating factoid review: %v\n", err)
		}

		if len(session.FactoidQueue) > 0 {
			session.CurrentFactoid = session.FactoidQueue[0]
			session.FactoidQueue = session.FactoidQueue[1:]
		} else {
			session.CurrentFactoid = ""
			session.Status = "completed"
		}
	} else if outcome == OutcomePostpone {
		session.FactoidQueue = append(session.FactoidQueue, session.CurrentFactoid)

		if len(session.FactoidQueue) > 0 {
			session.CurrentFactoid = session.FactoidQueue[0]
			session.FactoidQueue = session.FactoidQueue[1:]
		} else {
			session.CurrentFactoid = ""
			session.Status = "completed"
		}
	}

	if err := cm.persistSession(session); err != nil {
		return response, outcome, fmt.Errorf("failed to persist session: %v", err)
	}

	return response, outcome, nil
}

func (cm *ChatMaster) chatWithFactoid(ctx context.Context, factoid db.FactoidData, userMessage string) (string, ChatOutcome, error) {
	promptsDir := openai.DeterminePromptsPath()
	promptPath := filepath.Join(promptsDir, "chatmaster.txt")

	systemPrompt, err := openai.LoadPrompt(promptPath)
	if err != nil {
		return "", OutcomeOngoing, fmt.Errorf("failed to load chatmaster prompt: %v", err)
	}

	factoidJSON, err := json.Marshal(factoid)
	if err != nil {
		return "", OutcomeOngoing, fmt.Errorf("failed to marshal factoid: %v", err)
	}

	systemPrompt = strings.Replace(systemPrompt, "{{FACTOID}}", string(factoidJSON), 1)
	systemPrompt = strings.Replace(systemPrompt, "{{USER_INPUT}}", userMessage, 1)

	req := gopenai.ChatCompletionRequest{
		Model: cm.client.GetModel(),
		Messages: []gopenai.ChatCompletionMessage{
			{
				Role:    gopenai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
		},
	}

	resp, err := cm.client.GetClient().CreateChatCompletion(ctx, req)
	if err != nil {
		return "", OutcomeOngoing, fmt.Errorf("failed to create chat completion: %v", err)
	}

	responseContent := resp.Choices[0].Message.Content

	response := extractResponseContent(responseContent)

	//this si totaly temporary may change this later, quick and dirty method doe
	outcome := OutcomeOngoing
	if strings.Contains(responseContent, "<<SUCCESS>>") {
		outcome = OutcomeSuccess
	} else if strings.Contains(responseContent, "<<POSTPONE>>") {
		outcome = OutcomePostpone
	}

	return response, outcome, nil
}

func extractResponseContent(fullResponse string) string {
	startIdx := strings.Index(fullResponse, "<response>")
	endIdx := strings.Index(fullResponse, "</response>")

	if startIdx != -1 && endIdx != -1 && startIdx < endIdx {
		return strings.TrimSpace(fullResponse[startIdx+10 : endIdx])
	}

	return fullResponse
}

func createSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

func calculateDrops(factoid db.FactoidData) int {
	drops := 1

	drops += factoid.Difficulty - 1

	if drops > 5 {
		drops = 5
	}

	return drops
}

func (cm *ChatMaster) persistSession(session *ChatSession) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	sessionsDir := filepath.Join(homeDir, ".fluent", "sessions")
	err = os.MkdirAll(sessionsDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create sessions directory: %v", err)
	}

	sessionPath := filepath.Join(sessionsDir, fmt.Sprintf("%s_%s.json", session.ClassUUID, session.ID))

	sessionJSON, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %v", err)
	}

	err = os.WriteFile(sessionPath, sessionJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write session file: %v", err)
	}

	return nil
}

func (cm *ChatMaster) loadSession(classUUID string) (*ChatSession, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	sessionsDir := filepath.Join(homeDir, ".fluent", "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return nil, errors.New("no sessions directory")
	}

	files, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read sessions directory: %v", err)
	}

	var mostRecentFile os.DirEntry
	var mostRecentTime time.Time

	prefix := classUUID + "_"
	for _, file := range files {
		if strings.HasPrefix(file.Name(), prefix) {
			info, err := file.Info()
			if err != nil {
				continue
			}
			if mostRecentFile == nil || info.ModTime().After(mostRecentTime) {
				mostRecentFile = file
				mostRecentTime = info.ModTime()
			}
		}
	}

	if mostRecentFile == nil {
		return nil, fmt.Errorf("no session found for class %s", classUUID)
	}

	sessionPath := filepath.Join(sessionsDir, mostRecentFile.Name())
	sessionJSON, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %v", err)
	}

	var session ChatSession
	err = json.Unmarshal(sessionJSON, &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %v", err)
	}

	return &session, nil
}

func (cm *ChatMaster) GetActiveSessions() []*ChatSession {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	result := make([]*ChatSession, 0, len(cm.activeSessions))
	for _, session := range cm.activeSessions {
		result = append(result, session)
	}

	return result
}

func (cm *ChatMaster) GetSession(classUUID, sessionID string) (*ChatSession, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if session, exists := cm.activeSessions[classUUID]; exists && session.ID == sessionID {
		return session, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	sessionPath := filepath.Join(homeDir, ".fluent", "sessions", fmt.Sprintf("%s_%s.json", classUUID, sessionID))

	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	sessionJSON, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %v", err)
	}

	var session ChatSession
	err = json.Unmarshal(sessionJSON, &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %v", err)
	}

	return &session, nil
}