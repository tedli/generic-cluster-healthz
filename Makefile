.DEFAULT_GOAL := build
.PHONY: clean push build

TAG ?= $(shell date '+%Y%m%d%H%M')
IMAGE ?= tedli/generic-cluster-healthz

clean:
	docker images '$(IMAGE)*' --format='{{ .Repository }}:{{ .Tag }}' | xargs docker rmi

push:
	docker images '$(IMAGE)*' --format='{{ .Repository }}:{{ .Tag }}' | xargs -L1 docker push

build:
	docker build --no-cache -t $(IMAGE):$(TAG) -f Dockerfile .
