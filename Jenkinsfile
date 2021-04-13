pipeline {
    agent any

    environment { 
        CI = 'true'
    }
    stages {
        
        stage('Build Linux Packages') {
            steps {

               agent {
                  node {
                     label 'linux'
                     customWorkspace '/home/jenkins/'
                     reuseNode true
                  }
               }

                catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE'){
                    sh 'cd /home/jenkins/go/src/infini.sh/gateway && make config build-linux tar'
                    archiveArtifacts artifacts: '/home/jenkins/go/src/infini.sh/gateway/bin/gateway.tar.gz', fingerprint: true, followSymlinks: false, onlyIfSuccessful: true
                }
            }
        }

    }
}
