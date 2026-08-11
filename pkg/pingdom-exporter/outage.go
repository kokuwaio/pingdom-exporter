package pingdom

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
)

// OutageSummaryService provides an interface to Pingdom outage summary.
type OutageSummaryService struct {
	client *Client
}

// List returns a list of outage summaries from Pingdom.
func (os *OutageSummaryService) List(checkID int, params ...map[string]string) ([]OutageSummaryResponseState, error) {
	param := map[string]string{}
	if len(params) == 1 {
		param = params[0]
	}

	req, err := os.client.NewRequest("GET", fmt.Sprintf("/summary.outage/%d", checkID), param)
	if err != nil {
		return nil, err
	}

	resp, err := os.client.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return nil, err
	}

	// Check status code and parse error if needed
	if c := resp.StatusCode; !(200 <= c && c <= 299) {
		m := &errorJSONResponse{}
		json.Unmarshal(bodyBytes, &m)
		if m.Error != nil {
			return nil, m.Error
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	bodyString := string(bodyBytes)
	m := &listOutageSummaryJSONResponse{}
	err = json.Unmarshal([]byte(bodyString), &m)

	return m.Summary.States, err
}
