APP = exporters-webhook
NAMESPACE = utils

.PHONY:cert
cert:
	./generate_ssl.sh $(APP) $(NAMESPACE)

.PHONY:clean
clean:
	rm -vf *.key *.pem *.cert *.crt *.csr
