FROM golang:alpine

RUN  apk update \
  && apk add --no-cache shadow git bash perl openssh vim tar curl python3 python3-dev py-pip gcc g++ libcurl make\
  && usermod -s /bin/bash root \
  && rm -rf /var/cache/apk/*

RUN mkdir -p /go/src/infini.sh

WORKDIR /go/src/infini.sh/
COPY ./framework /go/src/infini.sh/framework
COPY ./gateway /go/src/infini.sh/gateway
COPY ./license /go/src/infini.sh/license
COPY ./vendor /go/src/infini.sh/vendor

RUN cd gateway && OFFLINE_BUILD=true make config build
RUN chmod a+x /go/src/infini.sh/gateway/bin/gateway

FROM alpine:latest
WORKDIR /
COPY  --from=0 /go/src/infini.sh/gateway/bin/gateway /gateway
COPY  --from=0 /go/src/infini.sh/gateway/bin/gateway.yml /gateway.yml
EXPOSE 2900
CMD ["/gateway"]