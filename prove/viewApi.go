package main

import (
	"encoding/json"
	"fmt"
)

// Adds a zone to a given view, creating it if needed
func AddZoneToView(view string, zone Zone) error {
	url := fmt.Sprintf("%s/views/%s", BaseUrl, view)

	newViewBody := map[string]interface{}{
		"name": zone.ID,
	}

	respBody, statusCode, err := HttpRequest("POST", url, newViewBody)
	if err != nil {
		return fmt.Errorf("failed to add view: %w", err)
	}

	fmt.Printf("%d Response: %s", statusCode, string(respBody))
	return nil
}

// Removes the given zone from the given view
func RemoveZoneFromView(view string, zone Zone) error {
	url := fmt.Sprintf("%s/views/%s/%s", BaseUrl, view, zone.ID)

	respBody, statusCode, err := HttpRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to remove view: %w", err)
	}

	fmt.Printf("%d Response: %s", statusCode, string(respBody))
	return nil
}

func GetViews() ([]byte, error) {
	url := fmt.Sprintf("%s/views", BaseUrl)

	respBody, _, err := HttpRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list views: %w", err)
	}

	return respBody, nil
}

func PrintViews() error {
	body, err := GetViews()
	if err != nil {
		return fmt.Errorf("failed to get views: %w", err)
	}

	var viewsResp Views
	if err := json.Unmarshal(body, &viewsResp); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	fmt.Println("\nViews:")
	for _, view := range viewsResp.Views {
		fmt.Printf("View Name: %s\n", view)
	}

	return nil
}
