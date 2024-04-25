package uuid

import (
	"encoding/json"
	"net/http"
)

func UsernameToUUID(username string) (string, error) {
	res, err := http.Get("https://api.mojang.com/users/profiles/minecraft/" + username)
	if err != nil {
		return "", err
	}

	var uuid MCResponse

	if err := json.NewDecoder(res.Body).Decode(&uuid); err != nil {
		return "", err
	}

	return *uuid.ID, nil
}

func UUIDtoUsername(UUID string) (string, error) {
	res, err := http.Get("https://sessionserver.mojang.com/session/minecraft/profile/" + UUID)
	if err != nil {
		return "", err
	}

	var username MCResponse

	if err := json.NewDecoder(res.Body).Decode(&username); err != nil {
		return "", err
	}

	return *username.Name, nil
}

type MCResponse struct {
	Name  *string `json:"name"`
	ID    *string `json:"id"`
	Error *string `json:"errorMessage"`
	Path  *string `json:"path"`
}
