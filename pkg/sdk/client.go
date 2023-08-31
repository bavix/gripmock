package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type StubApiClient struct {
	url        string
	httpClient *http.Client
}

func NewStubApiClient(url string, client *http.Client) *StubApiClient {
	return &StubApiClient{url: url, httpClient: client}
}

type Payload struct {
	Service string      `json:"service"`
	Method  string      `json:"method"`
	Data    interface{} `json:"data"`
}

type Response struct {
	Data  interface{} `json:"data"`
	Error string      `json:"error"`
}

func (c *StubApiClient) Search(payload Payload) (*Response, error) {
	postBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(c.url+"/api/stubs/search", "application/json", bytes.NewReader(postBody))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf(string(body))
	}

	result := new(Response)
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()

	if err := decoder.Decode(result); err != nil {
		return nil, err
	}

	return result, nil
}
