FROM golang:alpine as builder

WORKDIR $GOPATH/src/node-label-controller

COPY ./config config
COPY ./controller controller
COPY ./main.go .
COPY ./vendor vendor

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -v

FROM alpine:latest
COPY --from=builder /go/$GOPATH/src/node-label-controller/node-label-controller /bin/node-label-controller