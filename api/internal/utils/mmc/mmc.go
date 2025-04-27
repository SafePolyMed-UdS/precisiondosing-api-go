package mmc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"precisiondosing-api-go/internal/utils/tokens"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Token struct {
	tokenStr  string
	expiresIn time.Time
}

type API struct {
	accessToken     *Token        `json:"-"`
	login           string        `json:"-"`
	password        string        `json:"-"`
	loginURL        string        `json:"-"`
	sendURL         string        `json:"-"`
	expiryThreshold time.Duration `json:"-"`
	pdfPrefix       string        `json:"-"`
	mutex           sync.Mutex    `json:"-"`
}

func NewAPI(login, password, loginURL, sendURL, pdfPrefix string, expiryThreshold time.Duration) *API {
	return &API{
		login:           login,
		password:        password,
		loginURL:        loginURL,
		sendURL:         sendURL,
		expiryThreshold: expiryThreshold,
		pdfPrefix:       pdfPrefix,
	}
}

func (a *API) Send(pdf []byte, orderID string) error {
	err := a.RefreshToken()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	a.mutex.Lock()
	bearerToken := bearerHeader(a.accessToken.tokenStr)
	a.mutex.Unlock()

	// Form data:
	// file: <file>
	// order_id: <order_id>
	const orderIDField = "order_id"
	const fileField = "file"
	fileName := a.pdfPrefix + "-" + orderID + ".pdf"

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Order ID
	err = writer.WriteField(orderIDField, orderID)
	if err != nil {
		return fmt.Errorf("failed to write order ID to form: %w", err)
	}

	// File
	formFile, err := writer.CreateFormFile(fileField, fileName)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = formFile.Write(pdf)
	if err != nil {
		return fmt.Errorf("failed to write file to form: %w", err)
	}
	writer.Close()

	sendURL := a.sendURL + "/" + orderID
	_, err = post(sendURL, &buf, writer.FormDataContentType(), &bearerToken)
	if err != nil {
		return fmt.Errorf("failed to send to MMC: %w", err)
	}

	return nil
}

func (a *API) RefreshToken() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	timeThreshold := time.Now().Add(a.expiryThreshold)

	hasToken := a.accessToken != nil
	expired := true
	if hasToken {
		expired = timeThreshold.After(a.accessToken.expiresIn)
	}

	if hasToken && !expired {
		return nil
	}

	err := a.authenticate()
	if err != nil {
		return err
	}
	return nil
}

func bearerHeader(token string) string {
	return fmt.Sprintf("Bearer %s", token)
}

func (a *API) authenticate() error {
	// TODO: CHANGE THIS TO MMC version
	payload := map[string]string{"login": a.login, "password": a.password}

	url := a.loginURL
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	responseBody, postErr := post(url, bytes.NewReader(body), "application/json", nil)
	if postErr != nil {
		return fmt.Errorf("failed to authenticate: %w", postErr)
	}

	// TODO: CHANGE THIS TO MMC version
	type Response struct {
		Status string `json:"status"`
		Data   struct {
			tokens.AuthTokens
		} `json:"data"`
	}

	// Unmarshal JSON into a temporary variable
	var tmp Response
	if err = json.Unmarshal(responseBody, &tmp); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Check if response status is not "success"
	if tmp.Status != "success" {
		return fmt.Errorf("unexpected status: %s", tmp.Status)
	}

	a.accessToken, err = processToken(tmp.Data.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to process access token: %w", err)
	}
	return nil
}

func processToken(tokenStr string) (*Token, error) {
	claims := jwt.MapClaims{}
	_, _, err := new(jwt.Parser).ParseUnverified(tokenStr, claims)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	res := &Token{
		tokenStr:  tokenStr,
		expiresIn: time.Now(),
	}

	exp, ok := claims["exp"]
	if !ok {
		// TODO: LOG THAT EXP IS NOT FOUND -> WARNING
	}

	expFloat, ok := exp.(float64)
	if !ok {
		// TODO:LOG THAT EXP IS NOT FLOAT -> ERROR
	}

	res.expiresIn = time.Unix(int64(expFloat), 0)

	return res, nil
}

func post(url string, body io.Reader, contentType string, authHeader *string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if authHeader != nil {
		req.Header.Set("Authorization", *authHeader)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		errBody := bodyBytes
		if len(errBody) == 0 {
			errBody = []byte("Error body not found")
		}
		return nil, fmt.Errorf("error response: %s", errBody)
	}

	return bodyBytes, nil
}
