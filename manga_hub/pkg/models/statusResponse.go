package models

type StatusResponse struct {
	Status string `json:"status"`
	User		  `json:"user"`
}