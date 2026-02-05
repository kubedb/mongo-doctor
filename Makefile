
IMAGE_NAME     := ghcr.io/arnobkumarsaha/mongo-doctor
PLATFORMS      := linux/amd64,linux/arm64

# Tags – you can add :latest, :v1.2.3, sha-xxx, etc.
TAG            ?= latest
FULL_IMAGE     := $(IMAGE_NAME):$(TAG)

# ────────────────────────────────────────────────

builder:
	docker buildx create --name multi-builder --driver docker-container --use
	docker buildx inspect --bootstrap

.PHONY: image
image:
	docker buildx build \
		--builder multi-builder \
		--platform $(PLATFORMS) \
		--tag $(FULL_IMAGE) \
		--push \
		--pull \
		.

# Optional: also build & load locally (useful for quick testing on current machine)
image-local:
	docker buildx build \
		--builder multi-builder \
		--platform $(PLATFORMS) \
		--tag $(FULL_IMAGE) \
		--load \
		.

all: image
	kubectl apply -f yamls/rbac.yaml
	kubectl apply -f yamls/job.yaml

run:
	kubectl delete -f yamls/job.yaml || true
	$(MAKE) image
	kubectl apply -f yamls/job.yaml

#image:
#	docker build -t ghcr.io/arnobkumarsaha/mongo-doctor .
#	docker push ghcr.io/arnobkumarsaha/mongo-doctor
#	#kind load docker-image ghcr.io/arnobkumarsaha/mongo-doctor

re:
	kubectl delete -f yamls/job.yaml || true
	kubectl apply -f yamls/job.yaml

clean:
	kubectl delete -f yamls/job.yaml
	kubectl delete -f yamls/rbac.yaml