package main

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	deserializer  = serializer.NewCodecFactory(runtimeScheme).UniversalDeserializer()
	defaulter     = runtime.ObjectDefaulter(runtimeScheme)
)

var (
	exporterAnnotationsKey = "inject-exporters"
	exporterUpdatedAnnotationsKey = "inject-exporter-updated"
)


func loadConfig (fileSuffix, sidecarCfgDirectoryPath string) (Config, error){

	log.Infoln("Checkin sidecar configuration file located here: ", sidecarCfgDirectoryPath)
	data, err := ioutil.ReadFile(sidecarCfgDirectoryPath + "/config_" + fileSuffix + ".json")
	if err != nil {
		log.Error("Cannot read sidecar configuration file or file not found! ", err)
		return Config{}, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.Error("Cannot unmarshal sidecar configuration JSON file..", err)
		return Config{}, err
	}

	return config, nil
}

func (whs *WebhookServer) checkMutate(annotations map[string]string) ([]Config, bool) {
	if value, ok := annotations[exporterAnnotationsKey]; ok {
		exporterLists := strings.Split(value, ",")
		var exporterConfigurationList []Config

		for _, value := range exporterLists {
			configLoaded, err := loadConfig(value, whs.Parameters.SidecarConfigurationDirectory)
			if err != nil {
				return nil, false
			}
			exporterConfigurationList = append(exporterConfigurationList, configLoaded)
		}
		return exporterConfigurationList, true
	}
	return nil, false
}


func (whs *WebhookServer) mutate(review *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	var pod corev1.Pod
	if err := json.Unmarshal(review.Request.Object.Raw, &pod); err != nil {
		return &v1beta1.AdmissionResponse{
			Allowed: false,
			Result:  &metav1.Status{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Message:  err.Error(),
			},
		}
	}


	config, err := whs.checkMutate(pod.GetAnnotations())
	if !err {
		log.Infof("No need to mutate Pod %v", pod.Name)
		return &v1beta1.AdmissionResponse{
			Allowed:  true,
		}
	}

	whs.Parameters.SidecarConfiguration = config
	//Now is time to PATCH


	return &v1beta1.AdmissionResponse{
		Allowed:  true,
	}
}

func (whs *WebhookServer) healthHandler (writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application-json")
	writer.WriteHeader(http.StatusOK)

	jsonReturnData := make(map[string]string)
	jsonReturnData["status"] = "ALIVE"
	if err := json.NewEncoder(writer).Encode(jsonReturnData); err != nil {
		http.Error(writer, "Error on health check", http.StatusInternalServerError)
	}
}


func (whs *WebhookServer) mutateHandler(writer http.ResponseWriter, request *http.Request) {

	if request.Body == nil || request.Header.Get("Content-type") != "application/json" {
		log.Error("Error perfoming the request....")
		http.Error(writer, "Error perfoming the request, body empty or wrong header content-type...", http.StatusInternalServerError)
		return
	}

	data, err := ioutil.ReadAll(request.Body)
	if err != nil || len(data) == 0 {
		log.Error("Error reading the body or body is empty!!")
		http.Error(writer, "Error reading the body or body is empty!!", http.StatusInternalServerError)
	}
	log.Infoln("Successfully get the request....")

	admissionReview := v1beta1.AdmissionReview{}
	admissionReviewResponse := v1beta1.AdmissionReview{}
	_, _, err = deserializer.Decode(data, nil, &admissionReview)
	if err != nil {
		log.Error("Cannot decode object admission review....", err)
		http.Error(writer, "cannot decode object...", http.StatusInternalServerError)
	} else {
		kind := strings.ToLower(admissionReview.Kind)
		if strings.Contains(kind, "pod") {
			admissionReviewResponse.Response = whs.mutate(&admissionReview)
			admissionReviewResponse.Response.UID = admissionReview.Request.UID
		} else {
			admissionReviewResponse.Response = &v1beta1.AdmissionResponse{
				UID:              admissionReview.Request.UID,
				Allowed:          true,
			}
		}
	}

	if resp, err := json.Marshal(admissionReviewResponse); err != nil {
		_, err = writer.Write(resp) //here we terminate the process of /mutate
		if err != nil {
			log.Error("Can't write response...", err)
			http.Error(writer, "Can't write the response", http.StatusInternalServerError)
		}
	}

}
