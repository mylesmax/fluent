package openai

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client *openai.Client
	model  string
}

// encryption logic for encrypting api key w/ password
func EncryptKey(apiKey, password string) (string, error) {
	key := sha256.Sum256([]byte(password))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	//NONCE!
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	//seal makes the key encrypted
	ciphertext := gcm.Seal(nonce, nonce, []byte(apiKey), nil)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryption logic given password
func DecryptKey(encryptedKey, password string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		return "", err
	}

	key := sha256.Sum256([]byte(password))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	//NONCE!
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	//open breaks the seal
	p, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(p), nil
}

// make file for encrypted key
func CreateEncryptedKeyFile(apiKey, password, filePath string) error {
	encryptedKey, err := EncryptKey(apiKey, password)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(encryptedKey), 0600)
}

// combine this all such that we can make a new client with a password
func NewOpenAIClientWithPassword(keyFilePath, password string) (*OpenAIClient, error) {
	encryptedKey, err := ioutil.ReadFile(keyFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %v", err)
	}

	apiKey, err := DecryptKey(string(encryptedKey), password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API key: %v", err)
	}

	client := openai.NewClient(apiKey)

	return &OpenAIClient{
		client: client,
		model:  "o3-mini", //thinking model , cost is pretty expensive, bme dept please bless up
	}, nil
}

// todo maybe delete this at some pt, change it, i think there's a vulnerability here with storing as env var
func NewOpenAIClient() (*OpenAIClient, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY environment variable is not set")
	}

	client := openai.NewClient(apiKey)

	return &OpenAIClient{
		client: client,
		model:  "o3-mini",
	}, nil
}

func (c *OpenAIClient) GetClient() *openai.Client {
	return c.client
}

func (c *OpenAIClient) GetModel() string {
	return c.model
}

func DeterminePromptsPath() string {
	//this is a bit of a mess, but it works
	//such a headache to get the proper path with all the test directories, wails, main directories, etc
	paths := []string{
		"secret/prompts",
		"../secret/prompts",
		"../../backend/secret/prompts",
		"backend/secret/prompts",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return "../secret/prompts"
}

func LoadPrompt(promptPath string) (string, error) {
	promptBytes, err := os.ReadFile(promptPath)
	if err != nil {
		return "", err
	}
	return string(promptBytes), nil
}

func EstimateTokens(text string) int {
	//estimate: ~4 characters per token, will know when we get response, this is kinda just extra
	return len(text) / 4
}
