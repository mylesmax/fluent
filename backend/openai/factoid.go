package openai

import (
	"encoding/json"
	"fmt"
	"time"
)

// a "factoid" is raw information extracted from an excerpt of the text ("verbatim" field)
type Factoid struct {
	ID                       string    `json:"id,omitempty"`               // uuid
	Question                 string    `json:"question"`                   // self-explanatory
	Answer                   string    `json:"answer"`                     // self-explanatory
	Type                     string    `json:"type"`                       // "factual" or "conceptual"
	Verbatim                 string    `json:"verbatim"`                   // self-explanatory
	Context                  string    `json:"context"`                    // self-explanatory
	RequiresClarification    bool      `json:"requires_clarification"`     // true if the factoid is not clear; false otherwise
	AlternativeSubjectsCount int       `json:"alternative_subjects_count"` // how many alternative ways can we spin the question?
	Difficulty               int       `json:"difficulty,omitempty"`       // 1-5 rating (derived from user interaction)
	Examples                 []string  `json:"examples"`                   // e.g.s for concepts
	LastReview               time.Time `json:"last_review,omitempty"`      // srs: last review timestamp
	NextReview               time.Time `json:"next_review,omitempty"`      // srs: next review timestamp
	Stability                float64   `json:"stability,omitempty"`        // srs: stability
	ClassUUID                string    `json:"class_uuid,omitempty"`       // uuid of the class/course/profile
}
type FactoidResponse struct {
	Factoids []Factoid `json:"json"`
}

func NewFactoid(question, answer, factoidType string, verbatim, context string, requiresClarification bool,
	alternativeSubjectsCount int, examples []string, classUUID string) *Factoid {

	now := time.Now()

	defaultStability := 1.0

	return &Factoid{
		Question:                 question,
		Answer:                   answer,
		Type:                     factoidType,
		Verbatim:                 verbatim,
		Context:                  context,
		RequiresClarification:    requiresClarification,
		AlternativeSubjectsCount: alternativeSubjectsCount,
		Difficulty:               3, // middle difficulty, will eventually be adjusted through interaction
		Examples:                 examples,
		LastReview:               now,
		NextReview:               now.Add(time.Hour * 24),//tmrw
		Stability:                defaultStability,
		ClassUUID:                classUUID,
	}
}

// factoid to JSON
func (f *Factoid) ToJSON() (string, error) {
	bytes, err := json.Marshal(f)
	if err != nil {
		return "", fmt.Errorf("failed to marshal factoid: %v", err)
	}
	return string(bytes), nil
}

// factoid from JSON
func FactoidFromJSON(jsonStr string) (*Factoid, error) {
	var factoid Factoid
	err := json.Unmarshal([]byte(jsonStr), &factoid)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal factoid: %v", err)
	}
	return &factoid, nil
}

// validate the factoid
func (f *Factoid) Validate() error {
	if f.Question == "" {
		return fmt.Errorf("factoid question cannot be empty")
	}
	if f.Answer == "" {
		return fmt.Errorf("factoid answer cannot be empty")
	}
	if f.Type != "factual" && f.Type != "conceptual" {
		return fmt.Errorf("factoid type must be 'factual' or 'conceptual', got %s", f.Type)
	}

	if f.Type == "conceptual" && f.AlternativeSubjectsCount > 0 {
		return fmt.Errorf("conceptual items should have alternative_subjects_count of 0")
	}

	if f.Type == "factual" && len(f.Examples) > 0 {
		return fmt.Errorf("factual items should have empty examples array")
	}

	return nil
}

func ParseFactoidsFromResponse(rawResponse []byte) ([]Factoid, error) {
	//parsing the expected structure { "json": [...] }
	//parsing as a direct array [...]
	//if fail, return an error
	var response FactoidResponse
	errResponse := json.Unmarshal(rawResponse, &response)
	if errResponse == nil && len(response.Factoids) > 0 {
		return response.Factoids, nil
	}

	var directArray []Factoid
	errDirect := json.Unmarshal(rawResponse, &directArray)
	if errDirect == nil && len(directArray) > 0 {
		return directArray, nil
	}

	return nil, fmt.Errorf("failed to parse factoids from response:\n"+
		"Tried 'json' field: %v\n"+
		"Tried direct array: %v\n"+
		"raw response: %s", errResponse, errDirect, string(rawResponse))
}