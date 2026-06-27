FROM golang:1.26.4-alpine AS build

WORKDIR /src

ARG VERSION=dev

COPY go.mod go.sum ./
COPY vendor ./vendor
RUN go env -w CGO_ENABLED=0

COPY . .
RUN go build -mod=vendor -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/simply-dashed ./main.go

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/simply-dashed /app/simply-dashed
COPY config.example.yaml /app/config.yaml

# We dont build the icons into the image, they will be downloaded on first run and cached in /app/data/icons
# COPY data/icons /app/data/icons

EXPOSE 8080

ENTRYPOINT ["/app/simply-dashed"]
CMD ["-config", "/app/config.yaml", "-icon-dir", "/app/data/icons", "-refresh-icons=false"]
