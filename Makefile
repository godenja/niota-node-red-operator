# Image repository and tag used when building / pushing the operator container.
IMG ?= ghcr.io/godenja/niota-node-red-operator:latest

# controller-gen version used to regenerate CRD manifests and deepcopy code.
CONTROLLER_GEN_VERSION ?= v0.14.0

# Directory that contains the generated CRD manifests.
CRD_DIR := config/crd/bases

# ── tooling paths (local bin) ─────────────────────────────────────────────────
LOCALBIN ?= $(shell pwd)/bin
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: deps
deps: ## Download and tidy Go modules
	go mod tidy

.PHONY: fmt
fmt: ## Run go fmt
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: test
test: fmt vet ## Run unit tests
	go test ./... -v -count=1

##@ Code generation

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)

.PHONY: generate
generate: $(CONTROLLER_GEN) ## Regenerate zz_generated.deepcopy.go
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/..."

.PHONY: manifests
manifests: $(CONTROLLER_GEN) ## Regenerate CRD manifests from Go types
	$(CONTROLLER_GEN) rbac:roleName=niota-node-red-operator crd paths="./..." output:crd:artifacts:config=$(CRD_DIR)

##@ Build

.PHONY: build
build: deps fmt vet ## Build the manager binary
	go build -o bin/manager ./main.go

.PHONY: run
run: deps fmt vet ## Run the operator locally (needs a valid kubeconfig)
	go run ./main.go

.PHONY: docker-build
docker-build: ## Build the operator container image
	docker build -t $(IMG) .

.PHONY: docker-push
docker-push: ## Push the operator container image
	docker push $(IMG)

##@ Deployment

.PHONY: install
install: manifests ## Install CRDs into the cluster
	kubectl apply -f $(CRD_DIR)

.PHONY: uninstall
uninstall: ## Uninstall CRDs from the cluster
	kubectl delete -f $(CRD_DIR) --ignore-not-found

.PHONY: deploy
deploy: manifests ## Deploy the operator and RBAC into the cluster
	kubectl apply -f config/rbac/service_account.yaml
	kubectl apply -f config/rbac/role.yaml
	kubectl apply -f config/rbac/role_binding.yaml
	kubectl apply -f config/manager/manager.yaml

.PHONY: undeploy
undeploy: ## Remove the operator from the cluster
	kubectl delete -f config/manager/manager.yaml --ignore-not-found
	kubectl delete -f config/rbac/role_binding.yaml --ignore-not-found
	kubectl delete -f config/rbac/role.yaml --ignore-not-found
	kubectl delete -f config/rbac/service_account.yaml --ignore-not-found

.PHONY: sample
sample: ## Apply the sample NodeRedInstance CR
	kubectl apply -f config/samples/niota_v1alpha1_noderedinstance.yaml
