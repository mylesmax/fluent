package chatmaster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	ID                         string         `json:"id"`
	ClassUUID                  string         `json:"class_uuid"`
	CreatedAt                  time.Time      `json:"created_at"`
	LastActivity               time.Time      `json:"last_activity"`
	FactoidQueue               []string       `json:"factoid_queue"`
	CompletedQueue             []string       `json:"completed_queue"`
	CurrentFactoid             string         `json:"current_factoid"`
	ChatHistory                []ChatExchange `json:"chat_history"`
	DropsAwarded               int            `json:"drops_awarded"`
	Status                     string         `json:"status"`//active, completed, paused
	ExchangesForCurrentFactoid int            `json:"exchanges_for_current_factoid"`
	UserProficiencyLevel       string         `json:"user_proficiency_level"` // "beginner", "intermediate", "advanced"
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

	sessions, err := cm.ListSessionsByActivity(classUUID)
	if err == nil && len(sessions) > 0 {
		sessionID := sessions[0]
		session, err := cm.GetSession(classUUID, sessionID)
		if err == nil && session.Status == "active" {
			cm.activeSessions[sessionID] = session
			return session, nil
		}
	}

	session := &ChatSession{
		ID:                         createSessionID(),
		ClassUUID:                  classUUID,
		CreatedAt:                  time.Now(),
		LastActivity:               time.Now(),
		FactoidQueue:               make([]string, 0, len(factoids)),
		CompletedQueue:             make([]string, 0),
		CurrentFactoid:             "",
		ChatHistory:                make([]ChatExchange, 0),
		DropsAwarded:               0,
		Status:                     "active",
		ExchangesForCurrentFactoid: 0,
		UserProficiencyLevel:       "beginner",
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

	cm.activeSessions[session.ID] = session

	if err := cm.persistSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (cm *ChatMaster) ProcessUserMessage(ctx context.Context, classUUID, message string) (string, ChatOutcome, error) {
	log.Printf("CHAT DEBUG: ProcessUserMessage called for class %s with message: %s", classUUID, message)

	sessions, err := cm.ListSessionsByActivity(classUUID)
	if err != nil {
		log.Printf("CHAT DEBUG: Failed to list sessions: %v", err)
		return "", OutcomeOngoing, fmt.Errorf("failed to list sessions: %v", err)
	}

	if len(sessions) == 0 {
		log.Printf("CHAT DEBUG: No sessions found for class %s", classUUID)
		return "Sorry, there is no active learning session. Please start a new session.", OutcomeOngoing, nil
	}

	sessionID := sessions[0]
	log.Printf("CHAT DEBUG: Using most recent session: %s", sessionID)

	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.activeSessions[sessionID]
	if !exists {
		log.Printf("CHAT DEBUG: Session %s not found in active------Sessions map, loading from disk, sdjojaiodjaiodnjkadfniosdfjn", sessionID)
		var err error
		session, err = cm.GetSession(classUUID, sessionID)
		if err != nil {
			log.Printf("CHAT DEBUG: Failed to get session: %v", err)
			return "", OutcomeOngoing, fmt.Errorf("failed to get session: %v", err)
		}

		cm.activeSessions[sessionID] = session
		log.Printf("CHAT DEBUG: Added session %s to activeSessions map", sessionID)
	}

	log.Printf("CHAT DEBUG: Session %s status: %s", sessionID, session.Status)
	if session.Status == "completed" {
		return "This learning session is completed. You've mastered all the factoids!", OutcomeSuccess, nil
	} else if session.Status == "paused" {
		return "This learning session is paused. Please resume it to continue learning.", OutcomeOngoing, nil
	} else if session.Status == "inactive" {
		session.Status = "active"
		log.Printf("CHAT DEBUG: Reactivated inactive session %s", sessionID)
	}

	session.LastActivity = time.Now()

	if session.CurrentFactoid == "" && len(session.FactoidQueue) > 0 {
		session.CurrentFactoid = session.FactoidQueue[0]
		session.FactoidQueue = session.FactoidQueue[1:]
		log.Printf("CHAT DEBUG: Moved to next factoid: %s", session.CurrentFactoid)
	}

	if session.CurrentFactoid == "" && len(session.FactoidQueue) == 0 {
		session.Status = "completed"
		log.Printf("CHAT DEBUG: No more factoids, session marked as completed")
		return "Congratulations! You've completed all factoids in this learning session.", OutcomeSuccess, nil
	}

	factoid, err := db.GetFactoid(classUUID, session.CurrentFactoid)
	if err != nil {
		log.Printf("CHAT DEBUG: Failed to get factoid: %v", err)
		return "", OutcomeOngoing, fmt.Errorf("failed to get factoid: %v", err)
	}
	log.Printf("CHAT DEBUG: Loaded factoid: %s", factoid.Question)

	messages := []gopenai.ChatCompletionMessage{}

	if len(session.ChatHistory) == 0 {
		promptsDir := openai.DeterminePromptsPath()
		promptPath := filepath.Join(promptsDir, "chatmaster.txt")

		systemPrompt, err := openai.LoadPrompt(promptPath)
		if err != nil {
			log.Printf("CHAT DEBUG: Failed to load chat prompt: %v", err)
			return "", OutcomeOngoing, fmt.Errorf("failed to load chat prompt: %v", err)
		}
		log.Printf("CHAT DEBUG: Loaded prompt from %s", promptPath)

		promptInfo := map[string]interface{}{
			"factoid":          factoid,
			"user_proficiency": session.UserProficiencyLevel,
		}
		factoidJSON, err := json.Marshal(promptInfo)
		if err != nil {
			log.Printf("CHAT DEBUG: Failed to marshal factoid to JSON: %v", err)
			return "", OutcomeOngoing, fmt.Errorf("failed to marshal factoid: %v", err)
		}

		systemPrompt = strings.Replace(systemPrompt, "{{FACTOID}}", string(factoidJSON), 1)
		systemPrompt = strings.Replace(systemPrompt, "{{USER_INPUT}}", message, 1)
		systemPrompt = strings.Replace(systemPrompt, "{{HISTORY}}", "", 1)

		messages = append(messages, gopenai.ChatCompletionMessage{
			Role:    gopenai.ChatMessageRoleSystem,
			Content: systemPrompt,
		})
	} else {
		promptsDir := openai.DeterminePromptsPath()
		promptPath := filepath.Join(promptsDir, "chatmaster.txt")

		systemPrompt, err := openai.LoadPrompt(promptPath)
		if err != nil {
			log.Printf("CHAT DEBUG: Failed to load chat prompt: %v", err)
			return "", OutcomeOngoing, fmt.Errorf("failed to load chat prompt: %v", err)
		}

		factoidJSON, err := json.Marshal(factoid)
		if err != nil {
			log.Printf("CHAT DEBUG: Failed to marshal factoid to JSON: %v", err)
			return "", OutcomeOngoing, fmt.Errorf("failed to marshal factoid: %v", err)
		}

		systemPrompt = strings.Replace(systemPrompt, "{{FACTOID}}", string(factoidJSON), 1)
		systemPrompt = strings.Replace(systemPrompt, "{{USER_INPUT}}", "", 1)
		systemPrompt = strings.Replace(systemPrompt, "{{HISTORY}}", "", 1)

		messages = append(messages, gopenai.ChatCompletionMessage{
			Role:    gopenai.ChatMessageRoleSystem,
			Content: systemPrompt,
		})

		recentHistory := session.ChatHistory
		maxHistory := 5
		if len(recentHistory) > maxHistory {
			recentHistory = recentHistory[len(recentHistory)-maxHistory:]
		}

		for _, exchange := range recentHistory {
			messages = append(messages, gopenai.ChatCompletionMessage{
				Role:    gopenai.ChatMessageRoleUser,
				Content: exchange.UserMessage,
			})

			messages = append(messages, gopenai.ChatCompletionMessage{
				Role:    gopenai.ChatMessageRoleAssistant,
				Content: exchange.SystemMessage,
			})
		}
	}

	messages = append(messages, gopenai.ChatCompletionMessage{
		Role:    gopenai.ChatMessageRoleUser,
		Content: message,
	})

	req := gopenai.ChatCompletionRequest{
		Model:    cm.client.GetModel(),
		Messages: messages,
	}

	log.Printf("CHAT DEBUG: Sending chat request to API for session %s with %d messages", sessionID, len(messages))
	resp, err := cm.client.GetClient().CreateChatCompletion(ctx, req)
	if err != nil {
		log.Printf("CHAT DEBUG: Failed to create chat completion: %v", err)
		return "", OutcomeOngoing, fmt.Errorf("failed to create chat completion: %v", err)
	}

	response := resp.Choices[0].Message.Content
	log.Printf("CHAT DEBUG: Got response from API (first 50 chars): %s", response[:min(50, len(response))])

	fullPrompt, _ := json.Marshal(messages)
	err = db.RecordAIChatHistory(classUUID, sessionID, string(fullPrompt), response, resp.Usage.TotalTokens, calculateCost(resp.Usage))
	if err != nil {
		log.Printf("Warning: Failed to record chat API call in history: %v", err)
	} else {
		log.Printf("Successfully recorded chat API call for session %s", sessionID)
	}

	outcome, drops := determineOutcome(response, factoid, session.ExchangesForCurrentFactoid)
	log.Printf("CHAT DEBUG: Determined outcome: %s, drops: %d", outcome, drops)

	session.ExchangesForCurrentFactoid++
	log.Printf("CHAT DEBUG: Exchanges for current factoid: %d", session.ExchangesForCurrentFactoid)

	var nextFactoidResponse string
	var shouldInitiateNextFactoid bool = false

	if outcome == OutcomeSuccess || outcome == OutcomePostpone {
		if session.CurrentFactoid != "" {
			if outcome == OutcomeSuccess {
				cm.updateUserProficiency(session, session.ExchangesForCurrentFactoid, outcome)

				session.CompletedQueue = append(session.CompletedQueue, session.CurrentFactoid)
				session.DropsAwarded += drops
				log.Printf("CHAT DEBUG: Factoid completed, moved to completed queue, awarded %d drops", drops)
			} else {
				session.FactoidQueue = append(session.FactoidQueue, session.CurrentFactoid)
				log.Printf("CHAT DEBUG: Factoid postponed, moved to end of queue")
			}

			session.CurrentFactoid = ""
			session.ExchangesForCurrentFactoid = 0

			if len(session.FactoidQueue) > 0 {
				session.CurrentFactoid = session.FactoidQueue[0]
				session.FactoidQueue = session.FactoidQueue[1:]
				log.Printf("CHAT DEBUG: Moving to next factoid: %s", session.CurrentFactoid)
				shouldInitiateNextFactoid = true
			} else {
				session.Status = "completed"
				log.Printf("CHAT DEBUG: No more factoids, session marked as completed")
				response += "\n\nLETS GOOOOO! You've completed all factoids in this learning session."
			}
		}
	}

	session.ChatHistory = append(session.ChatHistory, ChatExchange{
		UserMessage:   message,
		SystemMessage: response,
		FactoidID:     session.CurrentFactoid,
		Timestamp:     time.Now(),
		Outcome:       outcome,
	})
	log.Printf("CHAT DEBUG: Added exchange to chat history, total exchanges: %d", len(session.ChatHistory))

	err = cm.persistSession(session)
	if err != nil {
		log.Printf("CHAT DEBUG: Warning: Failed to persist session: %v", err)
	} else {
		log.Printf("CHAT DEBUG: Successfully persisted session")
	}

	cm.activeSessions[sessionID] = session
	log.Printf("CHAT DEBUG: Updated session in activeSessions map")

	if shouldInitiateNextFactoid && session.CurrentFactoid != "" {
		nextFactoid, err := db.GetFactoid(classUUID, session.CurrentFactoid)
		if err == nil {
			nextFactoidResponse = "\n\nMoving on to the next factoid: " + nextFactoid.Question
			response += nextFactoidResponse
		}
	}

	return response, outcome, nil
}

//asjidoajdpajidoajiosnjhbsiufjsiofsnjf i hate this
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

func ExtractResponseContent(fullResponse string) string {
	return extractResponseContent(fullResponse)
}

func createSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

func calculateCost(usage gopenai.Usage) float64 {
	return float64(usage.TotalTokens) * 0.000001
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

	sessionPath := filepath.Join(sessionsDir, fmt.Sprintf("%s_session_%s.json", session.ClassUUID, session.ID))

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

	prefix := classUUID + "_session_"
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
	log.Printf("CHAT DEBUG: GetSession called for class %s, session %s", classUUID, sessionID)

	if session, exists := cm.activeSessions[sessionID]; exists {
		log.Printf("CHAT DEBUG: Found session %s in activeSessions map", sessionID)
		return session, nil
	}

	log.Printf("CHAT DEBUG: Session not found in activeSessions, looking for file")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("CHAT DEBUG: Failed to get home directory: %v", err)
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	sessionPath := filepath.Join(homeDir, ".fluent", "sessions", fmt.Sprintf("%s_session_%s.json", classUUID, sessionID))
	log.Printf("CHAT DEBUG: Looking for session file at: %s", sessionPath)

	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		log.Printf("CHAT DEBUG: Session file not found: %s", sessionPath)

		alternativePath := filepath.Join(homeDir, ".fluent", "sessions", fmt.Sprintf("%s_%s.json", classUUID, sessionID))
		log.Printf("CHAT DEBUG: Trying alternative path: %s", alternativePath)

		if _, err := os.Stat(alternativePath); os.IsNotExist(err) {
			log.Printf("CHAT DEBUG: Alternative path not found either")
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		sessionPath = alternativePath
		log.Printf("CHAT DEBUG: Using alternative path")
	}

	sessionJSON, err := os.ReadFile(sessionPath)
	if err != nil {
		log.Printf("CHAT DEBUG: Failed to read session file: %v", err)
		return nil, fmt.Errorf("failed to read session file: %v", err)
	}
	log.Printf("CHAT DEBUG: Read %d bytes from session file", len(sessionJSON))

	var session ChatSession
	err = json.Unmarshal(sessionJSON, &session)
	if err != nil {
		log.Printf("CHAT DEBUG: Failed to unmarshal session: %v", err)
		return nil, fmt.Errorf("failed to unmarshal session: %v", err)
	}

	log.Printf("CHAT DEBUG: Successfully loaded session %s (status: %s)", sessionID, session.Status)

	cm.activeSessions[sessionID] = &session

	return &session, nil
}

func determineOutcome(response string, factoid db.FactoidData, exchangeCount int) (ChatOutcome, int) {
	outcome := OutcomeOngoing
	drops := 0

	if strings.Contains(response, "<<SUCCESS>>") {
		outcome = OutcomeSuccess
		drops = 1
	} else if strings.Contains(response, "<<POSTPONE>>") {
		outcome = OutcomePostpone
	} else if exchangeCount >= 5 {
		successIndicators := []string{
			"correct", "that's right", "well done", "good job", "excellent",
			"perfect", "exactly", "you got it", "you're right",
		}

		for _, indicator := range successIndicators {
			if strings.Contains(strings.ToLower(response), indicator) {
				log.Printf("CHAT DEBUG: Extended exchange (count: %d) with success indicator '%s', triggering SUCCESS",
					exchangeCount, indicator)
				outcome = OutcomeSuccess
				drops = 1
				break
			}
		}
	}

	return outcome, drops
}

func (cm *ChatMaster) EndSession(classUUID, sessionID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	log.Printf("EndSession called for %s in class %s", sessionID, classUUID)
	session, err := cm.GetSession(classUUID, sessionID)
	if err != nil {
		log.Printf("Error getting session %s: %v", sessionID, err)
		return fmt.Errorf("failed to get session: %v", err)
	}

	log.Printf("Session %s current state - Status: %s, Queue: %d, CompletedQueue: %d, CurrentFactoid: %s",
		sessionID, session.Status, len(session.FactoidQueue), len(session.CompletedQueue),
		session.CurrentFactoid)

	if session.Status == "completed" {
		log.Printf("Session %s is already completed", sessionID)
		return nil
	}

	if len(session.FactoidQueue) == 0 && session.CurrentFactoid == "" {
		session.Status = "completed"
		log.Printf("Marking session %s as completed", sessionID)
	} else {
		session.Status = "inactive"
		log.Printf("Marking session %s as inactive", sessionID)
	}

	err = cm.persistSession(session)
	if err != nil {
		log.Printf("Error persisting session %s: %v", sessionID, err)
		return fmt.Errorf("failed to persist session: %v", err)
	}

	if _, exists := cm.activeSessions[sessionID]; exists {
		log.Printf("Removing session %s from active sessions map", sessionID)
		delete(cm.activeSessions, sessionID)
	} else {
		log.Printf("Session %s not found in active sessions map", sessionID)
	}

	log.Printf("EndSession for %s completed successfully", sessionID)
	return nil
}

func (cm *ChatMaster) InitiateFactoidConversation(ctx context.Context, classUUID string, sessionID string) (string, error) {
	log.Printf("CHAT DEBUG: Initiating factoid conversation for session %s", sessionID)

	session, err := cm.GetSession(classUUID, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %v", err)
	}

	if session.CurrentFactoid == "" {
		if len(session.FactoidQueue) > 0 {
			session.CurrentFactoid = session.FactoidQueue[0]
			session.FactoidQueue = session.FactoidQueue[1:]
			log.Printf("CHAT DEBUG: Moving to next factoid: %s", session.CurrentFactoid)
		} else {
			session.Status = "completed"
			return "Congratulations! You've completed all factoids in this learning session.", nil
		}
	}

	factoid, err := db.GetFactoid(classUUID, session.CurrentFactoid)
	if err != nil {
		return "", fmt.Errorf("failed to get factoid: %v", err)
	}

	promptsDir := openai.DeterminePromptsPath()
	promptPath := filepath.Join(promptsDir, "chatmaster.txt")

	systemPrompt, err := openai.LoadPrompt(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to load chat prompt: %v", err)
	}

	initialMessage := "Let's discuss the factoid about " + factoid.Question

	promptInfo := map[string]interface{}{
		"factoid":          factoid,
		"user_proficiency": session.UserProficiencyLevel,
	}
	promptInfoJSON, err := json.Marshal(promptInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal prompt info: %v", err)
	}

	systemPrompt = strings.Replace(systemPrompt, "{{FACTOID}}", string(promptInfoJSON), 1)
	systemPrompt = strings.Replace(systemPrompt, "{{USER_INPUT}}", initialMessage, 1)
	systemPrompt = strings.Replace(systemPrompt, "{{HISTORY}}", "", 1)

	messages := []gopenai.ChatCompletionMessage{
		{
			Role:    gopenai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	req := gopenai.ChatCompletionRequest{
		Model:    cm.client.GetModel(),
		Messages: messages,
	}

	log.Printf("CHAT DEBUG: Sending initial factoid conversation request to OpenAI API for session %s", sessionID)
	resp, err := cm.client.GetClient().CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create chat completion: %v", err)
	}

	response := resp.Choices[0].Message.Content

	fullPrompt, _ := json.Marshal(messages)
	err = db.RecordAIChatHistory(classUUID, sessionID, string(fullPrompt), response, resp.Usage.TotalTokens, calculateCost(resp.Usage))
	if err != nil {
		log.Printf("Warning: Failed to record initial chat API call in history: %v", err)
	}

	session.ChatHistory = append(session.ChatHistory, ChatExchange{
		UserMessage:   initialMessage,
		SystemMessage: response,
		FactoidID:     session.CurrentFactoid,
		Timestamp:     time.Now(),
		Outcome:       OutcomeOngoing,
	})

	session.LastActivity = time.Now()

	err = cm.persistSession(session)
	if err != nil {
		log.Printf("Warning: Failed to persist session after initiating conversation: %v", err)
	}

	cm.activeSessions[sessionID] = session

	return response, nil
}

func (cm *ChatMaster) StartSessionConversation(ctx context.Context, classUUID string) (string, error) {
	log.Printf("CHAT DEBUG: Starting session conversation for class %s", classUUID)

	sessions, err := cm.ListSessionsByActivity(classUUID)
	if err != nil || len(sessions) == 0 {
		return "", fmt.Errorf("no sessions found for class %s", classUUID)
	}

	sessionID := sessions[0]
	return cm.InitiateFactoidConversation(ctx, classUUID, sessionID)
}

func (cm *ChatMaster) updateUserProficiency(session *ChatSession, exchangeCount int, outcome ChatOutcome) {
	if outcome != OutcomeSuccess {
		return
	}

	if exchangeCount <= 1 {
		if session.UserProficiencyLevel == "beginner" {
			session.UserProficiencyLevel = "intermediate"
			log.Printf("CHAT DEBUG: Updated user proficiency from beginner to intermediate (fast success)")
		} else if session.UserProficiencyLevel == "intermediate" {
			session.UserProficiencyLevel = "advanced"
			log.Printf("CHAT DEBUG: Updated user proficiency from intermediate to advanced (fast success)")
		}
	} else if exchangeCount >= 5 {
		if session.UserProficiencyLevel == "advanced" {
			session.UserProficiencyLevel = "intermediate"
			log.Printf("CHAT DEBUG: Updated user proficiency from advanced to intermediate (slow success)")
		} else if session.UserProficiencyLevel == "intermediate" {
			session.UserProficiencyLevel = "beginner"
			log.Printf("CHAT DEBUG: Updated user proficiency from intermediate to beginner (slow success)")
		}
	}
}