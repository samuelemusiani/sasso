package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"samuelemusiani/sasso/internal/auth"
)

func FetchVPNConfigs(endpoint, secret string) (vpns []VPNProfile, err error) {
	client := http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(http.MethodGet, endpoint+"/internal/vpn", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to fetch vpn status: %w", err)
	}

	req = auth.AddAuthToRequest(req, secret)

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request to fetch vpn status: %w", err)
	}

	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error while closing request body: %w", closeErr)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch vpn status: non-200 status code. %s", res.Status)
	}

	err = json.NewDecoder(res.Body).Decode(&vpns)
	if err != nil {
		return nil, fmt.Errorf("failed to decode vpn status: %w", err)
	}

	return vpns, nil
}

func UpdateVPNConfig(endpoint, secret string, vpn VPNProfile) (err error) {
	body, err := json.Marshal(vpn)
	if err != nil {
		return fmt.Errorf("failed to marshal vpn data: %w", err)
	}

	client := http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(http.MethodPut, endpoint+"/internal/vpn", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request to update vpn config: %w", err)
	}

	req = auth.AddAuthToRequest(req, secret)

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform request to update vpn config: %w", err)
	}

	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error while closing request body: %w", closeErr)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update vpn config: non-200 status code. %s", res.Status)
	}

	return nil
}
