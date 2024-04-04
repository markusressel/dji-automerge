BINARY_NAME=dji-automerge
OUTPUT_DIR=bin/

test:   ## Run all tests
	@go clean --testcache && go test -v ./...

build: clean
	go build -o ${OUTPUT_DIR}${BINARY_NAME} main.go

run: build
	./${OUTPUT_DIR}${BINARY_NAME}

deploy: build
	mkdir -p ~/.custom/bin/
	cp ./${OUTPUT_DIR}${BINARY_NAME} ~/.custom/bin/${BINARY_NAME}

clean:
	go clean
	rm -rf ${OUTPUT_DIR}${BINARY_NAME}