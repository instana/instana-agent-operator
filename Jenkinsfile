
def IMAGE_NAME = 'instana/instana-agent-operator'
def DOCKERFILE = './src/main/docker/Dockerfile.jvm'

pipeline {
  agent any

  stages {
    stage('Maven Package') {
      agent {
        docker {
          image 'openjdk:8'
          reuseNode true
        }
      }
      steps {
        script {
          def TAG = gitTag()
          def VERSION = mavenVersion(TAG)

          sh "./mvnw versions:set -DnewVersion=$VERSION"

          sh "./mvnw -C -B clean package"
        }
      }
    }

    stage('Docker Build') {
      steps {
        script {
          def TAG = gitTag()
          def VERSION = dockerVersion(TAG)

          docker.withRegistry('https://index.docker.io/v1/', '8a04e3ab-c6db-44af-8198-1beb391c98d2') {
            def image = docker.build("$IMAGE_NAME:$VERSION", "-f $DOCKERFILE --build-arg VERSION=$VERSION --build-arg BUILD=$BUILD_NUMBER .")

            image.push()

            if (TAG && isFullRelease(TAG)) {
              image.push('latest')
            } else {
              echo "Skipping pushing latest tag because this is a pre-release or branch."
            }
          }
        }
      }
    }
  }
}

def mavenVersion(tag) {
  return tag ? versionFromTag(tag) : '1.0.0-SNAPSHOT'
}

def dockerVersion(tag) {
  return tag ? versionFromTag(tag) : env.BRANCH_NAME
}

def isFullRelease(tag) {
  def isPrerelease = (tag ==~ /^.*-.*$/)
  return !isPrerelease;
}

def versionFromTag(git_tag) {
  return (git_tag =~ /v?([0-9A-Za-z.-]+)/)[0][1]
}

def gitTag() {
  return sh(returnStdout: true, script: 'git tag --contains | head -1').trim()
}