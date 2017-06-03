FROM alpine

RUN apk add --update go git

ENV GOROOT=/usr/lib/go
ENV GOPATH=/go
ENV GOBIN=/go/bin
ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin:/usr/local/bin

WORKDIR /go/src/github.com/lavrs/blockchain
ADD . /go/src/github.com/lavrs/blockchain

RUN go get ./... \
    && go install \
    && go build

ENTRYPOINT ["/go/bin/blockchain"]