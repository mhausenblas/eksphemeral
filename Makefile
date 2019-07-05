eksphemeral_version:= v0.4.0
eksctl_base_image:= base
eksctl_deluxe_image:= deluxe

.PHONY: build publish pbase pdeluxe

build:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=${eksphemeral_version}" -o bin/eksp-macos .
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=${eksphemeral_version}" -o bin/eksp-linux .

pbase:
	docker build --file Dockerfile.${eksctl_base_image} --tag quay.io/mhausenblas/eksctl:$(eksctl_base_image) .
	docker push quay.io/mhausenblas/eksctl:$(eksctl_base_image)

pdeluxe:
	docker build --file Dockerfile.${eksctl_deluxe_image} --tag quay.io/mhausenblas/eksctl:$(eksctl_deluxe_image) .
	docker push quay.io/mhausenblas/eksctl:$(eksctl_deluxe_image)

publish: pbase pdeluxe