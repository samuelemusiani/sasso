package main

import (
	"encoding/json"
	"fmt"
)

func CreateZone(zone Zone) error {
	url := fmt.Sprintf("%s/zones", BaseUrl)

	reqBody := map[string]interface{}{
		"name": zone.Name,
		"kind": zone.Kind,
	}

	respBody, statusCode, err := HttpRequest("POST", url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create zone: %w", err)
	}

	fmt.Printf("%d Response: %s", statusCode, string(respBody))
	return nil
}

// Deletes this zone, all attached metadata and rrsets.
func DeleteZone(zone Zone) error {
	url := fmt.Sprintf("%s/zones/%s", BaseUrl, zone.ID)

	respBody, statusCode, err := HttpRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to delete view: %w", err)
	}

	fmt.Printf("%d Response: %s", statusCode, string(respBody))
	return nil
}

// Creates RRsets present in the payload and their comments
func NewRRsetInZone(RRset RRSet, zone Zone) error {
	url := fmt.Sprintf("%s/zones/%s", BaseUrl, zone.Name)

	rrsets := []map[string]interface{}{
		{
			"name":       RRset.Name,
			"type":       RRset.Type,
			"ttl":        RRset.TTL,
			"changetype": "REPLACE",
			"records":    RRset.Records,
		},
	}

	reqBody := map[string]interface{}{
		"rrsets": rrsets,
	}

	respBody, statusCode, err := HttpRequest("PATCH", url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create RRset: %w", err)
	}

	fmt.Printf("%d Response: %s", statusCode, string(respBody))
	return nil
}

// Delete RRsets present in the payload and their comments
func DeleteRRsetFromZone(RRset RRSet, zone Zone) error {
	url := fmt.Sprintf("%s/zones/%s", BaseUrl, zone.ID)

	rrsets := []map[string]interface{}{
		{
			"name":       RRset.Name,
			"type":       RRset.Type,
			"changetype": "DELETE",
		},
	}

	reqBody := map[string]interface{}{
		"rrsets": rrsets,
	}

	respBody, statusCode, err := HttpRequest("PATCH", url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create RRset: %w", err)
	}

	fmt.Printf("%d Response: %s", statusCode, string(respBody))
	return nil
}

func GetZones() ([]byte, error) {
	url := fmt.Sprintf("%s/zones", BaseUrl)

	respBody, _, err := HttpRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get zones: %w", err)
	}

	return respBody, nil
}

func GetRecords(zoneId string) ([]byte, error) { // !!! zoneId and zoneName looks equal but i think they are different... to decide which one to use
	url := fmt.Sprintf("%s/zones/%s", BaseUrl, zoneId)
	respBody, _, err := HttpRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get records for zone %s: %w", zoneId, err)
	}
	return respBody, nil
}

func PrintZones() error {
	body, err := GetZones()
	if err != nil {
		return fmt.Errorf("failed to get zones: %w", err)
	}
	var zones []Zone
	if err := json.Unmarshal(body, &zones); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	for _, zone := range zones {
		fmt.Printf("ID: %s, Name: %s, Kind: %s\n", zone.ID, zone.Name, zone.Kind)
	}
	return nil
}

func PrintRecords(zone Zone) error {
	body, err := GetRecords(zone.Name)
	if err != nil {
		return fmt.Errorf("failed to get records for zone %s: %w", zone.Name, err)
	}

	var recordsResp RecordsResponse
	if err := json.Unmarshal(body, &recordsResp); err != nil {
		return fmt.Errorf("failed to parse JSON for records in zone %s: %w", zone.Name, err)
	}
	fmt.Printf("\nRecords for zone %s:\n", zone.Name)
	for _, rrset := range recordsResp.RRSets {
		fmt.Printf("  Name: %s, Type: %s, TTL: %d\n", rrset.Name, rrset.Type, rrset.TTL)
		for _, record := range rrset.Records {
			fmt.Printf("    Record Content: %s, Disabled: %t\n", record.Content, record.Disabled)
		}
	}
	return nil
}
