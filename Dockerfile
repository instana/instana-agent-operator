FROM registry.access.redhat.com/ubi7/ubi:latest
MAINTAINER Instana, support@instana.com

LABEL name="instana-agent-operator" \
      vendor="Instana" \
      version="v0.0.1" \
      release="1" \
      summary="Experimental alpha version of the upcoming Kubernetes Operator for the Instana APM Agent" \
      description="This operator will deploy a daemon set to run the Instana APM Agent on each cluster node."

# TODO: Build a stripped-down image with quarkus.

RUN yum install -y java-11-openjdk-devel
COPY licenses /licenses
COPY ${project.build.directory}/${project.artifactId}-${project.version}-runner.jar ./instana-agent-operator.jar
EXPOSE 8080
CMD ["java", "-jar", "instana-agent-operator.jar"]
