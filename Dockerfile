FROM golang:1.21-bullseye AS builder

WORKDIR /src

# Caching step
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /go-tezos-keygen

FROM debian:bullseye-slim as runtime

WORKDIR /app

COPY --from=builder /go-tezos-keygen /app/go-tezos-keygen

EXPOSE 3000

ENTRYPOINT ["/app/go-tezos-keygen"]
