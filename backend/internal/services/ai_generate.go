package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AIGenerateService struct {
	httpClient *http.Client
	apiKey     string
	apiURL     string
	model      string
}

func NewAIGenerateService(apiKey, apiURL, model string) *AIGenerateService {
	return &AIGenerateService{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		apiKey:     apiKey,
		apiURL:     apiURL,
		model:      model,
	}
}

func (s *AIGenerateService) IsAvailable() bool {
	return s.apiKey != ""
}

type aiQuizResponse struct {
	Title      string         `json:"title"`
	Categories []aiCategory   `json:"categories"`
}

type aiCategory struct {
	Title     string       `json:"title"`
	Questions []aiQuestion `json:"questions"`
}

type aiQuestion struct {
	Text    string     `json:"text"`
	Options []aiOption `json:"options"`
}

type aiOption struct {
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
	Color     string `json:"color"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const systemPrompt = `You are a quiz generator. The user will describe what kind of quiz they want. You must respond with ONLY valid JSON (no markdown, no code fences, no explanations) in the following format:

{
  "title": "Quiz title in the language of the user's prompt",
  "categories": [
    {
      "title": "Category name",
      "questions": [
        {
          "text": "Question text?",
          "options": [
            {"text": "Option A", "is_correct": true, "color": "#e21b3c"},
            {"text": "Option B", "is_correct": false, "color": "#1368ce"},
            {"text": "Option C", "is_correct": false, "color": "#d89e00"},
            {"text": "Option D", "is_correct": false, "color": "#26890c"}
          ]
        }
      ]
    }
  ]
}

Rules:
- Generate 2-5 categories with 3-6 questions each (unless the user specifies otherwise)
- Each question must have 2 to 4 options
- Exactly one option must have "is_correct": true
- Use varied hex colors for options. Pick from: #e21b3c, #1368ce, #d89e00, #26890c, #864cbf, #0aa3b1, #e67e22, #e84393
- Make questions engaging, varied in difficulty, and factually accurate
- Write everything in the same language as the user's prompt
- Return ONLY the JSON object, nothing else`

func (s *AIGenerateService) GenerateQuiz(prompt string) (*ImportInput, string, error) {
	if !s.IsAvailable() {
		return nil, "", fmt.Errorf("AI generation is not configured")
	}

	reqBody := chatRequest{
		Model: s.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.apiURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, "", fmt.Errorf("failed to parse API response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, "", fmt.Errorf("empty response from AI")
	}

	content := chatResp.Choices[0].Message.Content
	content = cleanJSONContent(content)

	var quizData aiQuizResponse
	if err := json.Unmarshal([]byte(content), &quizData); err != nil {
		return nil, "", fmt.Errorf("AI returned invalid JSON: %w", err)
	}

	input := convertToImportInput(quizData)
	return &input, quizData.Title, nil
}

func cleanJSONContent(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
	}
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
	}
	if strings.HasSuffix(content, "```") {
		content = strings.TrimSuffix(content, "```")
	}
	return strings.TrimSpace(content)
}

func convertToImportInput(data aiQuizResponse) ImportInput {
	input := ImportInput{}
	for _, cat := range data.Categories {
		ic := ImportCategory{Title: cat.Title}
		for _, q := range cat.Questions {
			iq := ImportQuestion{Text: q.Text}
			for _, o := range q.Options {
				iq.Options = append(iq.Options, OptionInput{
					Text:      o.Text,
					IsCorrect: o.IsCorrect,
					Color:     o.Color,
				})
			}
			ic.Questions = append(ic.Questions, iq)
		}
		input.Categories = append(input.Categories, ic)
	}
	return input
}
