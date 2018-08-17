#!/usr/bin/make -f

OPERATOR="previousnext/solr-operator"

build:
	operator-sdk build ${OPERATOR}

push:
	docker push ${OPERATOR}

reload:
	kubectl -n kube-system get pods | grep operator | awk '{print $$1}' | xargs kubectl -n kube-system delete pod

dev-reload: build push reload

.PHONY: build push reload dev-reload