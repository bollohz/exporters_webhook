package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)


func main() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	whParams := &WebhookParameters{
		Port:                 "",
		SidecarConfiguration: []corev1.Container{},
		Timeout:              0,
	}
	flag.StringVar(&whParams.SidecarConfigurationDirectory, "sidecarCfgDirectory", "/etc/exporters_configuration", "Path for SidecarContainer Configuration file")
	flag.StringVar(&whParams.Port, "port", "8080", "Configuration Port for the WebHook Server")
	flag.StringVar(&whParams.CertFile, "tlsCertFile", "/etc/webhook/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&whParams.KeyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem", "File containing the x509 private key to -tlsCertFile.")
	flag.IntVar(&whParams.Timeout, "timeout", 300, "Timeout for graceful Shutdown of the server")
	flag.Parse()


	pair, err := tls.LoadX509KeyPair(whParams.CertFile, whParams.KeyFile)
	if err != nil {
		log.Errorf("Error during load of X509 KeyPair %v", err)
	}
	log.Infoln("Starting WebHook server....")
	whAddress := fmt.Sprintf("0.0.0.0:%v", whParams.Port)
	log.Infoln("Address is: ", whAddress)

	webhookServer := &WebhookServer{
		Parameters: whParams,
		Server: &http.Server{
			Addr:              whAddress,
			TLSConfig:         &tls.Config{
				InsecureSkipVerify: true,
				Certificates:  []tls.Certificate{pair},
			},
			ReadTimeout:       15 * time.Second,
			ReadHeaderTimeout: 15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       30 * time.Second,
		//	Handler is defined below
		},
	}

	//generate router and add it to webhookServer
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/health", webhookServer.healthHandler).
		Methods(http.MethodGet).
		Schemes("http", "https")
	muxRouter.HandleFunc("/mutate", webhookServer.mutateHandler)
	//Setting handler in server
	webhookServer.Server.Handler = muxRouter

	go func() {
		log.Infoln("Server started...")
		err := webhookServer.Server.ListenAndServeTLS(whParams.CertFile, whParams.KeyFile)
		if err != nil {
			log.Error("ERROR STARTING THE WEBHOOK SERVER! %v", err)
			panic(err)
		}
	}()

	// Listen to interrupt signal
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, syscall.SIGINT, syscall.SIGTERM)
	// block execution until the signal is not arrived...
	<-channel


	timeoutContext, cancel := context.WithTimeout(context.Background(), 100 * time.Duration(whParams.Timeout))
	defer cancel()
	log.Infoln("Shutting down the server....")
	_ = webhookServer.Server.Shutdown(timeoutContext)

	os.Exit(0)
}
