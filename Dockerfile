FROM golang:1.8-alpine

# RUN apk add --update git curl

WORKDIR /go/src/app
COPY src/ .

# RUN go get github.com/c-bata/go-prompt

RUN go install -v

CMD ["app"]
