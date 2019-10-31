FROM alpine

RUN apk update

RUN apk add jsonnet --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing

RUN apk add python3 bash git zip

RUN pip3 install --upgrade pip

RUN pip3 install pyyaml operator-courier

COPY olm/ olm/

COPY deploy/ deploy/
