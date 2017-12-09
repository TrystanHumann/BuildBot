## STAGE 1: GO GET FILES ##
FROM golang:1.9.0 as gofileget
LABEL vendor="Xd" \
  version="17.09.0-ce" \
  author="Trystan Humann <humanntrystan@hotmail.com>"

WORKDIR  /usr/local/go/src/github.com/buildbot/
ADD main.go .
RUN mkdir utils
RUN mkdir models
ADD utils ./utils
ADD models ./models
RUN go get ./...
## STAGE 2: BUILD IMAGE FOR ALPINE ##
FROM golang:1.9.0-alpine as gobuild

WORKDIR /usr/local/go/src
COPY --from=gofileget /usr/local/go/src /usr/local/go/src
COPY --from=gofileget /go/src /go/src
WORKDIR /usr/local/go/src/github.com/buildbot/
RUN go build -o build-bot-executable

## STAGE 3: RUN THE APP ##
FROM alpine
WORKDIR /app
COPY --from=gobuild /usr/local/go/src/github.com/buildbot/build-bot-executable /app
COPY --from=gofileget /usr/local/go/src/github.com/buildbot/my.db /app
COPY --from=gofileget /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ADD ./conf.json .

ENTRYPOINT /app/build-bot-executable