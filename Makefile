APP := simply-dashed

.PHONY: fmt test build docker

fmt:
	gofmt -w .

test:
	go test -mod=vendor ./...

build:
	go build -mod=vendor -trimpath -ldflags="-s -w" -o dist/$(APP) ./main.go

serve:
	go run -mod=vendor ./main.go -config config.yaml

vendor-icons:
	go run -mod=vendor ./cmd/iconfetch -config config.yaml -icon-dir data/icons

docker:
	docker build -t $(APP):dev .
