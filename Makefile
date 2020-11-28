.PHONY: build clean deploy test

build:
	env GOOS=linux go build -ldflags="-s -w" -o bin/api api/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/extractor ./extractor/main.go

clean:
	rm -rf ./bin

deploy: clean build
	sls deploy --verbose

test: clean
	go test