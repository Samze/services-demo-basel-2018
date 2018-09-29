FROM golang:1.10-alpine3.7
RUN apk add --no-cache git
RUN go get -u github.com/cloudfoundry-community/go-cfenv
RUN go get -u github.com/lib/pq
RUN go get -u cloud.google.com/go/pubsub
ADD . /go/src/github.com/Samze/services-demo-basel-2018/worker-app
WORKDIR /go/src/github.com/Samze/services-demo-basel-2018/worker-app
RUN go install -i

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=0 /go/bin/worker-app .
CMD ["./worker-app"]
