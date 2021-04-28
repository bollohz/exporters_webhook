#! /bin/sh
set -u

export NAMESPACE="${1}"
export CSR_NAME="${2}"
export CA_BUNDLE=$(kubectl get csr "${CSR_NAME}"  -o jsonpath='{.status.certificate}')

helm3 upgrade --install exporters-webhook ./helm/exporters-webhook --set caBundle="${CA_BUNDLE}" --namespace "${NAMESPACE}" -f ./helm/exporters-webhook/values.yaml --debug --version 1.0.0
