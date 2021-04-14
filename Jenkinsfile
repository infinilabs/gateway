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
                    archiveArtifacts artifacts: 'bin/*.*', fingerprint: true, followSymlinks: true, onlyIfSuccessful: true
                }
            }
        }

    }
}
