package openai

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
)

func setupTestClient(t *testing.T) *OpenAIClient {
	encryptedKeyFile := "../secret/openai.key.enc"
	tokenFile := "../secret/openai.token"

	if _, err := os.Stat(encryptedKeyFile); err == nil {
		password :=  "bme4015"

		client, err := NewOpenAIClientWithPassword(encryptedKeyFile, password)
		if err != nil {
			t.Fatalf("failed to create OpenAI client with password: %v", err)
		}
		return client
	} else {
		if _, err := os.Stat(tokenFile); err == nil {
			apiKeyBytes, err := os.ReadFile(tokenFile)
			if err != nil {
				t.Fatalf("failed to read token file: %v", err)
			}
			apiKey := strings.TrimSpace(string(apiKeyBytes))

			os.Setenv("OPENAI_API_KEY", apiKey)

			client, err := NewOpenAIClient()
			if err != nil {
				t.Fatalf("failed to create OpenAI client: %v", err)
			}
			return client
		}
	}
	
	if os.Getenv("OPENAI_API_KEY") != "" {
		client, err := NewOpenAIClient()
		if err != nil {
			t.Fatalf("failed to create OpenAI client with environment API key: %v", err)
		}
		return client
	}

	t.Fatal("didn't find api key")
	return nil
}

func TestOpenAIConnection(t *testing.T) {
	client := setupTestClient(t)

	if client == nil {
		t.Fatal("Client is nil")
	}

	if client.GetClient() == nil {
		t.Fatal("Client.GetClient() is nil")
	}

	fmt.Printf("model: %s\n", client.GetModel())
}

func TestSimpleMessage(t *testing.T) {
	client := setupTestClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := openai.ChatCompletionRequest{
		Model: client.GetModel(),
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "give a short response.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "what is the capital of france",
			},
		},
	}

	resp, err := client.GetClient().CreateChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("failed to create chat completion: %v", err)
	}

	fmt.Printf("response: %s\n", resp.Choices[0].Message.Content)

	if !containsIgnoreCase(resp.Choices[0].Message.Content, "Paris") {
		t.Errorf("response does not contain 'Paris': %s", resp.Choices[0].Message.Content)
	}
}

func containsIgnoreCase(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	return strings.Contains(s, substr)
}

type ChatCompletionRequest struct {
	Model          string                        `json:"model"`
	Messages       []ChatCompletionMessage       `json:"messages"`
	MaxTokens      int                           `json:"max_tokens,omitempty"`
	Temperature    float32                       `json:"temperature,omitempty"`
	Stream         bool                          `json:"stream,omitempty"`
	ResponseFormat *ChatCompletionResponseFormat `json:"response_format,omitempty"`
}

type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponseFormat struct {
	Type string `json:"type"`
}

const (
	ChatMessageRoleSystem    = "system"
	ChatMessageRoleUser      = "user"
	ChatMessageRoleAssistant = "assistant"
)
