pipeline {

    agent none

    environment { 
        CI = 'true'
    }
    stages {
        
        stage('Build Linux Packages') {

            agent {
                label 'linux'
            }

            steps {
                catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE'){
                    sh 'cd /home/jenkins/go/src/infini.sh/gateway && git stash && git pull origin master && make clean config build-linux build-arm'
                    sh label: 'package-linux64', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-linux64.tar.gz gateway-linux64 gateway.yml ../sample-configs'
                    sh label: 'package-linux32', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-linux32.tar.gz gateway-linux32 gateway.yml ../sample-configs'
                    sh label: 'package-arm5', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-arm5.tar.gz gateway-armv5 gateway.yml ../sample-configs'
                    sh label: 'package-arm6', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-arm6.tar.gz gateway-armv6 gateway.yml ../sample-configs'
                    sh label: 'package-arm7', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-arm7.tar.gz gateway-armv7 gateway.yml ../sample-configs'
                    sh label: 'package-arm64', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$VERSION-$BUILD_NUMBER-arm64.tar.gz gateway-arm64 gateway.yml ../sample-configs'
                    archiveArtifacts artifacts: 'gateway-$VERSION-$BUILD_NUMBER-*.tar.gz', fingerprint: true, followSymlinks: true, onlyIfSuccessful: true
                    sh label: 'docker-build', script: 'cd /home/jenkins/go/src/infini.sh/ && docker build -t infini-gateway  -f gateway/docker/Dockerfile .'
                    sh label: 'docker-tagging', script: 'docker tag infini-gateway medcl/infini-gateway:latest && docker tag infini-gateway medcl/infini-gateway:$VERSION-$BUILD_NUMBER'
                    sh label: 'docker-push', script: 'docker push medcl/infini-gateway:latest && docker push medcl/infini-gateway:$VERSION-$BUILD_NUMBER'
                }
            }
        }

    }
}
