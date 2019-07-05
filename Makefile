eksphemeral_version:= v0.4.0
eksctl_base_image:= base
eksctl_deluxe_image:= deluxe

.PHONY: build publish publishbaseimg publishdeluxeimg

build:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=${eksphemeral_version}" -o bin/eksp-macos .
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=${eksphemeral_version}" -o bin/eksp-linux .

publishbaseimg:
	docker build --file Dockerfile.${eksctl_base_image} --tag quay.io/mhausenblas/eksctl:$(eksctl_base_image) .
	docker push quay.io/mhausenblas/eksctl:$(eksctl_base_image)

publishdeluxeimg:
	docker build --file Dockerfile.${eksctl_deluxe_image} --tag quay.io/mhausenblas/eksctl:$(eksctl_deluxe_image) .
	docker push quay.io/mhausenblas/eksctl:$(eksctl_deluxe_image)

publish: publishbaseimg publishdeluxeimg