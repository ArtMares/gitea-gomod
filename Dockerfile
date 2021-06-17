FROM golang:1.16 as builder

ENV PROJECT_PATH github.com/artmares/gitea-gomod

RUN mkdir -p $GOPATH/src/$PROJECT_PATH
COPY ./ $GOPATH/src/$PROJECT_PATH
WORKDIR $GOPATH/src/$PROJECT_PATH
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/gitea-gomod .

FROM alpine:latest
MAINTAINER ArtMares <artmares@influ.su>

RUN apk update \
    && apk add ca-certificates \
    && rm -rf /var/cache/apk/* \
    && mkdir -p /app \
    && update-ca-certificates
COPY --from=builder /go/src/github.com/artmares/gitea-gomod/build/gitea-gomod /app/
RUN chmod +x /app/gitea-gomod
WORKDIR /app
CMD ["/app/gitea-gomod"]