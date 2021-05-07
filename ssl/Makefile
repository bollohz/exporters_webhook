APP       = exporters-webhook
NAMESPACE = utils
SECRET    = exporters-webhook-secrets

.PHONY:cert
cert:
	./generate_ssl.sh --service $(APP) --secret $(SECRET) --namespace $(NAMESPACE)

.PHONY:clean
clean:
	rm -vf *.key *.pem *.cert *.crt *.csr
