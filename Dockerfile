FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /stockvacancy-api ./cmd/api

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /stockvacancy-api /stockvacancy-api

EXPOSE 8080
CMD ["/stockvacancy-api"]
