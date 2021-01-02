FROM alpine

RUN apk update

RUN apk add jsonnet --repository=http://dl-cdn.alpinelinux.org/alpine/edge/community

RUN apk add python3 bash git zip jq curl

RUN apk add py3-pip

RUN pip3 install --upgrade pip

RUN pip3 install pyyaml requests semver

COPY olm/ olm/

COPY deploy/ deploy/
