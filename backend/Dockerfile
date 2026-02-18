FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -p 1 -o /server ./cmd/server

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /server /server

EXPOSE 8080

CMD ["/server"]
