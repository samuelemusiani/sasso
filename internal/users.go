package internal

import (
	"encoding/json"
	"errors"
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

	req, err := http.NewRequest("GET", endpoint+"/internal/user", nil)
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

	err = json.NewDecoder(res.Body).Decode(&users)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to decode nets status"))
	}

	return
}
