# Build stage
FROM golang:1.27.1-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main ./cmd/psycho

# Run stage - using Alpine for very small image
FROM alpine:3.21

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/main /app/main
COPY config/ /app/config/
COPY modules/analyze/dictionary.json /app/modules/analyze/dictionary.json
COPY samples/ /app/samples/

EXPOSE 8080
ENTRYPOINT ["/app/main"]
