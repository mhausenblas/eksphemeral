EKSPHEMERAL_STACK_NAME?=eksp
EKSPHEMERAL_SVC_BUCKET?=eksp-svc
EKSPHEMERAL_CLUSTERMETA_BUCKET?=eks-cluster-meta

.PHONY: deps clean deploy destroy status

deps:
	go get -u ./...

clean: 
	rm ./bin/*
	rm eksp-stack.yaml
	
build:
	GOOS=linux GOARCH=amd64 go build -v -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/status ./status
	GOOS=linux GOARCH=amd64 go build -v -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/createcluster ./createcluster
	GOOS=linux GOARCH=amd64 go build -v -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/destroycluster ./destroycluster

up: 
	sam package --template-file template.yaml --output-template-file eksp-stack.yaml --s3-bucket ${EKSPHEMERAL_SVC_BUCKET}
	sam deploy --template-file eksp-stack.yaml --stack-name ${EKSPHEMERAL_STACK_NAME} --capabilities CAPABILITY_IAM --parameter-overrides ClusterMetadataBucketName=${EKSPHEMERAL_CLUSTERMETA_BUCKET} 

deploy: build up

destroy: 
	aws s3 rm s3://${EKSPHEMERAL_CLUSTERMETA_BUCKET} --recursive
	aws cloudformation delete-stack --stack-name ${EKSPHEMERAL_STACK_NAME}

status:
	aws cloudformation describe-stacks --stack-name ${EKSPHEMERAL_STACK_NAME}