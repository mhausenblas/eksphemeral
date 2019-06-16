ui_version := 0.1
AWS_ACCESS_KEY_ID := $(shell aws configure get aws_access_key_id)
AWS_SECRET_ACCESS_KEY := $(shell aws configure get aws_secret_access_key)
AWS_DEFAULT_REGION := $(shell aws configure get region)

.PHONY: build run stop

build:
	GOOS=linux GOARCH=amd64 go build -o ./proxy main.go
	sed -i '.tmp' "s|EKSPHEMERAL_URL|$(EKSPHEMERAL_URL)|" js/app.js
	docker build --tag quay.io/mhausenblas/eksp-ui:$(ui_version) .
	mv js/app.js.tmp js/app.js
	# docker push quay.io/mhausenblas/eksp-ui:$(ui_version)

run:
	docker run	--name ekspui \
				--rm --detach \
				--publish 8080:8080 \
				quay.io/mhausenblas/eksp-ui:$(ui_version)

stop:
	docker kill ekspui