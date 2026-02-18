package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    fmt.Sprintf("https://api.telegram.org/bot%s", token),
	}
}

func (c *Client) call(method string, payload interface{}) (json.RawMessage, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/"+method, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("telegram: %s", apiResp.Description)
	}

	return apiResp.Result, nil
}

func (c *Client) SendMessage(chatID int64, text, parseMode string, replyMarkup interface{}) (int64, error) {
	req := SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: parseMode,
	}

	if replyMarkup != nil {
		rm, err := json.Marshal(replyMarkup)
		if err != nil {
			return 0, err
		}
		req.ReplyMarkup = rm
	}

	result, err := c.call("sendMessage", req)
	if err != nil {
		return 0, err
	}

	var msg MessageResult
	json.Unmarshal(result, &msg)
	return msg.MessageID, nil
}

func (c *Client) EditMessageText(chatID, messageID int64, text, parseMode string, replyMarkup interface{}) error {
	req := EditMessageTextRequest{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
		ParseMode: parseMode,
	}

	if replyMarkup != nil {
		rm, err := json.Marshal(replyMarkup)
		if err != nil {
			return err
		}
		req.ReplyMarkup = rm
	}

	_, err := c.call("editMessageText", req)
	return err
}

func (c *Client) AnswerCallbackQuery(callbackID, text string, showAlert bool) error {
	req := AnswerCallbackQueryRequest{
		CallbackQueryID: callbackID,
		Text:            text,
		ShowAlert:       showAlert,
	}
	_, err := c.call("answerCallbackQuery", req)
	return err
}

func (c *Client) SetWebhook(url, secretToken string) error {
	req := SetWebhookRequest{URL: url, SecretToken: secretToken}
	_, err := c.call("setWebhook", req)
	return err
}

func (c *Client) DeleteMessage(chatID, messageID int64) error {
	req := struct {
		ChatID    int64 `json:"chat_id"`
		MessageID int64 `json:"message_id"`
	}{ChatID: chatID, MessageID: messageID}
	_, err := c.call("deleteMessage", req)
	return err
}

func (c *Client) DeleteWebhook() error {
	_, err := c.call("deleteWebhook", struct{}{})
	return err
}
