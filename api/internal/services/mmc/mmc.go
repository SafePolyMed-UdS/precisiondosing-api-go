package mmc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/utils/helper"
	"precisiondosing-api-go/internal/utils/log"
	"precisiondosing-api-go/internal/utils/tokens"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Token struct {
	tokenStr  string
	expiresIn time.Time
}

type PayloadBuilder func(a *API) (any, error)
type ResponseParser func(responseBody []byte) (accessToken string, err error)

type API struct {
	accessToken     *Token         `json:"-"`
	login           string         `json:"-"`
	password        string         `json:"-"`
	loginURL        string         `json:"-"`
	sendURL         string         `json:"-"`
	expiryThreshold time.Duration  `json:"-"`
	pdfPrefix       string         `json:"-"`
	mutex           sync.Mutex     `json:"-"`
	logger          log.Logger     `json:"-"`
	mockSend        bool           `json:"-"`
	productionSpec  bool           `json:"-"`
	payloadBuilder  PayloadBuilder `json:"-"`
	responseParser  ResponseParser `json:"-"`
}

func NewAPI(config cfg.MMCConfig) *API {
	payloadBuilder := debugPayload
	responseParser := debugResponseParser
	if config.ProductionSpec {
		payloadBuilder = productionPayload
		responseParser = productionResponseParser
	}

	logger := log.WithComponent("mmc")
	if config.MockSend {
		logger.Warn("mock send enabled")
	}

	if !config.ProductionSpec {
		logger.Warn("API debug spec enabled")
	}

	return &API{
		login:           config.Login,
		password:        config.Password,
		loginURL:        config.AuthEndpoint,
		sendURL:         config.ResultEndpoint,
		expiryThreshold: config.ExpiryThreshold,
		pdfPrefix:       config.PDFPrefix,
		mockSend:        config.MockSend,
		productionSpec:  config.ProductionSpec,
		logger:          logger,
		payloadBuilder:  payloadBuilder,
		responseParser:  responseParser,
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

	if a.mockSend {
		return nil
	}

	sendURL := helper.RemoveTrailingSlash(a.sendURL) + "/" + orderID
	_, err = post(sendURL, &buf, writer.FormDataContentType(), &bearerToken)
	if err != nil {
		return fmt.Errorf("failed to send to MMC: %w", err)
	}

	a.logger.Info("file sent to MMC", log.Str("order_id", orderID))
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

	err := a.authenticate(a.payloadBuilder, a.responseParser)
	if err != nil {
		return err
	}

	a.logger.Info("token refreshed", log.Str("expires", time.Until(a.accessToken.expiresIn).String()))
	return nil
}

func bearerHeader(token string) string {
	return fmt.Sprintf("Bearer %s", token)
}

func (a *API) authenticate(buildPayload PayloadBuilder, parseResponse ResponseParser) error {
	payload, err := buildPayload(a)
	if err != nil {
		return fmt.Errorf("failed to build payload: %w", err)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	responseBody, postErr := post(a.loginURL, bytes.NewReader(body), "application/json", nil)
	if postErr != nil {
		return fmt.Errorf("failed to authenticate: %w", postErr)
	}

	accessToken, err := parseResponse(responseBody)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	a.accessToken, err = processToken(accessToken, &a.logger)
	if err != nil {
		return fmt.Errorf("failed to process access token: %w", err)
	}
	return nil
}

// Debugging functions for local testing
func debugPayload(a *API) (any, error) {
	return map[string]string{"login": a.login, "password": a.password}, nil
}

func debugResponseParser(responseBody []byte) (string, error) {
	type Response struct {
		Status string `json:"status"`
		Data   struct {
			tokens.AuthTokens
		} `json:"data"`
	}

	var tmp Response
	if err := json.Unmarshal(responseBody, &tmp); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	if tmp.Status != "success" {
		return "", fmt.Errorf("unexpected status: %s", tmp.Status)
	}
	return tmp.Data.AccessToken, nil
}

// Production functions for actual use
func productionPayload(a *API) (any, error) {
	return map[string]string{"name": a.login, "password": a.password}, nil
}

func productionResponseParser(responseBody []byte) (string, error) {
	type Response struct {
		Code    int `json:"mmc_status_code"`
		Payload struct {
			AccessToken string `json:"token"`
		} `json:"mmc_payload"`
	}

	var tmp Response
	if err := json.Unmarshal(responseBody, &tmp); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	if tmp.Code != 200 {
		return "", fmt.Errorf("unexpected code: %d", tmp.Code)
	}
	return tmp.Payload.AccessToken, nil
}

func processToken(tokenStr string, logger *log.Logger) (*Token, error) {
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
		logger.Warn("exp not found in token claims")
	}

	expFloat, ok := exp.(float64)
	if !ok {
		return nil, errors.New("exp is not a float64")
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
