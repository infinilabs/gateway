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
                    sh 'cd /home/jenkins/go/src/infini.sh/gateway && make clean config build-linux'
                    sh label: '', script: 'cd /home/jenkins/go/src/infini.sh/gateway/bin && tar cfz ${WORKSPACE}/gateway-$BUILD_NUMBER-linux64.tar.gz gateway-linux64 gateway.yml'
                    archiveArtifacts artifacts: 'gateway-$BUILD_NUMBER-linux64.tar.gz', fingerprint: true, followSymlinks: true, onlyIfSuccessful: true
                }
            }
        }

    }
}
