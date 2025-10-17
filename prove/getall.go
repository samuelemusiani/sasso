package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
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

type RRSets struct {
	Name     string    `json:"name"`
	Records  []Records `json:"records"`
	Type     string    `json:"type"`
	TTL      int       `json:"ttl"`
	Comments []string  `json:"comment"`
}

type RecordsResponse struct {
	Name   string   `json:"name"`
	RRSets []RRSets `json:"rrsets"`
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

func GetAll() {
	body, err := GetZones()
	if err != nil {
		log.Fatalf("failed to get zones: %v", err)
	}
	fmt.Printf("Zones: %s\n", string(body))

	var zones []Zone
	if err := json.Unmarshal(body, &zones); err != nil {
		log.Fatalf("failed to parse JSON: %v", err)
	}

	for _, zone := range zones {
		fmt.Printf("\nID: %s, Name: %s, KInd: %s\n", zone.ID, zone.Name, zone.Kind)
		recordsBody, err := GetRecords(zone.ID)
		if err != nil {
			log.Fatalf("failed to get records for zone %s: %v", zone.Name, err)
		}
		// fmt.Printf("Records for zone %s: %s\n", zone.Name, string(recordsBody))

		var recordsResp RecordsResponse
		if err := json.Unmarshal(recordsBody, &recordsResp); err != nil {
			log.Fatalf("failed to parse JSON for records in zone %s: %v", zone.Name, err)
		}
		for _, rrset := range recordsResp.RRSets {
			fmt.Printf("\n  Name: %s, Type: %s, TTL: %d\n", rrset.Name, rrset.Type, rrset.TTL)
			for _, record := range rrset.Records {
				fmt.Printf("    Record Content: %s, Disabled: %t\n", record.Content, record.Disabled)
			}
		}
	}

	body, err = GetNetworks()
	if err != nil {
		log.Fatalf("failed to get views: %v", err)
	}
	fmt.Printf("\nNetworks: %s\n", string(body))

	var networksResp NetworksResponse
	if err := json.Unmarshal(body, &networksResp); err != nil {
		log.Fatalf("failed to parse JSON: %v", err)
	}

	fmt.Println("\nNetworks:")
	for _, network := range networksResp.Networks {
		fmt.Printf("Network: %s, View: %s\n", network.Network, network.View)
	}

	body, err = GetViews()
	if err != nil {
		log.Fatalf("failed to get views: %v", err)
	}
	fmt.Printf("\nViews: %s\n", string(body))

	var viewsResp Views
	if err := json.Unmarshal(body, &viewsResp); err != nil {
		log.Fatalf("failed to parse JSON: %v", err)
	}

	fmt.Println("\nViews:")
	for _, view := range viewsResp.Views {
		fmt.Printf("View Name: %s\n", view)
	}
}

func HttpGetRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", ApiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request: %w", err)
	}

	return resp, nil
}

func GetBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return body, nil
}

func GetZones() ([]byte, error) {
	zonesUrl := BaseUrl + "/zones"

	resp, err := HttpGetRequest(zonesUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %w", err)
	}

	body, err := GetBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get response body: %w", err)
	}

	return body, nil
}

func GetRecords(zoneId string) ([]byte, error) { // !!! zoneId and zoneName looks equal but i think they are different... to decide which one to use
	recordsUrl := BaseUrl + "/zones/" + zoneId

	resp, err := HttpGetRequest(recordsUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}

	body, err := GetBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get response body: %w", err)
	}

	return body, nil
}

func GetNetworks() ([]byte, error) {
	networksUrl := BaseUrl + "/networks"

	resp, err := HttpGetRequest(networksUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	body, err := GetBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get response body: %w", err)
	}

	return body, nil
}

func GetViews() ([]byte, error) {
	viewsUrl := BaseUrl + "/views"

	resp, err := HttpGetRequest(viewsUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to list views: %w", err)
	}

	body, err := GetBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get response body: %w", err)
	}

	return body, nil
}
