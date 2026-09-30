.PHONY: install run test build clean fmt vet check docker docker-run

APP := go_redis
IMAGE ?= $(APP):local
VOLUME ?= go_redis_data

install:
	go mod download

run:
	go run .

test:
	go test -v ./...

check:
	go vet ./...
	go test -race ./...

docker:
	docker build --pull -t $(IMAGE) .

docker-run:
	docker run --rm --name go_redis -p 127.0.0.1:6379:6379 \
		-e REDIS_PASSWORD -v $(VOLUME):/data $(IMAGE)

build:
	go build -trimpath -o bin/$(APP) .

vet:
	go vet ./...

clean:
	rm -f bin/$(APP)
