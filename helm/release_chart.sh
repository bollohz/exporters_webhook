#! /bin/sh
set -u

export NAMESPACE="${1}"
export CA_BUNDLE=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}')

helm3 upgrade --install exporters-webhook ./helm/exporters-webhook --set caBundle="${CA_BUNDLE}" --namespace "${NAMESPACE}" -f ./helm/exporters-webhook/values.yaml --debug --version 1.0.0
