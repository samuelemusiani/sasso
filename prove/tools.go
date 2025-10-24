package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type Records struct {
	Content  string `json:"content"`
	Disabled bool   `json:"disabled"`
}

type RRSet struct {
	Name     string    `json:"name"`
	Records  []Records `json:"records"`
	Type     string    `json:"type"`
	TTL      int       `json:"ttl"`
	Comments []string  `json:"comment"`
}

type RecordsResponse struct {
	Name   string  `json:"name"`
	RRSets []RRSet `json:"rrsets"`
}

type Network struct {
	Network string `json:"network"`
	View    string `json:"view"`
}

type NetworksResponse struct {
	Networks []Network `json:"networks"`
}

type Views struct {
	Views []string `json:"views"`
}

func HttpRequest(method, url string, body interface{}) ([]byte, int, error) {
	//check body format
	var bodyReader io.Reader
	if method == "POST" || method == "PUT" || method == "PATCH" {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, -1, fmt.Errorf("failed reading body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	fmt.Println(method, " request to ", url)
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to create request: %w", err)
	}

	//check if need more headers
	req.Header.Set("X-API-Key", ApiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to perform request: %w", err)
	}

	respBytes, err := GetBody(resp)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to get response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, -1, fmt.Errorf("HTTP %d: %s \n", resp.StatusCode, respBytes)
	}

	return respBytes, resp.StatusCode, nil
}

func GetBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return body, nil
}
