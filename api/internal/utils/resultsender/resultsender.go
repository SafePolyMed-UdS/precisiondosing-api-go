package resultsender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"precisiondosing-api-go/internal/utils/queryerr"
	"precisiondosing-api-go/internal/utils/tokens"
	"sync"
	"time"
)

type Sender struct {
	accessToken string     `json:"-"`
	login       string     `json:"-"`
	password    string     `json:"-"`
	loginURL    string     `json:"-"`
	sendURL     string     `json:"-"`
	mutex       sync.Mutex `json:"-"`
}

func New(login, password, loginURL, sendURL string) *Sender {
	return &Sender{
		login:    login,
		password: password,
		loginURL: loginURL,
		sendURL:  sendURL,
	}
}

func (j *Sender) Send() {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	err := j.authenticate()
	if err != nil {
		fmt.Println("Authentication error:", err)
		return
	}
}

func (j *Sender) authenticate() *queryerr.Error {
	payload := map[string]string{"login": j.login, "password": j.password}

	url := j.loginURL
	body, err := json.Marshal(payload)
	if err != nil {
		return queryerr.NewInternal(fmt.Errorf("failed to marshal payload: %w", err))
	}

	responseBody, postErr := post(url, body, nil)
	if postErr != nil {
		return queryerr.NewInternal(fmt.Errorf("failed to authenticate: %w", postErr))
	}

	type JSendResponse struct {
		Status string `json:"status"`
		Data   struct {
			tokens.AuthTokens
			Role      string     `json:"role"`
			LastLogin *time.Time `json:"last_login"`
		} `json:"data"`
	}

	// Unmarshal JSON into a temporary variable
	var tmp JSendResponse
	if err = json.Unmarshal(responseBody, &tmp); err != nil {
		return queryerr.NewInternal(fmt.Errorf("failed to unmarshal JSON: %w", err))
	}

	// Check if response status is not "success"
	if tmp.Status != "success" {
		return queryerr.NewInternal(fmt.Errorf("unexpected status: %s", tmp.Status))
	}

	j.accessToken = tmp.Data.AuthTokens.AccessToken
	return nil
}

func post(url string, body []byte, authHeader *string) ([]byte, *queryerr.Error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, queryerr.NewInternal(fmt.Errorf("failed to create request: %w", err))
	}

	req.Header.Set("Content-Type", "application/json")
	if authHeader != nil {
		req.Header.Set("Authorization", *authHeader)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, queryerr.NewInternal(fmt.Errorf("failed to make request: %w", err))
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	// Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		errBody := bodyBytes
		if len(errBody) == 0 {
			errBody = []byte("Error body not found")
		}
		return nil, queryerr.NewHTTTP(resp.StatusCode, errBody)
	}

	if err != nil {
		return nil, queryerr.NewInternal(fmt.Errorf("failed to read response body: %w", err))
	}

	return bodyBytes, nil
}
