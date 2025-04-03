package main

import (
	"context"
	"encoding/json"
	"fluent/backend/chatmaster"
	"fluent/backend/db"
	"fluent/backend/extraction"
	"fluent/backend/openai"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx          context.Context
	openaiClient *openai.OpenAIClient
	extractor    *extraction.Extractor
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	if os.Getenv("OPENAI_API_KEY") == "" {
		apiKeyBytes, err := os.ReadFile("backend/secret/openai.token")
		if err == nil {
			apiKey := strings.TrimSpace(string(apiKeyBytes))
			os.Setenv("OPENAI_API_KEY", apiKey)
			log.Println("OpenAI API key loaded successfully from token file")
		} else {
			log.Printf("Warning: Failed to load OpenAI API key from file: %v", err)
			log.Println("Please set OPENAI_API_KEY environment variable or create a token file at backend/secret/openai.token")
		}
	} else {
		log.Println("Using existing OPENAI_API_KEY from environment")
	}

	//startup db
	_, err := db.InitDB()
	if err != nil {
		runtime.LogError(ctx, "Failed to init db: "+err.Error())
	}

	//init OpenAI client
	a.openaiClient, err = openai.NewOpenAIClient()
	if err != nil {
		runtime.LogError(ctx, "Failed to init OpenAI client: "+err.Error())
	}

	//init the extractor
	a.extractor = extraction.NewExtractor(a.openaiClient)

	// Set window transparency
	runtime.WindowSetBackgroundColour(ctx, 0, 0, 0, 0)
	runtime.WindowSetDarkTheme(ctx)
}

// ProcessTextToFactoids processes uploaded text into factoids (THIS CALLS PARSING AGENT)
func (a *App) ProcessTextToFactoids(classUUID string, sessionID string, text string) (string, error) {
	if a.openaiClient == nil || a.extractor == nil {
		return "", fmt.Errorf("openai client or extractor not initialized")
	}

	//process using the new extractor
	factoids, err := a.extractor.ProcessTextToFactoids(a.ctx, text, classUUID, sessionID)
	if err != nil {
		runtime.LogError(a.ctx, "failed to process text to factoids: "+err.Error())
		return "", err
	}

	//convert to db
	dbFactoids := make([]db.FactoidData, len(factoids))
	for i, f := range factoids {
		dbFactoids[i] = db.FactoidData{
			Question:                 f.Question,
			Answer:                   f.Answer,
			Type:                     f.Type,
			Verbatim:                 f.Verbatim,
			Context:                  f.Context,
			RequiresClarification:    f.RequiresClarification,
			AlternativeSubjectsCount: f.AlternativeSubjectsCount,
			Difficulty:               3, //default is middle difficulty
			Examples:                 f.Examples,
		}
	}

	//json
	factoidsJSON, err := json.Marshal(dbFactoids)
	if err != nil {
		runtime.LogError(a.ctx, "failed to marshal factoids: "+err.Error())
		return "", err
	}

	//store
	err = db.StoreFactoids(classUUID, sessionID, string(factoidsJSON))
	if err != nil {
		runtime.LogError(a.ctx, "failed to store factoids: "+err.Error())
		return "", err
	}

	log.Printf("proc'd and stored %d factoids", len(factoids))

	return string(factoidsJSON), nil
}

// GetFactoids retrieves factoids for a class/course/profile
func (a *App) GetFactoids(classUUID string) ([]db.FactoidData, error) {
	factoids, err := db.GetFactoids(classUUID)
	if err != nil {
		runtime.LogError(a.ctx, "failed to get factoids: "+err.Error())
		return nil, err
	}
	return factoids, nil
}

// GetDueFactoids retrieves factoids due for review
func (a *App) GetDueFactoids(classUUID string) ([]db.FactoidData, error) {
	factoids, err := db.GetDueFactoids(classUUID)
	if err != nil {
		runtime.LogError(a.ctx, "failed to get due factoids: "+err.Error())
		return nil, err
	}
	return factoids, nil
}

// UpdateFactoidReview makes it update, after we already did the review
func (a *App) UpdateFactoidReview(factoidID string, classUUID string, rating int) error {
	var newStability float64 //this is a basic spaced repetition algorithm, i will prob replace with FSRS later
	switch rating {
	case 5: //perfect
		newStability = 2.5
	case 4:
		newStability = 2.0
	case 3:
		newStability = 1.5
	case 2:
		newStability = 1.0
	default: //failed
		newStability = 0.5
	}

	nextReview := db.CalculateNextReview(newStability)

	err := db.UpdateFactoidReview(factoidID, classUUID, rating, newStability, nextReview)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to update factoid review: "+err.Error())
		return err
	}

	return nil
}

func (a *App) GetProfiles() []db.Profile {
	profiles, err := db.GetProfiles()
	if err != nil {
		runtime.LogError(a.ctx, "faild to get profs: "+err.Error())
		return []db.Profile{}
	}
	profiles = append(profiles, db.Profile{
		Name:         "Add New",
		Emoji:        "➕",
		GlowColor:    "rgba(52, 211, 153, 0.5)",
		CurrentDrops: 0,
	})
	return profiles
}

//get data for the database explorer
func (a *App) GetDatabaseExplorerData(classUUID string) (*db.DatabaseExplorerData, error) {
	data, err := db.GetDatabaseExplorerData(classUUID)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to get database explorer data: "+err.Error())
		return nil, err
	}
	return data, nil
}

func (a *App) AddProfile(profile db.Profile) error {
	return db.AddProfile(profile)
}

func (a *App) UpdateProfileName(oldName, newName string) error {
	return db.UpdateProfileName(oldName, newName)
}

func (a *App) DeleteProfile(name string) error {
	return db.DeleteProfile(name)
}

func (a *App) UpdateDrops(name string, drops int) error {
	return db.UpdateDrops(name, drops)
}

func (a *App) UpdateProfileGlowColor(name string, newGlowColor string) error {
	return db.UpdateProfileGlowColor(name, newGlowColor)
}

func (a *App) CreateLearnSession(classUUID string, content string, conversationJSON string) (string, error) {
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	ctx := context.Background()
	factoids, err := a.extractor.ProcessTextToFactoids(ctx, content, classUUID, sessionID)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to process text to factoids: "+err.Error())
		return "", err
	}

	cm := chatmaster.NewChatMaster(a.openaiClient)

	dbFactoids := make([]db.FactoidData, 0, len(factoids))
	for _, f := range factoids {
		dbFactoid := db.FactoidData{
			Question:                 f.Question,
			Answer:                   f.Answer,
			Type:                     f.Type,
			Verbatim:                 f.Verbatim,
			Context:                  f.Context,
			RequiresClarification:    f.RequiresClarification,
			AlternativeSubjectsCount: f.AlternativeSubjectsCount,
			Difficulty:               f.Difficulty,
			Examples:                 f.Examples,
			LastReview:               f.LastReview,
			NextReview:               f.NextReview,
			Stability:                f.Stability,
			ClassUUID:                classUUID,
		}

		factoidID, err := db.StoreFactoid(classUUID, dbFactoid)
		if err != nil {
			runtime.LogError(a.ctx, "Failed to store factoid: "+err.Error())
			continue
		}

		dbFactoid.ID = factoidID
		dbFactoids = append(dbFactoids, dbFactoid)
	}

	session, err := cm.SaveFactoidsToSession(dbFactoids, classUUID)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to create session: "+err.Error())
		return "", err
	}

	if conversationJSON != "" && session != nil {
		err = a.UpdateSessionChatHistory(classUUID, session.ID, conversationJSON)
		if err != nil {
			runtime.LogError(a.ctx, "Failed to initialize conversation: "+err.Error())
		}
	}

	log.Printf("Created session %s with %d factoids", session.ID, len(dbFactoids))

	if session != nil && session.CurrentFactoid != "" {
		_, err := cm.InitiateFactoidConversation(ctx, classUUID, session.ID)
		if err != nil {
			log.Printf("Warning: Failed to initiate factoid conversation: %v", err)
		} else {
			log.Printf("Successfully initiated factoid conversation for new session %s", session.ID)
		}
	}

	return session.ID, nil
}

func (a *App) ProcessUserMessage(classUUID string, message string) (map[string]interface{}, error) {
	ctx := context.Background()
	cm := chatmaster.NewChatMaster(a.openaiClient)

	response, outcome, err := cm.ProcessUserMessage(ctx, classUUID, message)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to process user message: "+err.Error())
		return nil, err
	}

	cleanResponse := chatmaster.ExtractResponseContent(response)

	sessions, err := cm.ListSessionsByActivity(classUUID)
	if err != nil || len(sessions) == 0 {
		return map[string]interface{}{
			"content":      cleanResponse,
			"outcome":      string(outcome),
			"dropsAwarded": 0,
		}, nil
	}

	session, err := cm.GetSession(classUUID, sessions[0])
	dropsAwarded := 0
	if err == nil && session != nil {
		dropsAwarded = session.DropsAwarded

		if outcome == chatmaster.OutcomeSuccess {
			profiles, err := db.GetProfiles()
			if err == nil {
				for _, profile := range profiles {
					if profile.ClassUUID == classUUID {
						currentDrops := profile.CurrentDrops + 1
						err := db.UpdateDrops(profile.Name, currentDrops)
						if err != nil {
							log.Printf("Warning: Failed to update drops for profile %s: %v", profile.Name, err)
						} else {
							log.Printf("Successfully awarded 1 drop to profile %s (new total: %d)",
								profile.Name, currentDrops)
						}
						break
					}
				}
			}

			if session.CurrentFactoid != "" && (outcome == chatmaster.OutcomeSuccess || outcome == chatmaster.OutcomePostpone) {
				log.Printf("Moving to next factoid %s after %s outcome", session.CurrentFactoid, outcome)
			}
		}
	}

	result := map[string]interface{}{
		"content":      cleanResponse,
		"outcome":      string(outcome),
		"dropsAwarded": dropsAwarded,
	}

	return result, nil
}

func (a *App) GetSessionStatistics(classUUID string, sessionID string) (map[string]interface{}, error) {
	cm := chatmaster.NewChatMaster(a.openaiClient)

	if sessionID == "" {
		sessions, err := cm.ListSessionsByActivity(classUUID)
		if err != nil || len(sessions) == 0 {
			return nil, fmt.Errorf("no sessions found for class %s", classUUID)
		}
		sessionID = sessions[0]
		log.Printf("Using most recent session: %s", sessionID)
	}

	session, err := cm.GetSession(classUUID, sessionID)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to get session: "+err.Error())
		return nil, err
	}

	factoids, err := db.GetFactoids(classUUID)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to get factoids: "+err.Error())
	}

	relevantFactoids := make([]db.FactoidData, 0)
	factoidMap := make(map[string]db.FactoidData)

	for _, f := range factoids {
		factoidMap[f.ID] = f
	}

	for _, id := range session.CompletedQueue {
		if factoid, exists := factoidMap[id]; exists {
			relevantFactoids = append(relevantFactoids, factoid)
		}
	}

	if session.CurrentFactoid != "" {
		if factoid, exists := factoidMap[session.CurrentFactoid]; exists {
			relevantFactoids = append(relevantFactoids, factoid)
		}
	}

	for _, id := range session.FactoidQueue {
		if factoid, exists := factoidMap[id]; exists {
			relevantFactoids = append(relevantFactoids, factoid)
		}
	}

	stats := map[string]interface{}{
		"session_id":         session.ID,
		"total_factoids":     len(session.CompletedQueue) + len(session.FactoidQueue) + (map[bool]int{true: 1, false: 0})[session.CurrentFactoid != ""],
		"completed_factoids": len(session.CompletedQueue),
		"remaining_factoids": len(session.FactoidQueue) + (map[bool]int{true: 1, false: 0})[session.CurrentFactoid != ""],
		"drops_awarded":      session.DropsAwarded,
		"status":             session.Status,
		"chat_history":       session.ChatHistory,
		"factoids":           relevantFactoids,
		"created_at":         session.CreatedAt,
		"last_activity":      session.LastActivity,
	}

	return stats, nil
}

func (a *App) ListSessionsByActivity(classUUID string) ([]string, error) {
	cm := chatmaster.NewChatMaster(a.openaiClient)

	sessions, err := cm.ListSessionsByActivity(classUUID)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to list sessions: "+err.Error())
		return nil, err
	}

	return sessions, nil
}

func (a *App) ResumeSession(classUUID string, sessionID string) error {
	cm := chatmaster.NewChatMaster(a.openaiClient)
	ctx := context.Background()

	if sessionID == "" {
		sessions, err := cm.ListSessionsByActivity(classUUID)
		if err != nil || len(sessions) == 0 {
			return fmt.Errorf("no sessions found for class %s", classUUID)
		}
		sessionID = sessions[0]
		log.Printf("Using most recent session: %s", sessionID)
	}

	session, err := cm.GetSession(classUUID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %v", err)
	}

	if session.Status != "paused" && session.Status != "completed" {
		log.Printf("Session %s is already active (status: %s)", sessionID, session.Status)
		return nil
	}

	err = cm.ResumeSession(classUUID, sessionID)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to resume session: "+err.Error())
		return err
	}

	log.Printf("Successfully resumed session %s", sessionID)

	if session.CurrentFactoid != "" {
		_, err := cm.InitiateFactoidConversation(ctx, classUUID, sessionID)
		if err != nil {
			log.Printf("Warning: Failed to initiate factoid conversation on resume: %v", err)
		} else {
			log.Printf("Successfully initiated factoid conversation for resumed session %s", sessionID)
		}
	}

	return nil
}

func (a *App) RecordAIUploadHistory(classUUID string, content string) error {
	log.Printf("Recorded content upload for class %s (%d bytes)", classUUID, len(content))
	//fixlater1!!
	return nil
}

// RecordAIChatHistory records an AI chat interaction in history
func (a *App) RecordAIChatHistory(classUUID string, sessionID string, prompt string, response string, tokens int, cost float64) error {
	err := db.RecordAIChatHistory(classUUID, sessionID, prompt, response, tokens, cost)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to record AI chat history: "+err.Error())
		return err
	}
	return nil
}

// RecordAIParserHistory records an AI parser interaction in history
func (a *App) RecordAIParserHistory(classUUID string, sessionID string, prompt string, response string, tokens int, cost float64) error {
	err := db.RecordAIParserHistory(classUUID, sessionID, prompt, response, tokens, cost)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to record AI parser history: "+err.Error())
		return err
	}
	return nil
}

// UpdateSessionChatHistory updates the chat history for a session
func (a *App) UpdateSessionChatHistory(classUUID string, sessionID string, chatHistoryJSON string) error {
	chatHistory := json.RawMessage(chatHistoryJSON)
	err := db.UpdateSessionChatHistory(classUUID, sessionID, chatHistory)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to update session chat history: "+err.Error())
		return err
	}
	return nil
}

// EndSession marks a session as ended
func (a *App) EndSession(classUUID string, sessionID string) error {
	log.Printf("Ending session %s for class %s", sessionID, classUUID)

	//first, use the ChatMaster to properly end the session in the session file
	cm := chatmaster.NewChatMaster(a.openaiClient)
	err := cm.EndSession(classUUID, sessionID)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to end session via ChatMaster: "+err.Error())
		//continue anyway to try the database update
	}

	//also update the database record to add end_timestamp
	err = db.EndSession(classUUID, sessionID)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to end session in database: "+err.Error())
		return err
	}

	//get session to confirm it's properly marked as inactive or completed
	session, err := cm.GetSession(classUUID, sessionID)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to get session after ending it: "+err.Error())
	} else {
		log.Printf("Session %s status after ending: %s", sessionID, session.Status)
	}

	log.Printf("Successfully ended session %s", sessionID)
	return nil
}

// GetClassSessions retrieves all sessions for a class
func (a *App) GetClassSessions(classUUID string) ([]db.SessionData, error) {
	sessions, err := db.GetClassSessions(classUUID)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to get class sessions: "+err.Error())
		return nil, err
	}
	return sessions, nil
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// beforeClose is called when the app is about to quit
func (a *App) beforeClose(ctx context.Context) bool {
	return false
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
}

// StartSessionConversation initiates a conversation for the current factoid in a session
func (a *App) StartSessionConversation(classUUID string) (string, error) {
	ctx := context.Background()
	cm := chatmaster.NewChatMaster(a.openaiClient)

	response, err := cm.StartSessionConversation(ctx, classUUID)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to start session conversation: "+err.Error())
		return "", err
	}

	cleanResponse := chatmaster.ExtractResponseContent(response)
	return cleanResponse, nil
}
