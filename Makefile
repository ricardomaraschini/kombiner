# Copyright 2025 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

IMAGE_NAME ?= kombiner
IMAGE_TAG ?= latest

.PHONY: build
build: build-controller build-scheduler

.PHONY: build-image-and-push
build-image-and-push: build-image push-image

.PHONY: build-image-and-save
build-image-and-save: build-image save-image

.PHONY: generate
generate: generate-code generate-crds

.PHONY: verify-boilerplates
verify-boilerplates:
	go tool boilersuite --author "Kubernetes" .

.PHONY: install
install:
	helm install kombiner -n kube-system ./helm \
		--wait --timeout 5m0s \
		--set image.repository=${IMAGE_NAME} \
		--set image.tag=${IMAGE_TAG}

.PHONY: uninstall
uninstall:
	helm uninstall -n kube-system kombiner

.PHONY: build-controller
build-controller:
	CGO_ENABLED=0 go build -o _output/bin/kombiner-controller ./cmd/kombiner-controller

.PHONY: build-scheduler
build-scheduler:
	CGO_ENABLED=0 go build -o _output/bin/kombiner-scheduler ./cmd/kombiner-scheduler

.PHONY: generate-crds
generate-crds:
	go tool controller-gen crd paths=./pkg/apis/kombiner/v1alpha1 output:crd:dir=./helm/crds/

.PHONY: generate-code
generate-code:
	./hack/update-codegen.sh

.PHONY: test-unit
test-unit:
	./hack/run-unit-tests.sh

.PHONY: test-e2e
test-e2e:
	go tool ginkgo run -v e2e/

.PHONY: build-image
build-image:
	docker build -t ${IMAGE_NAME}:${IMAGE_TAG} .

.PHONY: push-image
push-image: build-image
	docker push ${IMAGE_NAME}:${IMAGE_TAG}

.PHONY: save-image
save-image:
	mkdir -p _output/images/
	docker save ${IMAGE_NAME}:${IMAGE_TAG} -o _output/images/kombiner.tar

.PHONY: clean
clean:
	rm -rf _output

.PHONY: format
format:
	./hack/update-gofmt.sh
