package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"samuelemusiani/sasso/internal/auth"
	"time"
)

func FetchPortForwards(endpoint, secret string) ([]PortForward, error) {
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", endpoint+"/internal/port-forwards", nil)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to create request to fetch port forwards"))
	}
	auth.AddAuthToRequest(req, secret)

	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to perform request to fetch port forwards"))
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.Join(err, fmt.Errorf("failed to fetch port forwards: non-200 status code. %s", res.Status))
	}

	var portForwards []PortForward
	err = json.NewDecoder(res.Body).Decode(&portForwards)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to decode port forwards status"))
	}

	return portForwards, nil
}
