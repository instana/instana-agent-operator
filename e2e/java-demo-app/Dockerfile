FROM registry.access.redhat.com/ubi9/ubi-minimal:latest AS build
RUN microdnf update -y \
  && microdnf install -y java-21-openjdk-devel \
  && microdnf clean all
WORKDIR /app
COPY HelloWorldServer.java /app/
RUN javac HelloWorldServer.java

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
RUN microdnf update -y \
  && microdnf install -y java-21-openjdk-headless \
  && microdnf clean all
ENV USER_UID=1001
USER ${USER_UID}:${USER_UID}
WORKDIR /app
COPY --from=build --chown=${USER_UID}:${USER_UID} /app/*.class /app/
EXPOSE 8080

CMD ["java", "HelloWorldServer"]
