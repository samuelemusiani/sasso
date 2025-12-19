package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"samuelemusiani/sasso/internal/auth"
)

func FetchNets(endpoint, secret string) (nets []Net, err error) {
	client := http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(http.MethodGet, endpoint+"/internal/net", nil)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to create request to fetch nets status"))
	}

	auth.AddAuthToRequest(req, secret)

	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to perform request to fetch nets status"))
	}

	defer func() {
		if e := res.Body.Close(); e != nil {
			err = fmt.Errorf("error while closing request body: %w", e)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return nil, errors.Join(err, fmt.Errorf("failed to fetch nets status: non-200 status code. %s", res.Status))
	}

	err = json.NewDecoder(res.Body).Decode(&nets)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to decode nets status"))
	}

	return
}

func UpdateNet(endpoint, secret string, net Net) (err error) {
	body, err := json.Marshal(net)
	if err != nil {
		return errors.Join(err, errors.New("failed to marshal net update"))
	}

	client := http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(http.MethodPut, endpoint+"/internal/net/"+fmt.Sprintf("%d", net.ID), bytes.NewBuffer(body))
	if err != nil {
		return errors.Join(err, errors.New("failed to create request to fetch net status"))
	}

	auth.AddAuthToRequest(req, secret)

	res, err := client.Do(req)
	if err != nil {
		return errors.Join(err, errors.New("failed to perform request to fetch net status"))
	}

	defer func() {
		if e := res.Body.Close(); e != nil {
			err = fmt.Errorf("error while closing request body: %w", e)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return errors.Join(err, fmt.Errorf("failed to update net: non-200 status code. %s", res.Status))
	}

	return
}
