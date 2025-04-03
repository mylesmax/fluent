package extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"fluent/backend/db"
	"fluent/backend/openai"

	gopenai "github.com/sashabaranov/go-openai"
)

type Extractor struct {
	client *openai.OpenAIClient
}

func NewExtractor(client *openai.OpenAIClient) *Extractor {
	return &Extractor{
		client: client,
	}
}

func (e *Extractor) ProcessTextToFactoids(ctx context.Context, text string, classUUID string, sessionID string) ([]openai.Factoid, error) {
	promptsDir := openai.DeterminePromptsPath()
	promptPath := filepath.Join(promptsDir, "factoid_extraction.txt")

	systemPrompt, err := openai.LoadPrompt(promptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load factoid extraction prompt: %v", err)
	}

	systemPrompt = strings.Replace(systemPrompt, "{{EDUCATIONAL_CONTENT}}", text, 1)

	log.Printf("Processing text to factoids. Text length: %d characters", len(text))

	req := gopenai.ChatCompletionRequest{
		Model: e.client.GetModel(),
		Messages: []gopenai.ChatCompletionMessage{
			{
				Role:    gopenai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
		},
		ResponseFormat: &gopenai.ChatCompletionResponseFormat{
			Type: gopenai.ChatCompletionResponseFormatTypeJSONObject,
		},
	}

	resp, err := e.client.GetClient().CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion: %v", err)
	}

	responseContent := resp.Choices[0].Message.Content
	log.Printf("Raw response from OpenAI: %s", responseContent)

	if classUUID != "" && sessionID != "" {
		err = db.RecordAIParserHistory(classUUID, sessionID, systemPrompt, responseContent, resp.Usage.TotalTokens, calculateCost(resp.Usage))
		if err != nil {
			log.Printf("warning: failed to record extractor API call in history: %v", err)
		} else {
			log.Printf("successfully recorded extractor API call for session %s", sessionID)
		}
	} else {
		log.Printf("warning: cannot record extractor API call in history: missing classUUID or sessionID")
	}

	factoids, err := parseFactoidsFromResponse(responseContent)
	if err != nil {
		return nil, err
	}

	log.Printf("generated %d factoids from text", len(factoids))
	log.Printf("total tokens: %d", resp.Usage.TotalTokens)

	return factoids, nil
}

func calculateCost(usage gopenai.Usage) float64 {
	return float64(usage.TotalTokens) * 0.000001
}//åaaaaaaaaaaaaaaaaaahahhhhhhhhhhhh

// there are various formats that the API response can take, so we try each one until we find one that works
func parseFactoidsFromResponse(responseContent string) ([]openai.Factoid, error) {
	formats := []struct {
		description string
		parser      func(string) ([]openai.Factoid, error)
	}{
		{"array", parseAsArray},
		{"factoids field", parseAsFactoids},
		{"items field", parseAsItems},
		{"response field", parseAsResponse},
		{"output field", parseAsOutput},
		{"result field", parseAsResult},
		{"data field", parseAsData},
		{"single factoid", parseAsSingle},
	}

	var errors []string
	for _, format := range formats {
		factoids, err := format.parser(responseContent)
		if err == nil && len(factoids) > 0 {
			applyDefaultDifficulty(factoids)
			log.Printf("Successfully parsed %d factoids from %s format", len(factoids), format.description)
			return factoids, nil
		}
		errors = append(errors, fmt.Sprintf("Tried %s: %v", format.description, err))
	}

	return nil, fmt.Errorf("failed to parse factoids from response:\n%s\nraw response: %s",
		strings.Join(errors, "\n"), responseContent)
}

func parseAsArray(content string) ([]openai.Factoid, error) {
	var factoids []openai.Factoid
	err := json.Unmarshal([]byte(content), &factoids)
	if err != nil || len(factoids) == 0 {
		return nil, err
	}
	return factoids, nil
}

func parseAsFactoids(content string) ([]openai.Factoid, error) {
	var response struct {
		Factoids []openai.Factoid `json:"factoids"`
	}
	err := json.Unmarshal([]byte(content), &response)
	if err != nil || len(response.Factoids) == 0 {
		return nil, err
	}
	return response.Factoids, nil
}

func parseAsItems(content string) ([]openai.Factoid, error) {
	var response struct {
		Items []openai.Factoid `json:"items"`
	}
	err := json.Unmarshal([]byte(content), &response)
	if err != nil || len(response.Items) == 0 {
		return nil, err
	}
	return response.Items, nil
}

func parseAsResponse(content string) ([]openai.Factoid, error) {
	var response struct {
		Response []openai.Factoid `json:"response"`
	}
	err := json.Unmarshal([]byte(content), &response)
	if err != nil || len(response.Response) == 0 {
		return nil, err
	}
	return response.Response, nil
}

func parseAsOutput(content string) ([]openai.Factoid, error) {
	var response struct {
		Output []openai.Factoid `json:"output"`
	}
	err := json.Unmarshal([]byte(content), &response)
	if err != nil || len(response.Output) == 0 {
		return nil, err
	}
	return response.Output, nil
}

func parseAsResult(content string) ([]openai.Factoid, error) {
	var response struct {
		Result []openai.Factoid `json:"result"`
	}
	err := json.Unmarshal([]byte(content), &response)
	if err != nil || len(response.Result) == 0 {
		return nil, err
	}
	return response.Result, nil
}

func parseAsSingle(content string) ([]openai.Factoid, error) {
	var factoid openai.Factoid
	err := json.Unmarshal([]byte(content), &factoid)
	if err != nil || factoid.Question == "" || factoid.Answer == "" {
		return nil, err
	}
	return []openai.Factoid{factoid}, nil
}

func parseAsData(content string) ([]openai.Factoid, error) {
	var response struct {
		Data []openai.Factoid `json:"data"`
	}
	err := json.Unmarshal([]byte(content), &response)
	if err != nil || len(response.Data) == 0 {
		return nil, err
	}
	return response.Data, nil
}

func applyDefaultDifficulty(factoids []openai.Factoid) {
	for i := range factoids {
		if factoids[i].Difficulty == 0 {
			factoids[i].Difficulty = 3 //default middle difficulty
		}
	}
}
