FROM golang:1-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server .

FROM alpine:3.21

RUN apk --no-cache add ca-certificates

COPY --from=builder /server /server

EXPOSE 5000

CMD ["/server"]
