package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"samuelemusiani/sasso/server/db"
)

type Zone struct {
	Name   string  `json:"name"`
	RRSets []RRSet `json:"rrsets"`
}

type Record struct {
	Ip       string `json:"content"`
	Disabled bool   `json:"disabled"`
}

type RRSet struct {
	Name    string   `json:"name"`
	Records []Record `json:"records"`
	Type    string   `json:"type"`
	TTL     int      `json:"ttl"`
}

type RecordsResponse struct {
	Name   string  `json:"name"`
	RRSets []RRSet `json:"rrsets"`
}

type network struct {
	Network string `json:"network"`
	View    string `json:"view"`
}

type View struct {
	Name     string   `json:"name"`
	Networks []string `json:"network"`
	Zones    []Zone   `json:"zones"`
}

type Views struct {
	Views []View `json:"views"`
}

var (
	BaseUrl string
	ApiKey  string
)

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

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to create request: %w", err)
	}
	req.Close = true

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

func CompareRecords(RRSet2 RRSet, RRSet1 RRSet) bool {
	if len(RRSet1.Records) != len(RRSet2.Records) {
		return false
	}
	recordMap := make(map[string]bool)
	for _, record := range RRSet1.Records {
		recordMap[record.Ip] = true
	}
	for _, record := range RRSet2.Records {
		if _, exists := recordMap[record.Ip]; !exists {
			return false
		}
	}
	return true
}

// ValidateUniqueNetNames checks that no two nets have the same ID
// Returns an error listing all duplicate IDs found
func ValidateUniqueNetNames(nets []db.Net) error {
	seen := make(map[uint]bool)
	duplicates := []uint{}

	for _, net := range nets {
		if seen[net.ID] {
			duplicates = append(duplicates, net.ID)
		}
		seen[net.ID] = true
	}

	if len(duplicates) > 0 {
		return fmt.Errorf("duplicate net IDs found: %v", duplicates)
	}
	return nil
}

// ValidateUniqueZoneNames checks that no two zones have the same name
// Returns an error listing all duplicate names found
func ValidateUniqueZoneNames(zones []Zone) error {
	seen := make(map[string]bool)
	duplicates := []string{}

	for _, zone := range zones {
		if seen[zone.Name] {
			duplicates = append(duplicates, zone.Name)
		}
		seen[zone.Name] = true
	}

	if len(duplicates) > 0 {
		return fmt.Errorf("duplicate zone names found: %v", duplicates)
	}
	return nil
}

// ValidateUniqueRRSetNames checks that no two RRSets have the same name+type combination
// Returns an error listing all duplicate name+type pairs found
func ValidateUniqueRRSetNames(rrsets []RRSet) error {
	seen := make(map[string]bool)
	duplicates := []string{}

	for _, rrset := range rrsets {
		// RRSets are unique by name+type combination
		key := fmt.Sprintf("%s:%s", rrset.Name)
		if seen[key] {
			duplicates = append(duplicates, key)
		}
		seen[key] = true
	}

	if len(duplicates) > 0 {
		return fmt.Errorf("duplicate RRSet name combinations found: %v", duplicates)
	}
	return nil
}
