package dns

import (
	"samuelemusiani/sasso/server/db"
)

type View struct {
	Name 	 string 'json:"name"'
	Network  string 'json:"network"'
}

type Views struct {
	Views []String 'json:"views"'
}

func GetViewByName(viewName string) (View, error) {
	
}


//potrebbe avere senso ritornare un err 
func DoesExistViewWithUserName(userName string) (bool) {
	views, err := GetViews()
	if err != nil {
		return false
	}

	for _, view := range views {
		if view.Name == userName {
			return true
		}
	}
	return false
}

func GetViews() ([]byte, error) {
	url := fmt.Sprintf("%s/views", BaseUrl)

	respBody, _, err := HttpRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list views: %w", err)
	}

	var views Views
	if err := json.Unmarshal(respBody, &views); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var response []View
	for _, viewString := range views {
		view, err := GetViewByName(viewString)
		if err != nil {
			return nil, fmt.Errorf("failed to get view by name: %w", err)
		}
		response = append(response, view)
	}
	return response, nil
}

