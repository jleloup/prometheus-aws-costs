########
# Helm #
########

HELM_OUT="/tmp/helm-render.yaml"
KUBERNETES_VERSION="1.30.0"
chart=helm/chart

dependency-update := true

ifeq ($(dependency-update), true)
	override extra_args += --dependency-update
endif

update-snapshot := false

ifeq ($(update-snapshot), true)
	override unit_test_args += --update-snapshot
endif

.PHONY: helm-dependency
helm-dependency:
	cd $(chart); helm dependency build

.PHONY: helm-lint
helm-lint:
	cd $(chart); helm lint

.PHONY: helm-template
helm-template:
	cd $(chart); \
	helm template fake-render . -f values.yaml -f values-dev.yaml $(extra_args) \
	| tee $(HELM_OUT)

.PHONY: kubeconform
kubeconform:
	kubeconform \
	-summary \
	-ignore-missing-schemas \
	-kubernetes-version $(KUBERNETES_VERSION) \
	-schema-location default \
	-schema-location 'https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json' \
	$(HELM_OUT)

.PHONY: helm-docs
helm-docs:
	cd $(chart); helm-docs

.PHONY: helm-package
helm-package:
	helm package $(extra_args) $(chart)

.PHONY: helm-unit-test
helm-unit-test:
	helm unittest $(unit_test_args) $(chart)
