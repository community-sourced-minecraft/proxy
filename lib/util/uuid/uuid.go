package uuid

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type MCResponse struct {
	Name  *string `json:"name"`
	ID    *string `json:"id"`
	Error *string `json:"errorMessage"`
	Path  *string `json:"path"`
}

func UsernameToUUID(username string) (string, error) {
	res, err := http.Get("https://api.mojang.com/users/profiles/minecraft/" + username)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var uuid MCResponse
	if err := json.NewDecoder(res.Body).Decode(&uuid); err != nil {
		return "", err
	}

	if uuid.Error != nil {
		return "", fmt.Errorf("mojang api: %s", *uuid.Error)
	}

	return *uuid.ID, nil
}

func UUIDtoUsername(UUID string) (string, error) {
	res, err := http.Get("https://sessionserver.mojang.com/session/minecraft/profile/" + UUID)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var username MCResponse
	if err := json.NewDecoder(res.Body).Decode(&username); err != nil {
		return "", err
	}

	if username.Error != nil {
		return "", fmt.Errorf("mojang api: %s", *username.Error)
	}

	return *username.Name, nil
}
