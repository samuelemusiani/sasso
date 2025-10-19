package main

import (
	"fmt"
)

// Deletes this zone, all attached metadata and rrsets.
func DeleteZone(zone Zone) error {
	url := fmt.Sprintf("%s/zones/%s", BaseUrl, zone.ID)

	respBody, statusCode, err := HttpRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to delete view: %w", err)
	}

	fmt.Println("%d Response: %s", statusCode, string(respBody))
	return nil
}

// Creates RRsets present in the payload and their comments
func NewRRsetInZone(RRset RRSet, zone Zone) error {
	url := fmt.Sprintf("%s/zones/%s", BaseUrl, zone.ID)

	rrsets := map[string]interface{}{
		"name":       RRset.Name,
		"type":       RRset.Type,
		"ttl":        RRset.TTL,
		"changetype": "REPLACE",
		"records":    RRset.Records,
	}

	reqBody := map[string]interface{}{
		"rrsets": rrsets,
	}

	respBody, statusCode, err := HttpRequest("PATCH", url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create RRset: %w", err)
	}

	fmt.Println("%d Response: %s", statusCode, string(respBody))
	return nil
}

// Delete RRsets present in the payload and their comments
func DeleteRRsetFromZone(RRset RRSet, zone Zone) error {
	url := fmt.Sprintf("%s/zones/%s", BaseUrl, zone.ID)

	rrsets := map[string]interface{}{
		"name":       RRset.Name,
		"type":       RRset.Type,
		"changetype": "DELETE",
	}

	reqBody := map[string]interface{}{
		"rrsets": rrsets,
	}

	respBody, statusCode, err := HttpRequest("PATCH", url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create RRset: %w", err)
	}

	fmt.Println("%d Response: %s", statusCode, string(respBody))
	return nil
}
