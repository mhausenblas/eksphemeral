EKSPHEMERAL_STACK_NAME?=eksp
EKSPHEMERAL_SVC_BUCKET?=eks-svc
EKSPHEMERAL_CLUSTERMETA_BUCKET?=eks-cluster-meta

eksphemeral_version:= v0.3.0

.PHONY: deps clean build up downloadbin deploy destroy status

deps:
	go get -u ./...
	
build:
	GOOS=linux GOARCH=amd64 go build -v -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/status ./status
	GOOS=linux GOARCH=amd64 go build -v -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/createcluster ./createcluster
	GOOS=linux GOARCH=amd64 go build -v -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/destroycluster ./destroycluster
	GOOS=linux GOARCH=amd64 go build -v -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/prolongcluster ./prolongcluster

up: 
	sam package --template-file template.yaml --output-template-file eksp-stack.yaml --s3-bucket ${EKSPHEMERAL_SVC_BUCKET}
	sam deploy --template-file eksp-stack.yaml --stack-name ${EKSPHEMERAL_STACK_NAME} --capabilities CAPABILITY_IAM --parameter-overrides ClusterMetadataBucketName="${EKSPHEMERAL_CLUSTERMETA_BUCKET}" NotificationFromEmailAddress="${EKSPHEMERAL_EMAIL_FROM}"

downloadbin:
	mkdir -p bin
	curl -sL https://github.com/mhausenblas/eksphemeral/releases/download/${eksphemeral_version}/createcluster -o bin/createcluster
	curl -sL https://github.com/mhausenblas/eksphemeral/releases/download/${eksphemeral_version}/destroycluster -o bin/destroycluster
	curl -sL https://github.com/mhausenblas/eksphemeral/releases/download/${eksphemeral_version}/prolongcluster -o bin/prolongcluster
	curl -sL https://github.com/mhausenblas/eksphemeral/releases/download/${eksphemeral_version}/status -o bin/status
	chmod +x bin/*

deploy: build up

install: downloadbin up

destroy:
	aws s3 rm s3://${EKSPHEMERAL_CLUSTERMETA_BUCKET} --recursive
	aws cloudformation delete-stack --stack-name ${EKSPHEMERAL_STACK_NAME}

status:
	aws cloudformation describe-stacks --stack-name ${EKSPHEMERAL_STACK_NAME}