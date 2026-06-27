APP := simply-dashed
VERSION ?= dev

.PHONY: fmt test build docker

fmt:
	gofmt -w .

test:
	go test -mod=vendor ./...

build:
	go build -mod=vendor -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o dist/$(APP) ./main.go

serve:
	go run -mod=vendor ./main.go -config config.yaml

vendor-icons:
	go run -mod=vendor ./cmd/iconfetch -config config.yaml -icon-dir data/icons

docker:
	docker build --build-arg VERSION=$(VERSION) -t $(APP):$(VERSION) .
