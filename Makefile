IMAGE_REPO = docker.io/bollohz
IMAGE_NAME = exporters-webhook
IMAGE_TAG  = 1.0.1
APP		   = exporters-webhook
NAMESPACE  = utils
CSR_NAME   = exporters-webhook.utils.svc

.PHONY:build scan image
build:
	@echo "Building docker image..."
	@docker build -t $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG) .

image:
	@echo "Baking the image...."
	$(MAKE) build
	$(MAKE) push

.PHONY:push
push:
	@echo "Pushing the docker image for $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG) and $(IMAGE_REPO)/$(IMAGE_NAME):latest..."
	@docker tag $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_REPO)/$(IMAGE_NAME):latest
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG)
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME):latest

scan:
	@echo "Scanning docker image with SNYK..."
	@docker scan $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG)

release-chart:
	@echo "Releasing chart..."
	$(MAKE) -C ./ssl cert
	./helm/release_chart.sh $(NAMESPACE) $(CSR_NAME)
