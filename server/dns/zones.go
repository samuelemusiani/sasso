package dns

import (
	"encoding/json"
	"fmt"
)

// Create a new zone from arg passed
func newZoneWithRRSets(zone Zone) error {
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

// Create a new zones from array of zones
func newZonesWithRRSets(zones []Zone) error {
	for _, zone := range zones {
		err := newZoneWithRRSets(zone)
		if err != nil {
			return fmt.Errorf("failed to create zones: %w", err)
		}
	}
	return nil
}

// Deletes array of zones
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
		return fmt.Errorf("failed to delete zone: %w", err)
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

		return fmt.Errorf("failed to create RRset", "err", err)
	}

	return nil
}

// Add array of RRsets to the specified zone
func addRRSetsToZone(RRsets []RRSet, zone Zone) error {
	for _, rrset := range RRsets {
		err := newRRSetInZone(rrset, zone)
		if err != nil {
			return fmt.Errorf("failed to add RRset to zone", "zone", zone.Name, "err", err)
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
		return fmt.Errorf("failed to delete RRset", "err", err, "rrset", RRset.Name)
	}

	return nil
}

// ---NOTICE: this way we are deleting rrsets based on the struct we pass; we may fail to delete if a rrset is not in struct
// and not on database, or we may leave an rrset behind if it is in dns and not in struct
func deleteAllRRSetsFromZone(zone Zone) error {

	for _, rrset := range zone.RRSets {
		err := deleteRRSetFromZone(rrset, zone)
		if err != nil {
			return fmt.Errorf("Failed to delete all RRsets", "err", err)
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
		return Zone{}, fmt.Errorf("Failed to get zone", "zone", zoneName, "err", err)
	}

	var RRSets []RRSet
	var tmp struct {
		RRSets []RRSet `json:"rrsets"`
	}

	if err := json.Unmarshal(respBody, &tmp); err != nil {
		return Zone{}, fmt.Errorf("Failed to parse zone JSON", "zone", zoneName, "err", err)
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
