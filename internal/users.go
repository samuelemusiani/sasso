package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"samuelemusiani/sasso/internal/auth"
)

type User struct {
	ID                 uint `json:"id"`
	NumberOFVPNConfigs uint `json:"number_of_vpn_configs"`
}

func FetchUsers(endpoint, secret string) (users []User, err error) {
	client := http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(http.MethodGet, endpoint+"/internal/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to fetch nets status: %w", err)
	}

	req = auth.AddAuthToRequest(req, secret)

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request to fetch nets status: %w", err)
	}

	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error while closing request body: %w", closeErr)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch nets status: non-200 status code. %s", res.Status)
	}

	err = json.NewDecoder(res.Body).Decode(&users)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nets status: %w", err)
	}

	return users, nil
}
