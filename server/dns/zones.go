package dns

import (
	"encoding/json"
	"fmt"
)

func createZoneWithRRSets(zone Zone) error {
	url := fmt.Sprintf("%s/zones", BaseUrl)

	reqBody := map[string]interface{}{
		"name": zone.Name,
		"kind": "Native",
	}

	_, _, err := HttpRequest("POST", url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create zone: %w", err)
	}
	for _, rrset := range zone.RRSets {
		err := newRRSetInZone(rrset, zone)
		if err != nil {
			logger.With("error", err).Error("Failed to create RRset in zone")
			return fmt.Errorf("failed to create RRset in zone %s: %w", zone.Name, err)
		}
	}

	return nil
}

func createZonesWithRRSets(zones []Zone) error {
	for _, zone := range zones {
		err := createZoneWithRRSets(zone)
		if err != nil {
			return fmt.Errorf("failed to create zones: %w", err)
		}
	}
	return nil
}

func deleteZonesFromDNS(zones []Zone) error {
	for _, zone := range zones {
		err := deleteZoneFromDNS(zone)
		if err != nil {
			return fmt.Errorf("failed to delete zone from DNS: %w", err)
		}
	}
	return nil
}

// Deletes this zone, all attached metadata and rrsets.
func deleteZoneFromDNS(zone Zone) error {
	url := fmt.Sprintf("%s/zones/%s", BaseUrl, zone.Name)

	respBody, statusCode, err := HttpRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to delete view: %w", err)
	}

	fmt.Printf("%d Response: %s", statusCode, string(respBody))
	return nil
}

// Creates RRset present in the payload and their comments
func newRRSetInZone(RRset RRSet, zone Zone) error {
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

	_, _, err := HttpRequest("PATCH", url, reqBody)
	if err != nil {

		return fmt.Errorf("failed to create RRset: %w", err)
	}

	return nil
}

// Add array of RRsets to the specified zone
func addRRSetsToZone(RRsets []RRSet, zone Zone) error {
	for _, rrset := range RRsets {
		err := newRRSetInZone(rrset, zone)
		if err != nil {
			return fmt.Errorf("failed to add RRset to zone %s: %w", zone.Name, err)
		}
	}
	return nil
}

// Delete RRsets present in the payload and their comments
func deleteRRSetFromZone(RRset RRSet, zone Zone) error {
	url := fmt.Sprintf("%s/zones/%s", BaseUrl, zone.Name)

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

	_, _, err := HttpRequest("PATCH", url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to delete RRset: %w", err, "rrset", RRset.Name)
	}

	return nil
}

// ---NOTICE: this way we are deleting rrsets based on the struct we pass; we may fail to delete if not a rrset is in struct
// and not on database, or we may leave an rrset behind if it is in dns and not in struct
func deleteAllRRSetsFromZone(zone Zone) error {

	for _, rrset := range zone.RRSets {
		err := deleteRRSetFromZone(rrset, zone)
		if err != nil {
			return fmt.Errorf("Failed to delete all RRsets: %w", err)
		}
	}

	return nil
}

//
// func GetAllZones() ([]byte, error) {
// 	url := fmt.Sprintf("%s/zones", BaseUrl)
//
// 	respBody, _, err := HttpRequest("GET", url, nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get zones: %w", err)
// 	}
//
// 	fmt.Printf("Zones Response: %s", string(respBody))
//
// 	return respBody, nil
//}

func GetStructZoneWithRecordsByName(zoneName string) (Zone, error) {
	url := fmt.Sprintf("%s/zones/%s", BaseUrl, zoneName)

	respBody, _, err := HttpRequest("GET", url, nil)
	if err != nil {
		return Zone{}, fmt.Errorf("failed to get zone %s: %w", zoneName, err)
	}

	var RRSets []RRSet
	var tmp struct {
		RRSets []RRSet `json:"rrsets"`
	}

	if err := json.Unmarshal(respBody, &tmp); err != nil {
		return Zone{}, fmt.Errorf("failed to parse zone JSON for zone %s: %w", zoneName, err)
	}

	RRSets = tmp.RRSets

	return Zone{
		Name:   zoneName,
		RRSets: RRSets,
	}, nil
}

// func GetZoneRecords(zone *Zone) ([]Record, error) {
// 	url := fmt.Sprintf("%s/zones/%s", BaseUrl, zone.ID)
// 	respBody, _, err := HttpRequest("GET", url, nil)
// 	if err != nil {
// 		return Records{}, fmt.Errorf("failed to get records for zone %s: %w", zone.ID, err)
// 	}
//
// 	var recordsResp RecordsResponse
// 	if err := json.Unmarshal(respBody, &recordsResp); err != nil {
// 		return Records{}, fmt.Errorf("failed to parse records JSON for zone %s: %w", zone.ID, err)
// 	}
//
// 	return Records{Records: recordsResp.RRSets[0].Records}, nil
// }
