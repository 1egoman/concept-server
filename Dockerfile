FROM golang:1.8-alpine

RUN apk add --update git curl
# build-base

WORKDIR /go/src/app
COPY src/ .

RUN go get github.com/c-bata/go-prompt
# RUN go get github.com/GeertJohan/go.linenoise
RUN go get github.com/reiver/go-porterstemmer

RUN go install -v

CMD ["app"]
