package abdata

import (
	"encoding/json"
	"fmt"
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
	CompoundL    string          `json:"compound_left"`
	CompoundR    string          `json:"compound_right"`
	DosesL       []*CompoundDose `json:"doses_left"`
	DosesR       []*CompoundDose `json:"doses_right"`
}

type Interactions struct {
	Interactions []CompoundInteraction `json:"interactions"`
}

func (j *API) GetCommpoundInteractions(compounds []string) (*Interactions, *queryerr.Error) {

	if !j.AccessValid() {
		err := j.Refresh()
		if err != nil {
			return nil, err
		}
	}

	url := fmt.Sprintf("%s/interactions/compounds/", j.BaseURL)
	url += fmt.Sprintf("?compounds=%s", strings.Join(compounds, ","))

	j.Mutex.Lock()
	authHeader := bearerHeader(j.AccessToken)
	j.Mutex.Unlock()

	body, err := get(url, &authHeader)
	if err != nil {
		return nil, err
	}

	interactions := &Interactions{}
	if err := json.Unmarshal(body, interactions); err != nil {
		return nil, queryerr.NewInternal(fmt.Errorf("cannot unmarshal interactions: %w", err))
	}

	return interactions, nil
}
