package externalapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"ghscraper.htm/log"
	"ghscraper.htm/system"
)

var headers = map[string]string{
	"Accept": "application/vnd.github.v3+json",
}

type ExternalAPI interface {
	Get(url string) ([]map[string]interface{}, error)
	ReqCount() int
	BaseURL() string
}

type externalAPI struct {
	baseURL  string
	reqCount int
}

var api ExternalAPI

func NewExternalAPI(baseURL string) ExternalAPI {
	if api == nil {
		api = &externalAPI{baseURL: baseURL, reqCount: 0}
	}

	return api
}

func (a *externalAPI) Get(resourceUrl string) ([]map[string]interface{}, error) {
	log.Info.Printf("Retrieving data from %s\n", resourceUrl)
	client := &http.Client{}
	var respJSON []map[string]interface{}

	req, err := http.NewRequest("GET", resourceUrl, nil)
	if err != nil {
		return nil, err
	}

	for key, val := range headers {
		req.Header.Add(key, val)
	}

	if system.Cfg.BasicAuthToken != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Basic %s", system.Cfg.BasicAuthToken))
	}

	a.reqCount = a.reqCount + 1
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Received non OK http status from third party, status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&respJSON); err != nil {
		return nil, err
	}

	return respJSON, nil
}

func (a *externalAPI) ReqCount() int {
	return a.reqCount
}

func (a *externalAPI) BaseURL() string {
	return a.baseURL
}
