FROM golang:latest as builder
WORKDIR /go/src/github.com/jdnielss/gofile
COPY . .
RUN GOOS=linux go build -o gofile .

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/github.com/jdnielss/gofile/gofile .
ENTRYPOINT ["./gofile"]
