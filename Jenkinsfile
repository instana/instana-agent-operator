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
          def BUILD_ARGS = "-f $DOCKERFILE --pull --build-arg VERSION=$VERSION --build-arg BUILD=$BUILD_NUMBER ."

          if (isFullRelease(TAG)) {
            docker.withRegistry('https://index.docker.io/v1/', '8a04e3ab-c6db-44af-8198-1beb391c98d2') {
              def image = docker.build("instana/instana-agent-operator:$VERSION", BUILD_ARGS)
              image.push()
            }
          } else {
            echo "Skipping pushing tag because this is a pre-release or branch."
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
          filename 'olm/Dockerfile.ci'
          reuseNode true
        }
      }
      steps {
        script {
          def TAG = gitTag()
          def VERSION = dockerVersion(TAG)

          sh "./olm/create-artifacts.sh $VERSION olm"

          sh "./olm/create-artifacts.sh $VERSION redhat"

          withCredentials([string(credentialsId: 'GH_API_TOKEN', variable: 'GH_API_TOKEN')]) {
            if (isFullRelease(TAG)) {
              sh "./olm/operator-resources/create-github-release.sh $VERSION $GH_API_TOKEN"
            }
          }
        }
      }

      post {
        success {
          archiveArtifacts artifacts: 'target/redhat-*.zip'
          archiveArtifacts artifacts: 'target/olm-*.zip'
          archiveArtifacts artifacts: 'target/operator-resources/**/*'
        }
      }
    }

    stage('Upload RedHat Operator Bundle') {
      steps {
        script {
          def TAG = gitTag()
          def VERSION = dockerVersion(TAG)
          def BUNDLE_DIR = "target/downloads/${VERSION}/redhat-${VERSION}"
          def BUILD_ARGS = "-f olm/Dockerfile.bundle ${BUNDLE_DIR}"


          if (isFullRelease(TAG)) {
            withCredentials([string(credentialsId: 'GH_API_TOKEN', variable: 'GH_API_TOKEN')]) {
              sh "./olm/operator-resources/download-github-release-assets.sh $VERSION $GH_API_TOKEN"
            }

            sh "unzip ${BUNDLE_DIR}.zip -d ${BUNDLE_DIR}"

            docker.withRegistry('https://scan.connect.redhat.com/v1/', '5fc350a1-9257-4291-9f2a-df9257b9e791') {
              def image = docker.build("scan.connect.redhat.com/ospid-5fc350a1-9257-4291-9f2a-df9257b9e791/instana-agent-operator-bundle:$VERSION", BUILD_ARGS)
              image.push()
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
