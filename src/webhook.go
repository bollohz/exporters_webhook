package main

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
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

func loadConfig (fileSuffix, sidecarCfgDirectoryPath string) (corev1.Container, error){

	log.Infoln("Checkin sidecar configuration file located here: ", sidecarCfgDirectoryPath)
	data, err := ioutil.ReadFile(sidecarCfgDirectoryPath + "/config_" + fileSuffix + ".yaml")
	if err != nil {
		log.Error("Cannot read sidecar configuration file or file not found! ", err)
		return corev1.Container{}, err
	}

	var config corev1.Container
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Error("Cannot unmarshal sidecar configuration JSON file..", err)
		return corev1.Container{}, err
	}

	return config, nil
}

func updateAnnotations() MutatingPatch{
	return MutatingPatch{
		Op:    "add",
		Path:  "/metadata/annotations",
		Value: map[string]string{
			"injected-by-sidecar": "true",
		},
	}
}

func addSidecarContainerExporter(target []corev1.Container, sidecarConfiguration []corev1.Container) []MutatingPatch{
	var patches []MutatingPatch

	first := len(target) == 0
	var value interface{}

	for _, add := range sidecarConfiguration {
		value = add
		path := "/spec/containers"
		if first {
			first = false
			value = []corev1.Container{add}
		} else {
			path = path + "/-"
		}

		patches = append(patches, MutatingPatch{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patches
}

func (whs *WebhookServer) checkMutateAndGetConfig(labels map[string]string) ([]corev1.Container, bool) {
	log.Infof("Checking configuration for handling mutate correctly....")
	if value, ok := labels[exporterAnnotationsKey]; ok {
		exporterLists := strings.Split(value, ",")
		var exporterConfigurationList []corev1.Container
		for _, value := range exporterLists {
			configLoaded, err := loadConfig(value, whs.Parameters.SidecarConfigurationDirectory)
			if err != nil {
				log.Errorf("Error during load of config %v: ", value, err)
				return nil, false
			}
			exporterConfigurationList = append(exporterConfigurationList, configLoaded)
		}
		log.Infof("Exporter configuration list are.... %v", exporterConfigurationList)
		return exporterConfigurationList, true
	}
	return nil, false
}

func (whs *WebhookServer) createPatch(pod *corev1.Pod) ([]byte, v1beta1.PatchType, error) {
	var patches []MutatingPatch
	patchType := v1beta1.PatchTypeJSONPatch

	log.Infof("Add Sidecar configuration...")
	patches = append(patches, addSidecarContainerExporter(pod.Spec.Containers, whs.Parameters.SidecarConfiguration)...)
	log.Infof("Add annotations...")
	patches = append(patches, updateAnnotations())
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		log.Errorf("Error during marshal of patch response")
		return nil, patchType, err
	}

	return patchBytes, patchType, nil
}


func (whs *WebhookServer) mutate(review *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {

	log.Infof("Mutating the admissionReview.....")
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

	config, err := whs.checkMutateAndGetConfig(pod.GetLabels())
	if !err {
		log.Infof("No need to mutate Pod %v", pod.Name)
		return &v1beta1.AdmissionResponse{
			Allowed:  true,
		}
	}

	//Now is time to PATCH
	whs.Parameters.SidecarConfiguration = config
	patchBytes, JSONPatchType, errorPatch := whs.createPatch(&pod)
	log.Infof("Sending patch request... %v", patchBytes)
	if errorPatch != nil {
		return &v1beta1.AdmissionResponse{
			Allowed: false,
			Result:  &metav1.Status{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Message:  errorPatch.Error(),
			},
		}
	}

	return &v1beta1.AdmissionResponse{
		UID: review.Request.UID,
		Allowed:  true,
		Patch: patchBytes,
		PatchType: &JSONPatchType,
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
	log.Infof("Request body is %v", data)
	log.Infoln("Successfully get the request....")

	admissionReview := v1beta1.AdmissionReview{}
	admissionReviewResponse := v1beta1.AdmissionReview{}
	_, _, err = deserializer.Decode(data, nil, &admissionReview)
	if err != nil {
		log.Error("Cannot decode object admission review....", err)
		http.Error(writer, "cannot decode object...", http.StatusInternalServerError)
	} else {
		kind := strings.ToLower(admissionReview.Kind)
		log.Infof("Deserialize object....")
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
