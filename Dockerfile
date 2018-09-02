FROM golang:1.11.0-alpine3.8 as compile
WORKDIR /go/src/github.com/nestorsokil/coding-guru/
COPY . .
RUN apk --update add ca-certificates \
&& apk add --no-cache git \
&& go get -u github.com/golang/dep/... \
&& dep ensure \
&& CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin .

FROM scratch as runtime
WORKDIR /root/
COPY --from=compile /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=compile /go/src/github.com/nestorsokil/coding-guru/bin .
CMD ["./bin"]