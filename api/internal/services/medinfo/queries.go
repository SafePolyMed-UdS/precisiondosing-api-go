package medinfo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type CompoundDose struct {
	Value               *float64 `json:"value"`
	Unit                *string  `json:"unit"`
	Suffix              *string  `json:"suffix"`
	DosageForm          *string  `json:"dosage_form"`
	EquivalentSubstance bool     `json:"equivalent_substance"`
}

type CompoundInteraction struct {
	Plausibility *string         `json:"plausibility"`
	Relevance    *string         `json:"relevance"`
	Frequency    *string         `json:"frequency"`
	Credibility  *string         `json:"credibility"`
	Direction    *string         `json:"direction"`
	CompoundsL   []string        `json:"compounds_left"`
	CompoundsR   []string        `json:"compounds_right"`
	DosesL       []*CompoundDose `json:"doses_left"`
	DosesR       []*CompoundDose `json:"doses_right"`
}

type Error struct {
	StatusCode int
	Err        error
	InputError bool
}

func (e *Error) Error() string {
	return fmt.Sprintf("Error with status code %d: %s", e.StatusCode, e.Err.Error())
}

func newError(statusCode int, e error, inputError bool) *Error {
	return &Error{
		StatusCode: statusCode,
		Err:        e,
		InputError: inputError,
	}
}

func extractErrorMsgFromJSend(body []byte) string {
	type jSendError struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	type errorResponse struct {
		Error string `json:"error"`
	}

	type jsendFailure struct {
		Status string        `json:"status"`
		Data   errorResponse `json:"data"`
	}

	var jsError jSendError
	if err := json.Unmarshal(body, &jsError); err == nil {
		return jsError.Message
	}

	var jsFailure jsendFailure
	if err := json.Unmarshal(body, &jsFailure); err == nil {
		return jsFailure.Data.Error
	}

	return "Unknown error"
}

func extractJSendResponse[T any](body []byte) (*T, error) {
	type jSendResponse[T any] struct {
		Status string `json:"status"`
		Data   T      `json:"data"`
	}

	var response jSendResponse[T]
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("cannot unmarshal response: %w", err)
	}
	return &response.Data, nil
}

func (a *API) GetCommpoundInteractions(compounds []string) ([]CompoundInteraction, *Error) {
	if !a.AccessValid() {
		err := a.Refresh()
		if err != nil {
			return nil, newError(err.StatusCode, fmt.Errorf("failed to authenticate: %w", err), false)
		}
	}

	compoundStr := strings.Join(compounds, ",")
	compoundStr = url.QueryEscape(compoundStr)
	url := fmt.Sprintf("%s/interactions/compounds/", a.baseURL)
	url += fmt.Sprintf("?compounds=%s", compoundStr)

	a.mutex.Lock()
	authHeader := bearerHeader(a.AccessToken)
	a.mutex.Unlock()

	body, err := get(url, &authHeader)
	if err != nil {
		sc := err.StatusCode
		errMsg := extractErrorMsgFromJSend(err.Message)
		return nil, newError(sc, errors.New(errMsg), sc == http.StatusNotFound)
	}

	data, e := extractJSendResponse[[]CompoundInteraction](body)
	if e != nil {
		return nil, newError(http.StatusInternalServerError, e, false)
	}

	return *data, nil
}

type CompoundResponse struct {
	Name      string   `json:"name"`
	Standards []string `json:"standards"`
	Preferred bool     `json:"preferred"`
}

type CompoundMatch struct {
	Input   string               `json:"input"`
	Matches [][]CompoundResponse `json:"matches"`
}

func (a *API) GetCommpoundSynonyms(compounds []string) ([]CompoundMatch, *Error) {
	if !a.AccessValid() {
		err := a.Refresh()
		if err != nil {
			return nil, newError(err.StatusCode, fmt.Errorf("failed to authenticate: %w", err), false)
		}
	}

	compoundStr := strings.Join(compounds, ",")
	compoundStr = url.QueryEscape(compoundStr)
	url := fmt.Sprintf("%s/compounds/names/", a.baseURL)
	url += fmt.Sprintf("?names=%s", compoundStr)

	a.mutex.Lock()
	authHeader := bearerHeader(a.AccessToken)
	a.mutex.Unlock()

	body, err := get(url, &authHeader)
	if err != nil {
		sc := err.StatusCode
		errMsg := extractErrorMsgFromJSend(err.Message)
		return nil, newError(sc, errors.New(errMsg), false)
	}

	data, e := extractJSendResponse[[]CompoundMatch](body)
	if e != nil {
		return nil, newError(http.StatusInternalServerError, e, false)
	}

	d := *data
	// !!! We have to check if compounds are not found .. the API endpoint does not check that
	// not like the interaction endpoint
	for _, c := range compounds {
		found := false
		for _, m := range d {
			if m.Input == c && len(m.Matches) > 0 {
				found = true
				break
			}
		}
		if !found {
			return nil, newError(http.StatusNotFound, fmt.Errorf("compound %s not in database", c), true)
		}
	}

	return d, nil
}
