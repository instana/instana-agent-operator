FROM azul/zulu-openjdk:8
MAINTAINER Fabian Stäber <fabian.staeber@instana.com>
COPY target/instana-operator-1.0.0-SNAPSHOT-runner.jar .
EXPOSE 8080
CMD ["java", "-jar", "instana-operator-1.0.0-SNAPSHOT-runner.jar"]
