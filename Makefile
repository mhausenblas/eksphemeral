eksctl_version := 0.1

.PHONY: publishimg

publishimg:
	docker build -t quay.io/mhausenblas/eksctl:$(eksctl_version) .
	docker push quay.io/mhausenblas/eksctl:$(eksctl_version)