pipeline {

    agent none

    environment { 
        CI = 'true'
    }
    stages {

       stage('build') {

        parallel {

        stage('Build Linux Packages') {

            agent {
                label 'linux'
            }

            steps {
                catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE'){
                    sh 'cd /home/jenkins/go/src/infini.sh/gateway && git stash && git pull origin master && make clean config build-linux'
                    sh label: 'package-linux-amd64', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-linux-amd64.tar.gz gateway-linux-amd64 gateway.yml ../sample-configs'
                    sh label: 'package-linux-386', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-linux-386.tar.gz gateway-linux-386 gateway.yml ../sample-configs'
                    sh label: 'package-linux-mips', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-linux-mips.tar.gz gateway-linux-mips gateway.yml ../sample-configs'
                    sh label: 'package-linux-mipsle', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-linux-mipsle.tar.gz gateway-linux-mipsle gateway.yml ../sample-configs'
                    sh label: 'package-linux-mips64', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-linux-mips64.tar.gz gateway-linux-mips64 gateway.yml ../sample-configs'
                    sh label: 'package-linux-mips64le', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-linux-mips64le.tar.gz gateway-linux-mips64le gateway.yml ../sample-configs'

                    sh 'cd /home/jenkins/go/src/infini.sh/gateway && git stash && git pull origin master && make config build-arm'
                    sh label: 'package-linux-arm5', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-linux-arm5.tar.gz gateway-linux-armv5 gateway.yml ../sample-configs'
                    sh label: 'package-linux-arm6', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-linux-arm6.tar.gz gateway-linux-armv6 gateway.yml ../sample-configs'
                    sh label: 'package-linux-arm7', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-linux-arm7.tar.gz gateway-linux-armv7 gateway.yml ../sample-configs'
                    sh label: 'package-linux-arm64', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-linux-arm64.tar.gz gateway-linux-arm64 gateway.yml ../sample-configs'

                    sh 'cd /home/jenkins/go/src/infini.sh/gateway && git stash && git pull origin master && make config build-darwin'
                    sh label: 'package-mac-amd64', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && zip -r ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-mac-amd64.zip gateway-mac-amd64 gateway.yml ../sample-configs'
                    sh label: 'package-mac-arm64', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && zip -r ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-mac-arm64.zip gateway-mac-arm64 gateway.yml ../sample-configs'

                    sh 'cd /home/jenkins/go/src/infini.sh/gateway && git stash && git pull origin master && make config build-win'
                    sh label: 'package-win-amd64', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && zip -r ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-windows-amd64.zip gateway-windows-amd64.exe gateway.yml ../sample-configs'
                    sh label: 'package-win-386', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && zip -r ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-windows-386.zip gateway-windows-386.exe gateway.yml ../sample-configs'
                    archiveArtifacts artifacts: 'gateway-$VERSION-$BUILD_NUMBER-*', fingerprint: true, followSymlinks: true, onlyIfSuccessful: false
                }
            }
         }
        }
      }
    }
}
