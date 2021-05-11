FROM golang:1.16-alpine AS builder
RUN apk update && apk add --no-cache git && apk add -U --no-cache ca-certificates
WORKDIR /app/
ADD go.mod go.sum ./
RUN go mod download
ADD . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o gopengraph-image-gen .

FROM westy92/headless-chrome-alpine
WORKDIR /app/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/gopengraph-image-gen /app/gopengraph-image-gen
RUN mkdir -p /tmp
ENTRYPOINT ["/app/gopengraph-image-gen"]
