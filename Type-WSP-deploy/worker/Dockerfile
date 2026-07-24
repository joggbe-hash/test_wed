FROM golang:1-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder

WORKDIR /build

COPY shared /shared
COPY worker/go.mod worker/go.sum ./
RUN go mod download

COPY worker .
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker .

FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d

RUN apk --no-cache add ca-certificates

COPY --from=builder /worker /worker

CMD ["/worker"]
