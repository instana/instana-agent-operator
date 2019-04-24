FROM openjdk:11
MAINTAINER Instana, support@instana.com
COPY ${project.build.directory}/${project.artifactId}-${project.version}-runner.jar ./instana-agent-operator.jar
EXPOSE 8080
CMD ["java", "-jar", "instana-agent-operator.jar"]
