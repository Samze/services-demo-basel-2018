FROM golang:1.10-alpine3.7
RUN apk add --no-cache git
RUN go get -u github.com/cloudfoundry-community/go-cfenv
RUN go get -u github.com/lib/pq
RUN go get -u cloud.google.com/go/pubsub
ADD ./web-app /go/src/github.com/Samze/services-demo-basel-2018/web-app
ADD ./config /go/src/github.com/Samze/services-demo-basel-2018/config
WORKDIR /go/src/github.com/Samze/services-demo-basel-2018/web-app
RUN go install -i

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY ./web-app /web-app
WORKDIR /web-app
COPY --from=0 /go/bin/web-app /web-app
CMD ["./web-app"]
