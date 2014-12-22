FROM golang:1.3.3

ENV HOME /root
ENV DEBIAN_FRONTEND noninteractive

WORKDIR $GOPATH/src/github.com/rafecolton/libsquash
ADD Deps Deps
RUN go get github.com/hamfist/deppy && \
  deppy restore && \
  rm -rf $GOPATH/src/github.com/hamfist/deppy && \
  rm -rf $GOPATH/pkg/github.com/hamfist/deppy && \
  rm -f $GOPATH/bin/deppy
RUN apt-get update -y \
  && apt-get install -y --no-install-recommends \
    curl \
    vim \
    less \
  && curl -sSL https://get.docker.com/ | sh

RUN export BIN=/usr/local/bin && \
  mkdir -p "$BIN" && \
  curl -sL http://stedolan.github.io/jq/download/linux64/jq -o $BIN/jq && \
  chmod +x $BIN/jq

ADD . $GOPATH/src/github.com/rafecolton/libsquash
ADD .docker/extract /usr/local/bin/extract
RUN go get ./... && go install ./docker-squash
