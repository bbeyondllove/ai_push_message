package models

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}
