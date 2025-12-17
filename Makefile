REGISTRY_URL := 192.168.1.196:30500
IMAGE_NAME := orb-backend
TAG := latest

.PHONY: build push

build:
	@echo "Building Docker image..."
	docker build -t $(REGISTRY_URL)/$(IMAGE_NAME):$(TAG) -f backend/Dockerfile backend

push:
	@echo "Pushing Docker image to registry..."
	docker push $(REGISTRY_URL)/$(IMAGE_NAME):$(TAG)

deploy:
	@echo "Deploying Docker image to kubernetes..."
	kubectl apply -f orb-backend-k8s.yaml
