package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"samuelemusiani/sasso/internal/auth"
)

func FetchPortForwards(endpoint, secret string) (portForwards []PortForward, err error) {
	client := http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(http.MethodGet, endpoint+"/internal/port-forwards", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to fetch port forwards: %w", err)
	}

	req = auth.AddAuthToRequest(req, secret)

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request to fetch port forwards: %w", err)
	}

	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error while closing request body: %w", closeErr)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch port forwards: non-200 status code. %s", res.Status)
	}

	err = json.NewDecoder(res.Body).Decode(&portForwards)
	if err != nil {
		return nil, fmt.Errorf("failed to decode port forwards response: %w", err)
	}

	return portForwards, nil
}
