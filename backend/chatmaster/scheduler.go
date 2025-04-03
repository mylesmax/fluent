package chatmaster

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"fluent/backend/db"
	"fluent/backend/extraction"
	"fluent/backend/openai"
)
//use queueueueu
type FactoidQueue struct {
	Items     []string //factoids in the queue
	Current   string   //currently active factoid
	Completed []string //completed factoids
	Postponed []string //postponed factoids
}

type Scheduler struct {
	chatMaster     *ChatMaster
	activeQueues   map[string]*FactoidQueue
	activeSessions map[string]string
	mu             sync.Mutex
}

func NewScheduler(chatMaster *ChatMaster) *Scheduler {
	return &Scheduler{
		chatMaster:     chatMaster,
		activeQueues:   make(map[string]*FactoidQueue),
		activeSessions: make(map[string]string),
	}
}

func (s *Scheduler) EnqueueFactoids(ctx context.Context, classUUID string, factoids []openai.Factoid) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
			return fmt.Errorf("failed to store factoid: %v", err)
		}

		dbFactoid.ID = factoidID
		dbFactoids = append(dbFactoids, dbFactoid)
	}

	queue, exists := s.activeQueues[classUUID]

	if !exists {
		queue = &FactoidQueue{
			Items:     make([]string, 0, len(dbFactoids)),
			Completed: make([]string, 0),
			Postponed: make([]string, 0),
		}
		s.activeQueues[classUUID] = queue
	}

	for _, f := range dbFactoids {
		queue.Items = append(queue.Items, f.ID)
	}

	if queue.Current == "" && len(queue.Items) > 0 {
		queue.Current = queue.Items[0]
		queue.Items = queue.Items[1:]
	}

	session, err := s.chatMaster.SaveFactoidsToSession(dbFactoids, classUUID)
	if err != nil {
		return fmt.Errorf("couldn't create/upd. chat session: %v", err)
	}

	s.activeSessions[classUUID] = session.ID

	log.Printf("Added %d factoids for class %s", len(factoids), classUUID)
	return nil
}

func (s *Scheduler) GetCurrentFactoid(ctx context.Context, classUUID string) (*openai.Factoid, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	queue, exists := s.activeQueues[classUUID]
	if !exists || queue.Current == "" {
		return nil, errors.New("no active factoid for this class")
	}

	factoids, err := db.GetFactoids(classUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get factoids: %v", err)
	}

	for _, f := range factoids {
		if f.ID == queue.Current {
			return &openai.Factoid{
				ID:                       f.ID,
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
				ClassUUID:                f.ClassUUID,
			}, nil
		}
	}

	return nil, fmt.Errorf("factoid not found: %s", queue.Current)
}

func (s *Scheduler) ProcessUserMessage(ctx context.Context, userMessage, classUUID string) (string, ChatOutcome, error) {
	response, outcome, err := s.chatMaster.ProcessUserMessage(ctx, classUUID, userMessage)
	if err != nil {
		return "", OutcomeOngoing, fmt.Errorf("failed to process message: %v", err)
	}

	if outcome == OutcomeSuccess || outcome == OutcomePostpone {
		s.mu.Lock()
		defer s.mu.Unlock()

		queue, exists := s.activeQueues[classUUID]
		if !exists {
			return response, outcome, nil
		}

		if outcome == OutcomeSuccess {
			queue.Completed = append(queue.Completed, queue.Current)
			//drops
			//sudhfishfiushduifhsiufhiufhduifhsiufhsudifn yah WHY Is this not working GG
			//move to diff method TODO
		} else if outcome == OutcomePostpone {
			queue.Postponed = append(queue.Postponed, queue.Current)
			queue.Items = append(queue.Items, queue.Current)
		}

		if len(queue.Items) > 0 {
			queue.Current = queue.Items[0]
			queue.Items = queue.Items[1:]
		} else {
			queue.Current = ""
		}
	}

	return response, outcome, nil
}

func (s *Scheduler) GetSessionProgress(classUUID string) (map[string]interface{}, error) {
	s.mu.Lock()
	sessionID, exists := s.activeSessions[classUUID]
	s.mu.Unlock()

	if !exists {
		return nil, errors.New("no active session for this class")
	}

	return s.chatMaster.GetSessionStatistics(classUUID, sessionID)
}

func (s *Scheduler) ResetQueue(classUUID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	queue, exists := s.activeQueues[classUUID]
	if !exists {
		return errors.New("no active queue for this class")
	}

	allFactoids := append([]string{}, queue.Items...)
	if queue.Current != "" {
		allFactoids = append(allFactoids, queue.Current)
	}
	allFactoids = append(allFactoids, queue.Completed...)
	allFactoids = append(allFactoids, queue.Postponed...)

	queue.Items = allFactoids
	queue.Current = ""
	queue.Completed = []string{}
	queue.Postponed = []string{}

	if len(queue.Items) > 0 {
		queue.Current = queue.Items[0]
		queue.Items = queue.Items[1:]
	}

	return nil
}

func (s *Scheduler) CreateSessionFromOlfactionDemo(extractor *extraction.Extractor, ctx context.Context, classUUID string) error {
	promptsDir := openai.DeterminePromptsPath()
	olfactionPath := filepath.Join(promptsDir, "olfaction.txt")
	olfactionText, err := openai.LoadPrompt(olfactionPath)
	if err != nil {
		return fmt.Errorf("failed to load olfaction text: %v", err)
	}

	factoids, err := extractor.ProcessTextToFactoids(ctx, olfactionText, classUUID, "demo_session")
	if err != nil {
		return fmt.Errorf("failed to process olfaction text: %v", err)
	}

	return s.EnqueueFactoids(ctx, classUUID, factoids)
}