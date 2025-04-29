package providers

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type OraClient struct {
	APIKey  string
	BaseURL string
}

func NewOraClient(apiKey string) *OraClient {
	return &OraClient{APIKey: apiKey, BaseURL: "https://api.ora.ai/v1"} // Adjust per ORA docs
}

func (c *OraClient) Complete(prompt string) (string, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"prompt": prompt,
		"model":  "deepseek-v3",
	})
	req, _ := http.NewRequest("POST", c.BaseURL+"/completions", bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Text string `json:"text"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) > 0 {
		return result.Choices[0].Text, nil
	}
	return "No proposal generated", nil
}
