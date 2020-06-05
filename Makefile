.PHONY : prepare build run

SERVICE_NAME := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
$(eval $(SERVICE_NAME):;@:)
ifndef SERVICE_NAME
$(error SERVICE_NAME is not set)
endif

PROTO_FILES := $(shell find api/proto/$(SERVICE_NAME) -name '*.proto')

init:
	sh scripts/create_service.sh $(SERVICE_NAME)

prepare:
	ln -sf cmd/$(SERVICE_NAME)/main.go main_service.go

build: prepare
	go build -o bin

run: build
	./bin

proto:
	$(foreach proto_file, $(PROTO_FILES),\
	protoc -I . $(proto_file) --go_out=plugins=grpc:.;)

docker: prepare
	docker build --build-arg SERVICE_NAME=$(SERVICE_NAME) -t $(SERVICE_NAME):latest .

run-container:
	docker rm -f $(SERVICE_NAME)
	docker rmi $(docker images | grep "^<none>" | awk "{print $3}")
	docker run --name=$(SERVICE_NAME) --network="host" -d $(SERVICE_NAME)

clear:
	rm main_service.go bin backend-microservices warung wedding
