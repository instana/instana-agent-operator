FROM openjdk:11
MAINTAINER Fabian Stäber, fabian.staeber@instana.com
COPY target/instana-operator.jar .
EXPOSE 8080
CMD ["java", "-jar", "instana-operator.jar"]
