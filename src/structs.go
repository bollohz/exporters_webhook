package main

import (
	"net/http"
)

type WebhookServer struct {
	Server *http.Server
	Parameters *WebhookParameters
}


type WebhookParameters struct {
	Port string
	SidecarConfiguration *Config
	Timeout int
}

type Config struct {
	ContainerImage string `json:"containerImage"`
	ContainerCommand string `json:"containerCommand"`
}
