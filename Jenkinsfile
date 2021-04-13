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
                    sh 'cd /home/jenkins/go/src/infini.sh/gateway && make config build-linux tar'
                    archiveArtifacts artifacts: '/home/jenkins/go/src/infini.sh/gateway/bin/gateway.tar.gz', fingerprint: true, followSymlinks: false, onlyIfSuccessful: true
                }
            }
        }

    }
}
