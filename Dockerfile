#
# (c) Copyright IBM Corp. 2021
# (c) Copyright Instana Inc.
#

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.3
MAINTAINER Instana, support@instana.com

ARG VERSION=dev
ARG BUILD=1

LABEL name="instana-agent-operator" \
      vendor="Instana Inc" \
      maintainer="Instana Inc" \
      version=$VERSION \
      build=$BUILD \
      summary="Kubernetes / OpenShift Operator for the Instana APM Agent" \
      description="This operator will deploy a daemon set to run the Instana APM Agent on each cluster node."

ENV OPERATOR=instana-agent-operator \
    USER_UID=1001 \
    USER_NAME=instana-agent-operator

RUN  microdnf install unzip && microdnf clean all
COPY LICENSE /licenses/
COPY build/_output/bin /usr/local/bin
COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
