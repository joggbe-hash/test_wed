FROM golang:1-alpine AS builder

WORKDIR /build

COPY shared /shared
COPY api/go.mod api/go.sum ./
RUN go mod download

COPY api .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server .

FROM alpine:3.21

RUN apk --no-cache add ca-certificates

COPY --from=builder /server /server

EXPOSE 5000

CMD ["/server"]
