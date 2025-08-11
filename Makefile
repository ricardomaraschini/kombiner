PROJECT := kombiner

.PHONY: build
build: build-controller build-scheduler

.PHONY: build-controller
build-controller:
	@echo "building controller..."
	CGO_ENABLED=0 go build -o _output/bin/kombiner-controller ./cmd/kombiner-controller

.PHONY: build-scheduler
build-scheduler:
	@echo "building scheduler..."
	CGO_ENABLED=0 go build -o _output/bin/kombiner-scheduler ./cmd/kombiner-scheduler

.PHONY: generate
generate: generate-code generate-crds

.PHONY: generate-crds
generate-crds:
	@echo "generating crds..."
	go tool controller-gen crd paths=./pkg/apis/kombiner/v1alpha1 output:crd:dir=./manifests

.PHONY: generate-code
generate-code:
	@echo "updating codegen..."
	./hack/update-codegen.sh

.PHONY: clean
clean:
	rm -rf _output
