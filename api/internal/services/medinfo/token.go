package medinfo

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

type API struct {
	tokens.AuthTokens
	Role            string        `json:"role"`
	LastLogin       *time.Time    `json:"last_login"`
	baseURL         string        `json:"-"`
	login           string        `json:"-"`
	password        string        `json:"-"`
	expiryThreshold time.Duration `json:"-"`
	mutex           sync.Mutex    `json:"-"`
}

func NewAPI(baseURL, login, password string, expiryThreshold time.Duration) *API {
	return &API{
		baseURL:         baseURL,
		login:           login,
		password:        password,
		expiryThreshold: expiryThreshold,
	}
}

func (a *API) AccessValid() bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.AccessToken != "" && a.AccessExpiresIn.After(time.Now())
}

func (a *API) RefreshValid() bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.RefreshToken != "" && a.RefreshExpiresIn.After(time.Now())
}

func (a *API) Refresh() *queryerr.Error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// try to refresh the access token if it is expired
	timeThreshold := time.Now().Add(a.expiryThreshold)
	if a.RefreshToken != "" && a.RefreshExpiresIn.After(timeThreshold) {
		err := refresh(a)
		if err != nil {
			return err
		}
		return nil
	}

	// try to login if the refresh token is expired
	err := login(a)
	if err != nil {
		return err
	}

	return nil
}

func authenticate(a *API, endpoint string, payload any, authHeader *string) *queryerr.Error {
	url := a.baseURL + endpoint
	body, err := json.Marshal(payload)
	if err != nil {
		return queryerr.NewInternal(fmt.Errorf("failed to marshal payload: %w", err))
	}

	responseBody, postErr := post(url, body, authHeader)
	if postErr != nil {
		return queryerr.NewInternal(fmt.Errorf("failed to post to %s: %w", endpoint, postErr))
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

	a.AuthTokens = tmp.Data.AuthTokens
	a.Role = tmp.Data.Role
	a.LastLogin = tmp.Data.LastLogin

	return nil
}

func refresh(a *API) *queryerr.Error {
	payload := map[string]string{"refresh_token": a.RefreshToken}
	authHeader := bearerHeader(a.AccessToken)
	return authenticate(a, "/user/refresh-token", payload, &authHeader)
}

func login(a *API) *queryerr.Error {
	payload := map[string]string{"login": a.login, "password": a.password}
	return authenticate(a, "/user/login", payload, nil)
}

func bearerHeader(token string) string {
	return fmt.Sprintf("Bearer %s", token)
}

func post(url string, body []byte, authHeader *string) ([]byte, *queryerr.Error) {
	return request(http.MethodPost, url, body, authHeader)
}

func get(url string, authHeader *string) ([]byte, *queryerr.Error) {
	return request(http.MethodGet, url, nil, authHeader)
}

func request(method, url string, body []byte, authHeader *string) ([]byte, *queryerr.Error) {
	req, err := http.NewRequestWithContext(context.Background(), method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, queryerr.NewInternal(fmt.Errorf("failed to create %s request: %w", method, err))
	}

	req.Header.Set("Content-Type", "application/json")
	if authHeader != nil {
		req.Header.Set("Authorization", *authHeader)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, queryerr.NewInternal(fmt.Errorf("failed to make %s request: %w", method, err))
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
		return nil, queryerr.NewInternal(fmt.Errorf("failed to read %s response body: %w", method, err))
	}

	return bodyBytes, nil
}
