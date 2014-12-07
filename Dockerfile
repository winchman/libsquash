FROM golang:1.3.3

ENV HOME /root
ENV DEBIAN_FRONTEND noninteractive

WORKDIR $GOPATH/src/github.com/jwilder/docker-squash
ADD Deps Deps
RUN go get github.com/hamfist/deppy && \
  deppy restore && \
  rm -rf $GOPATH/src/github.com/hamfist/deppy && \
  rm -rf $GOPATH/pkg/github.com/hamfist/deppy && \
  rm -f $GOPATH/bin/deppy

ADD . $GOPATH/src/github.com/jwilder/docker-squash

RUN go get ./...
