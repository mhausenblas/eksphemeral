eksphemeral_version:= v0.2.0
eksctl_base_image:= base
eksctl_deluxe_image:= deluxe

.PHONY: build publishimg

build:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=${eksphemeral_version}" -o bin/eksp-macos .
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=${eksphemeral_version}" -o bin/eksp-linux .

publishbaseimg:
	docker build -t quay.io/mhausenblas/eksctl:$(eksctl_base_image) .
	docker push quay.io/mhausenblas/eksctl:$(eksctl_base_image)

publishdeluxeimg:
	docker build -t quay.io/mhausenblas/eksctl:$(eksctl_deluxe_image) .
	docker push quay.io/mhausenblas/eksctl:$(eksctl_deluxe_image)