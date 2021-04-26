package main

import (
	"net/http"
)

type WebhookServer struct {
	Server *http.Server
	Parameters *WebhookParameters
}


type WebhookParameters struct {
	Port                          string
	SidecarConfigurationDirectory string
	Timeout                       int
	SidecarConfiguration          []Config
}

type Config struct {
	ContainerImage string `json:"containerImage"`
	ContainerCommand string `json:"containerCommand"`
}
