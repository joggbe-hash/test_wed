FROM golang:1-alpine AS builder

WORKDIR /build

COPY shared /shared
COPY worker/go.mod worker/go.sum ./
RUN go mod download

COPY worker .
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker .

FROM alpine:3.21

RUN apk --no-cache add ca-certificates

COPY --from=builder /worker /worker

CMD ["/worker"]
