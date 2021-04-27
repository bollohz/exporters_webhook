package main

import (
	corev1 "k8s.io/api/core/v1"
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
	SidecarConfiguration          []corev1.Container
}

type MutatingPatch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}
