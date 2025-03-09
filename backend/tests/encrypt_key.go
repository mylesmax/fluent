package main

import (
	"fmt"
	"os"
	"strings"

	"fluent/backend/openai"
)

//to run dis
//go run backend/tests/encrypt_key.go backend/secret/openai.token backend/secret/openai.key.enc

func main() {
	inputFile := os.Args[1]
	outputFile := os.Args[2]

	keyBytes, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("err reading input file: %v\n", err)
		os.Exit(1)
	}

	apiKey := strings.TrimSpace(string(keyBytes))

	password := "bme4015"

	encryptedKey, err := openai.EncryptKey(apiKey, password)
	if err != nil {
		fmt.Printf("err encrypting api key: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(outputFile, []byte(encryptedKey), 0600)
	if err != nil {
		fmt.Printf("err writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("api key successfully encrypted and saved to", outputFile)
}
