PROJECT := placement-request-controller

.PHONY: build
build:
	@echo "building placement-request-controller..."
	CGO_ENABLED=0 go build -o _output/bin/placement-request-controller ./cmd/placement-request-controller

.PHONY: generate
generate: generate-code generate-crds

.PHONY: generate-crds
generate-crds:
	@echo "generating crds..."
	go tool controller-gen crd paths=./pkg/apis/v1alpha1 output:crd:dir=./manifests

.PHONY: generate-code
generate-code:
	@echo "updating codegen..."
	./hack/update-codegen.sh

.PHONY: clean
clean:
	rm -rf _output
