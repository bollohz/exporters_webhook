# Kubernetes Mutating Webhook for Sidecar Injection of Prometheus Exporter Containers

This tutoral shows how to build and deploy a [MutatingAdmissionWebhook](https://kubernetes.io/docs/admin/admission-controllers/#mutatingadmissionwebhook-beta-in-19) that injects a nginx sidecar container into pod prior to persistence of the object.

## Prerequisites

- [git](https://git-scm.com/downloads)
- [go](https://golang.org/dl/) version v1.16+
- [docker](https://docs.docker.com/install/) version 19.00+
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) version v1.19.1+

Please run this command before start using this WebHook
```
kubectl api-versions | grep admissionregistration.k8s.io
```
The result should be:
```
admissionregistration.k8s.io/v1
admissionregistration.k8s.io/v1beta1
```

## Working locally

1. Install all the go dependecies with `go mod tidy`
2. Start in local the webserver with <br/> 
   `go run ./src -sidecarCfgFilePath conf/exporters_configuration/config.json
   `
```
## Stopping the container is based with SIGTERM or SIGKILL signal.
```
# Release it on the cluster 

1. Move in the target K8s context and namespace
2. Modified the values inside `Makefile` for 
```
IMAGE_REPO = YOUR_REPO_HERE
IMAGE_NAME = YOUR_IMAGE_NAME_HERE
IMAGE_TAG  = YOUR_TAG_HERE
APP		   = exporters-webhook
NAMESPACE  = YOUR_NAMESPACE
CSR_NAME   = exporters-webhook.YOUR_NAMESPACE.svc
```
3. Launch the command `make release-chart`

# Or just release the chart!!

In any case you can directly use my DockerImage and use just release the chart inside the cluster.
For doing so, please run the `./helm/release_chart.sh` script from the root folder of the project.

# How does it works?

1. Add the label `exporter-injection: enabled` in the PodSpec template definiton (it works also for Deployment or Pod object, check the examples.)
2. Add the label `inject-exporters: value1,value2...` for the exporter you want to add inside the pod definition
3. Keep in mind the fact that the exporter Split the label `inject-exporters` for the number of requested exporter and add it dynamically, so you can add 1 or more sidecar at the same time
4. Some exporters are defined inside the `.Values.configurationMap` field, if you want to add or override the exporters use this field.
5. The dataPath inside the CM must contain the `config_` prefix (example: `config_nginx.yaml` and you will add `inject-exporters: nginx` for correctly retrieve the sidecarConfiguration)
6. The sidecar definition are based on the Container Struct from K8s.io. For example:
```
name: nginx-exporter
image: nginx/nginx-prometheus-exporter:0.4.2
ports:
  - containerPort: 9113
resources:
  requests:
    memory: 10Mi
    cpu: 10m
  limits:
    memory: 50Mi
    cpu: 200m
args: ["-nginx.scrape-uri", "http://localhost:81/nginx-status"]
```

# Throubleshooting 

Sometimes you may find that pod is injected with sidecar container as expected, check the following items:

1. The sidecar-injector webhook is in running state and no error logs.
2. The namespace in which application pod is deployed has the correct labels as configured in `mutatingwebhookconfiguration`.
3. Check the `caBundle` is patched to `mutatingwebhookconfiguration` object by checking if `caBundle` fields is empty.
4. Check if the application pod has annotation `exporter-injection: enabled`.

Please refeer at the example inside the helm folder, files:
- test-deployment.yaml
- test-pod.yaml
