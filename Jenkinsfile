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
          def BUILD_ARGS = "-f $DOCKERFILE --build-arg VERSION=$VERSION --build-arg BUILD=$BUILD_NUMBER ."

          docker.withRegistry('https://index.docker.io/v1/', '8a04e3ab-c6db-44af-8198-1beb391c98d2') {
            def image = docker.build("instana/instana-agent-operator:$VERSION", BUILD_ARGS)

            image.push()

            if (isFullRelease(TAG)) {
              image.push('latest')
            } else {
              echo "Skipping pushing latest tag because this is a pre-release or branch."
            }
          }

          if (isFullRelease(TAG)) {
            docker.withRegistry('https://scan.connect.redhat.com/v1/', '60f49bbb-514e-4945-9c28-be68576d10e2') {
              // annoyingly no way to reuse the existing image with docker jenkins plugin.
              // probably should just pull all of this into a shell script
              def image = docker.build("scan.connect.redhat.com/ospid-6da7e6aa-00e1-4355-9c15-21d63fb091b6/instana-agent-operator:$VERSION", BUILD_ARGS)

              image.push()
            }
          }
        }
      }
    }

    stage('OLM Artifact Generation') {
      agent {
        dockerfile {
          filename 'olm/Dockerfile'
          reuseNode true
        }
      }
      steps {
        script {
          def TAG = gitTag()
          def VERSION = dockerVersion(TAG)

          sh "./olm/createCSV.sh $VERSION olm"

          sh "./olm/createCSV.sh $VERSION redhat registry.connect.redhat.com"
        }
      }

      post {
        success {
          archiveArtifacts artifacts: 'target/*.zip'
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
  if (!tag) {
    return false
  }
  def isPrerelease = (tag ==~ /^.*-.*$/)
  return !isPrerelease;
}

def versionFromTag(git_tag) {
  return (git_tag =~ /v?([0-9A-Za-z.-]+)\+?.*/)[0][1]
}

def gitTag() {
  return sh(returnStdout: true, script: 'git tag --contains | head -1').trim()
}