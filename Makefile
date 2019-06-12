eksphemeral_version:= v0.2.0
eksctl_version := 0.2

.PHONY: build publishimg

build:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=${eksphemeral_version}" -o bin/eksp-macos .
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=${eksphemeral_version}" -o bin/eksp-linux .

publishimg:
	docker build -t quay.io/mhausenblas/eksctl:$(eksctl_version) .
	docker push quay.io/mhausenblas/eksctl:$(eksctl_version)