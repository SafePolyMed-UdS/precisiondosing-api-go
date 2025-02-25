package abdata

import (
	"encoding/json"
	"fmt"
	"net/url"
	"precisiondosing-api-go/internal/utils/queryerr"
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

func (j *API) GetCommpoundInteractions(compounds []string) ([]CompoundInteraction, *queryerr.Error) {
	if !j.AccessValid() {
		err := j.Refresh()
		if err != nil {
			return nil, err
		}
	}

	compoundStr := strings.Join(compounds, ",")
	compoundStr = url.QueryEscape(compoundStr)
	url := fmt.Sprintf("%s/interactions/compounds/", j.BaseURL)
	url += fmt.Sprintf("?compounds=%s", compoundStr)

	j.Mutex.Lock()
	authHeader := bearerHeader(j.AccessToken)
	j.Mutex.Unlock()

	body, err := get(url, &authHeader)
	if err != nil {
		return nil, err
	}

	type JSendResponse struct {
		Status string                `json:"status"`
		Data   []CompoundInteraction `json:"data"`
	}

	var response JSendResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, queryerr.NewInternal(fmt.Errorf("cannot unmarshal interactions: %w", err))
	}

	return response.Data, nil
}
