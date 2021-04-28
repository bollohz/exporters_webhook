#! /bin/sh
set -u

export CA_BUNDLE=$(kubectl get csr exporters-webhook.utils.svc  -o jsonpath='{.status.certificate}')
helm3 template --set caBundle="${CA_BUNDLE}" -f exporters-webhook/values.yaml ./exporters-webhook > exporters-webhook/out.yaml --debug
