pipeline {
    agent any

    environment { 
        CI = 'true'
    }
    stages {
        
        stage('Build') {
            steps {
                catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE'){
                    sh 'cd /home/jenkins/go/src/infini.sh/gateway && make config build'
                }
            }
        }

    }
}
