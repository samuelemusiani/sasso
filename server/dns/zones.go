package dns

import (
	// "encoding/json"
	"fmt"
)

func CreateZones(zones []Zone) error {
	for _, zone := range zones {
		url := fmt.Sprintf("%s/zones", BaseUrl)

		reqBody := map[string]interface{}{
			"id":   zone.Name,
			"kind": "Native",
		}

		_, _, err := HttpRequest("POST", url, reqBody)
		if err != nil {
			return fmt.Errorf("failed to create zone: %w", err)
		}
	}
	return nil
}

// // Deletes this zone, all attached metadata and rrsets.
// func DeleteZone(zone Zone) error {
// 	url := fmt.Sprintf("%s/zones/%s", BaseUrl, zone.ID)
//
// 	respBody, statusCode, err := HttpRequest("DELETE", url, nil)
// 	if err != nil {
// 		return fmt.Errorf("failed to delete view: %w", err)
// 	}
//
// 	fmt.Printf("%d Response: %s", statusCode, string(respBody))
// 	return nil
// }

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

// // Delete RRsets present in the payload and their comments
// func DeleteRRsetFromZone(RRset RRSet, zone Zone) error {
// 	url := fmt.Sprintf("%s/zones/%s", BaseUrl, zone.ID)
//
// 	rrsets := []map[string]interface{}{
// 		{
// 			"name":       RRset.Name,
// 			"type":       RRset.Type,
// 			"changetype": "DELETE",
// 		},
// 	}
//
// 	reqBody := map[string]interface{}{
// 		"rrsets": rrsets,
// 	}
//
// 	respBody, statusCode, err := HttpRequest("PATCH", url, reqBody)
// 	if err != nil {
// 		return fmt.Errorf("failed to create RRset: %w", err)
// 	}
//
// 	fmt.Printf("%d Response: %s", statusCode, string(respBody))
// 	return nil
// }
//
// func GetZones() ([]byte, error) {
// 	url := fmt.Sprintf("%s/zones", BaseUrl)
//
// 	respBody, _, err := HttpRequest("GET", url, nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get zones: %w", err)
// 	}
//
// 	return respBody, nil
// }
//
// func GetZoneRecords(zone *Zone) (Records, error) {
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
