package abdata

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
	Role      string     `json:"role"`
	LastLogin *time.Time `json:"last_login"`
	BaseURL   string
	Login     string
	Password  string
	Mutex     sync.Mutex
}

func NewJWT(baseURL, login, password string) *API {
	return &API{
		BaseURL:  baseURL,
		Login:    login,
		Password: password,
	}
}

func (j *API) AccessValid() bool {
	j.Mutex.Lock()
	defer j.Mutex.Unlock()
	return j.AccessToken != "" && j.AccessExpiresIn.After(time.Now())
}

func (j *API) RefreshValid() bool {
	j.Mutex.Lock()
	defer j.Mutex.Unlock()
	return j.RefreshToken != "" && j.RefreshExpiresIn.After(time.Now())
}

func (j *API) Refresh() *queryerr.Error {
	j.Mutex.Lock()
	defer j.Mutex.Unlock()

	// try to refresh the access token if it is expired
	if j.RefreshToken != "" && j.RefreshExpiresIn.After(time.Now()) {

		err := refresh(j)
		if err != nil {
			return err
		}
		return nil
	}

	// try to login if the refresh token is expired
	err := login(j)
	if err != nil {
		return err
	}

	return nil
}

func refresh(j *API) *queryerr.Error {
	url := j.BaseURL + "/user/refresh-token"
	body := []byte(fmt.Sprintf(`{"refresh_token": "%s"}`, j.RefreshToken))

	authHeader := bearerHeader(j.AccessToken)
	body, err := post(url, body, &authHeader)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, j); err != nil { //nolint:musttag // need additional non-json fields
		return queryerr.NewInternal(fmt.Errorf("failed to unmarshal JSON: %w", err))
	}

	return nil
}

func login(j *API) *queryerr.Error {
	url := j.BaseURL + "/user/login"
	body := []byte(fmt.Sprintf(`{"login": "%s", "password": "%s"}`, j.Login, j.Password))
	body, err := post(url, body, nil)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, j); err != nil { //nolint:musttag // need additional non-json fields
		return queryerr.NewInternal(fmt.Errorf("failed to unmarshal JSON: %w", err))
	}

	return nil
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
