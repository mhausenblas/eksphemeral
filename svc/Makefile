.PHONY: deps clean deploy status

deps:
	go get -u ./...

clean: 
	rm ./bin/*
	rm eksp-stack.yaml
	
build:
	GOOS=linux GOARCH=amd64 go build -v -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/status ./status
	GOOS=linux GOARCH=amd64 go build -v -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/createcluster ./createcluster
	GOOS=linux GOARCH=amd64 go build -v -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/destroycluster ./destroycluster

deploy: build
	sam package --template-file template.yaml --output-template-file eksp-stack.yaml --s3-bucket eks-svc
	sam deploy --template-file eksp-stack.yaml --stack-name eksp --capabilities CAPABILITY_IAM

status:
	aws cloudformation describe-stacks --stack-name eksp