ui_version := 0.1

.PHONY: build

build:
	GOOS=linux GOARCH=amd64 go build -o ./proxy main.go
	docker build --tag quay.io/mhausenblas/eksp-ui:$(ui_version) .
	docker push quay.io/mhausenblas/eksp-ui:$(ui_version)

run:
	docker run --rm --detach --publish 8080:8080 quay.io/mhausenblas/eksp-ui:$(ui_version)
