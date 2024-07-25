package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Message represents a chat message
type APIConfig struct {
	OpenAI     string `json:"openai"`
	Groq       string `json:"groq"`
	Perplexity string `json:"perplexity"`
	Google     string `json:"google"`
	Together   string `json:"together"`
}

type ChatModel struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	SVG      string `json:"svg"`
	Key      string `json:"key"`
	Active   bool   `json:"active"`
}

type PromptConfig struct {
	BasePrompt string `json:"base_prompt"`
}

type Config struct {
	API               APIConfig   `json:"api"`
	AllowedChatModels []ChatModel `json:"allowed_chat_models"`
	Prompt            PromptConfig `json:"prompt"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type APIRequest struct {
	Model        string    `json:"model"`
	Messages     []Message `json:"messages"`
	Temperature  float64   `json:"temperature"`
}

type APIResponse struct {
	// Add fields based on the actual API response structure
	// This is a placeholder structure
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	config := Config{
		API: APIConfig{
			OpenAI:     os.Getenv("OPENAI_API_KEY"),
			Groq:       os.Getenv("GROQ_API_KEY"),
			Perplexity: os.Getenv("PERPLEXITY_API_KEY"),
			Google:     os.Getenv("GOOGLE_API_KEY"),
			Together:   os.Getenv("TOGETHER_API_KEY"),
		},
		AllowedChatModels: []ChatModel{
			{ID: 1, Type: "chat", URL: "https://api.openai.com/v1/chat/completions", Name: "GPT-4o Mini", Provider: "openai", SVG: "icons.openai", Key: "gpt-4o-mini", Active: true},
			{ID: 2, Type: "chat", URL: "https://api.openai.com/v1/chat/completions", Name: "GPT-3.5 Turbo", Provider: "openai", SVG: "icons.openai", Key: "gpt-3.5-turbo", Active: true},
			{ID: 3, Type: "chat", URL: "https://api.groq.com/openai/v1/chat/completions", Name: "Llama 3.1 8B", Provider: "groq", SVG: "icons.meta", Key: "llama-3.1-8b-instant", Active: true},
			{ID: 4, Type: "chat", URL: "https://api.groq.com/openai/v1/chat/completions", Name: "Mixtral 8x7B", Provider: "groq", SVG: "icons.mixtral", Key: "mixtral-8x7b-32768", Active: true},
			{ID: 5, Type: "chat", URL: "https://api.groq.com/openai/v1/chat/completions", Name: "Gemma 2 9B", Provider: "groq", SVG: "icons.google", Key: "gemma2-9b-it", Active: true},
			{ID: 6, Type: "chat", URL: "https://api.perplexity.ai/chat/completions", Name: "Llama 3 Online 8B", Provider: "perplexity", SVG: "icons.meta", Key: "llama-3-sonar-small-32k-online", Active: true},
			{ID: 7, Type: "chat", URL: "https://api.perplexity.ai/chat/completions", Name: "Llama 3 Online 70B", Provider: "perplexity", SVG: "icons.meta", Key: "llama-3-sonar-large-32k-online", Active: true},
		},
		Prompt: PromptConfig{
			BasePrompt: `
<instructions>
    - You are a helpful assistant.
    - You are specialised in content writing of any given topic.
    - You are a professional content writer.     
    - Always create a 3 outline and then describe each outline in detail.
    - When describing outline use PEEL (Point, Explain, Example, Link) method. 
    - Always use simple and easy to understand language.
    - Always use proper grammar and punctuation.
    - Always use proper formatting.
    - If any content exist in context, please use it as a reference, only if it is related to the topic.
    - Always share only the answer. and don't share the question.
    - Always use the main content as the main answer and do not use word PEEL and outline.
    - IMPORTANT: please always use <content></context> tag before answering any questions, if using context exists in the input.
    - Do not print <content></context> tag in the main output.
</instructions>

<context></context>

<output_instructions>
    - The main output should be markdown.
</output_instructions>
`,
		},
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		fmt.Println("Error marshalling to JSON:", err)
		return
	}

	// Write to file
	err = os.WriteFile("config.json", jsonData, 0644)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	fmt.Println("Configuration file created successfully: config.json")

	// Prompt the user for a question
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Please enter your question: ")
	question, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}
	question = strings.TrimSpace(question)


	fmt.Print("Do you want to load a custom prompt? (yes/no): ")
	useCustomPrompt, _ := reader.ReadString('\n')
	useCustomPrompt = strings.TrimSpace(strings.ToLower(useCustomPrompt))

	var basePrompt string
	if useCustomPrompt == "yes" {
		fmt.Print("Enter the name of the prompt file (without .txt extension): ")
		promptFileName, _ := reader.ReadString('\n')
		promptFileName = strings.TrimSpace(promptFileName)

		basePrompt, err = loadPromptFromFile(promptFileName)
		if err != nil {
			fmt.Printf("Error loading custom prompt: %v\nUsing default prompt instead.\n", err)
			basePrompt = config.Prompt.BasePrompt
		}
	} else {
		basePrompt = config.Prompt.BasePrompt
	}

	

	runningModelID := 1 // This would be set based on your application logic
	messages := []Message{
		{Role: "system", Content: basePrompt},
		{Role: "user", Content: question},
	}

	response, err := callAPI(config, runningModelID, messages)
	if err != nil {
		fmt.Println("Error calling API:", err)
		return
	}

	err = saveResponseAsMarkdown(question, response, basePrompt)
	if err != nil {
		fmt.Println("Error saving response:", err)
		return
	}

	fmt.Println("Response saved successfully as markdown.")

}


func loadPromptFromFile(fileName string) (string, error) {
	promptsDir := "prompts"
	filePath := filepath.Join(promptsDir, fileName+".txt")

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	return string(content), nil
}


func callAPI(config Config, runningModelID int, messages []Message) (string, error) {
	selectedModel := config.AllowedChatModels[runningModelID-1] // Adjust index if your IDs start from 0

	apiKey := config.API.OpenAI // Default to OpenAI, adjust based on the provider
	switch selectedModel.Provider {
	case "groq":
		apiKey = config.API.Groq
	case "perplexity":
		apiKey = config.API.Perplexity
	case "google":
		apiKey = config.API.Google
	case "together":
		apiKey = config.API.Together
	}

	requestBody := APIRequest{
		Model:       selectedModel.Key,
		Messages:    messages,
		Temperature: 0.7,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshalling request: %v", err)
	}

	req, err := http.NewRequest("POST", selectedModel.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var apiResponse APIResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling response: %v", err)
	}

	// Assuming the API returns at least one choice
	if len(apiResponse.Choices) > 0 {
		return apiResponse.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no content in API response")
}

func saveResponseAsMarkdown(question, content, prompt string) error {
	// Create timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	
	// Create filename with timestamp
	filename := fmt.Sprintf("output/response_%s.md", timestamp)

	// Create the markdown content
	// markdownContent := fmt.Sprintf("# API Response\n\n## Prompt\n\n%s\n\n## Question\n\n%s\n\n## Answer\n\n%s", prompt, question, content)
	markdownContent := fmt.Sprintf("%s", content)
	// Write to file
	err := ioutil.WriteFile(filename, []byte(markdownContent), 0644)
	if err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}

	return nil
}