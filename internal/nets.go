package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"samuelemusiani/sasso/internal/auth"
)

func FetchNets(parentCtx context.Context, endpoint, secret string) (nets []Net, err error) {
	client := http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(parentCtx, http.MethodGet, endpoint+"/internal/net", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to fetch nets: %w", err)
	}

	req = auth.AddAuthToRequest(req, secret)

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request to fetch nets: %w", err)
	}

	defer func() {
		// Checking for nil err to avoid overwriting previous errors
		if closeErr := res.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error while closing request body: %w", closeErr)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch nets: non-200 status code. %s", res.Status)
	}

	err = json.NewDecoder(res.Body).Decode(&nets)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nets response: %w", err)
	}

	return nets, nil
}

func UpdateNet(parentCtx context.Context, endpoint, secret string, net Net) (err error) {
	body, err := json.Marshal(net)
	if err != nil {
		return fmt.Errorf("failed to marshal net data: %w", err)
	}

	client := http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(parentCtx, http.MethodPut, endpoint+"/internal/net/"+strconv.FormatUint(uint64(net.ID), 10), bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request to update net: %w", err)
	}

	req = auth.AddAuthToRequest(req, secret)

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform request to update net: %w", err)
	}

	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error while closing request body: %w", closeErr)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update net: non-200 status code. %s", res.Status)
	}

	return nil
}
