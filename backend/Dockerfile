FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /agent-os ./cmd/server

FROM alpine:latest
RUN apk add --no-cache bash
WORKDIR /app
COPY --from=builder /agent-os .
EXPOSE 3112
CMD ["/app/agent-os"]
