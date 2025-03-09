package extraction

import (
	"context"
	"fluent/backend/openai"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func setupTestClient(t *testing.T) *openai.OpenAIClient {
	encryptedKeyFile := "../secret/openai.key.enc"
	tokenFile := "../secret/openai.token"

	if _, err := os.Stat(encryptedKeyFile); err == nil {
		password := os.Getenv("OPENAI_KEY_PASSWORD")
		if password == "" {
			passwordFile := os.Getenv("OPENAI_KEY_PASSWORD_FILE")
			if passwordFile != "" {
				passwordBytes, err := os.ReadFile(passwordFile)
				if err != nil {
					t.Fatalf("Failed to read password file: %v", err)
				}
				password = strings.TrimSpace(string(passwordBytes))
			}
		}

		if password == "" {
			password = "bme4015"
		}

		client, err := openai.NewOpenAIClientWithPassword(encryptedKeyFile, password)
		if err != nil {
			t.Fatalf("Failed to create OpenAI client with password: %v", err)
		}
		return client
	} else {
		if _, err := os.Stat(tokenFile); err == nil {
			apiKeyBytes, err := os.ReadFile(tokenFile)
			if err != nil {
				t.Fatalf("Failed to read token file: %v", err)
			}
			apiKey := strings.TrimSpace(string(apiKeyBytes))

			os.Setenv("OPENAI_API_KEY", apiKey)

			client, err := openai.NewOpenAIClient()
			if err != nil {
				t.Fatalf("Failed to create OpenAI client with API key: %v", err)
			}
			return client
		}
	}

	if os.Getenv("OPENAI_API_KEY") != "" {
		client, err := openai.NewOpenAIClient()
		if err != nil {
			t.Fatalf("Failed to create OpenAI client with environment API key: %v", err)
		}
		return client
	}

	t.Fatal("no api key found")
	return nil
}

func TestFactoidExtraction(t *testing.T) {
	client := setupTestClient(t)
	extractor := NewExtractor(client)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sampleTextBytes, err := os.ReadFile("../secret/testdata/olfaction.txt")
	if err != nil {
		t.Fatalf("Failed to read olfaction.txt: %v", err)
	}
	sampleText := string(sampleTextBytes)

	fmt.Println("processing olfaction text to factoids...")
	factoids, err := extractor.ProcessTextToFactoids(ctx, sampleText)
	if err != nil {
		t.Fatalf("failed to process text to factoids: %v", err)
	}

	if len(factoids) == 0 {
		t.Fatalf("No factoids extracted from text")
	}

	fmt.Printf("Successfully extracted %d factoids:\n\n", len(factoids))

	for i, f := range factoids {
		if err := f.Validate(); err != nil {
			t.Errorf("Factoid %d is invalid: %v", i+1, err)
			continue
		}

		jsonData, err := f.ToJSON()
		if err != nil {
			t.Errorf("Failed to convert factoid %d to JSON: %v", i+1, err)
			continue
		}

		fmt.Printf("Factoid %d: %s\n", i+1, jsonData)

		if f.Question == "" {
			t.Errorf("Factoid %d has no question", i+1)
		}
		if f.Answer == "" {
			t.Errorf("Factoid %d has no answer", i+1)
		}
		if f.Type != "factual" && f.Type != "conceptual" {
			t.Errorf("Factoid %d has invalid type: %s", i+1, f.Type)
		}
		if f.Difficulty < 1 || f.Difficulty > 5 {
			t.Errorf("Factoid %d has invalid difficulty: %d", i+1, f.Difficulty)
		}
	}
}
