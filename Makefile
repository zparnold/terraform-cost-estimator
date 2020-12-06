.PHONY: build clean deploy test

build:
	env GOOS=linux go build -ldflags="-s -w" -o bin/api api/main.go

clean:
	rm -rf ./bin

deploy: clean build
	sls deploy --verbose --stage prod --region us-east-2

deploy-dev: clean build
	sls deploy --verbose --stage dev --region us-east-2

test: clean
	go test