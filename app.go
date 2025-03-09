package main

import (
	"context"
	"encoding/json"
	"fluent/backend/db"
	"fluent/backend/extraction"
	"fluent/backend/openai"
	"fmt"
	"log"

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
	factoids, err := a.extractor.ProcessTextToFactoids(a.ctx, text)
	if err != nil {
		runtime.LogError(a.ctx, "faildd to process text to factoids: "+err.Error())
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

// CreateLearnSession creates a new learn session in the database
func (a *App) CreateLearnSession(classUUID string, uploadPrompt string, chatHistoryJSON string) (string, error) {
	chatHistory := json.RawMessage(chatHistoryJSON)
	sessionID, err := db.CreateLearnSession(classUUID, uploadPrompt, chatHistory)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to create learn session: "+err.Error())
		return "", err
	}
	return sessionID, nil
}

// RecordAIUploadHistory records an AI upload interaction in history
func (a *App) RecordAIUploadHistory(classUUID string, sessionID string, prompt string, response string, tokens int, cost float64) error {
	err := db.RecordAIUploadHistory(classUUID, sessionID, prompt, response, tokens, cost)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to record AI upload history: "+err.Error())
		return err
	}
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
	err := db.EndSession(classUUID, sessionID)
	if err != nil {
		runtime.LogError(a.ctx, "Failed to end session: "+err.Error())
		return err
	}
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
