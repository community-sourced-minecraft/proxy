package rpc

import "go.minekube.com/gate/pkg/util/uuid"

type Type string

const (
	TypeTransferPlayer Type = "TRANSFER_PLAYER"
)

type Request struct {
	Type Type   `json:"type"`
	Data string `json:"data"`
}

type Response struct {
	Type Type   `json:"type"`
	Data string `json:"data"`
}

type TransferPlayerRequest struct {
	UUID        uuid.UUID `json:"uuid"`
	Source      string    `json:"source"`
	Destination string    `json:"destination"`
}

type TransferPlayerResponse struct {
	Status Status `json:"status"`
}

type Status string

const (
	StatusOk    Status = "OK"
	StatusError Status = "ERROR"
)
