node('tos-builder') {
    properties([buildDiscarder(
            logRotator(artifactDaysToKeepStr: '', artifactNumToKeepStr: '', daysToKeepStr: '60', numToKeepStr: '100')),
                gitLabConnection('gitlab-172.16.1.41'),
                parameters([string(defaultValue: '', description: '', name: 'RELEASE_TAG')]),
                pipelineTriggers([])
    ])


    currentBuild.result = "SUCCESS"
    @Library('jenkins-library') _
    waitDocker {}

    def tag_name = ''
    stage('scm checkout') {
        checkout(scm)
    }

    withEnv([
            'DOCKER_HOST=unix:///var/run/docker.sock',
            'DOCKER_REPO=172.16.1.99',
            'COMPONENT_NAME=walm',
            'DOCKER_PROD_NS=gold',
    ]) {

        try {
            withCredentials([
                    usernamePassword(
                            credentialsId: 'harbor',
                            passwordVariable: 'DOCKER_PASSWD',
                            usernameVariable: 'DOCKER_USER')
            ]) {
                stage('release build') {
                    sh """#!/bin/bash -ex
                      docker login -u \$DOCKER_USER -p \$DOCKER_PASSWD \$DOCKER_REPO
                      REV=\$(git rev-parse HEAD)
                      export DOCKER_IMG_NAME=\$DOCKER_REPO/\$DOCKER_PROD_NS/\$COMPONENT_NAME:${env.BRANCH_NAME}
                      export DOCKER_IMG_NAME_LATEST=\$DOCKER_REPO/\$DOCKER_PROD_NS/\$COMPONENT_NAME
                      docker build --label CODE_REVISION=\${REV} \
                        --label BRANCH=$env.BRANCH_NAME \
                        --label COMPILE_DATE=\$(date +%Y%m%d-%H%M%S) \
                        -t \$DOCKER_IMG_NAME -f Dockerfile .
                      docker tag \$DOCKER_IMG_NAME \$DOCKER_IMG_NAME_LATEST
                      docker push \$DOCKER_IMG_NAME
                      docker push \$DOCKER_IMG_NAME_LATEST
                    """
                }
            }
        } catch (e) {
            currentBuild.result = "FAILED"
            echo 'Err: Incremental Build failed with Error: ' + e.toString()
            throw e
        } finally {
            sendMail {
                emailRecipients = "tosdev@transwarp.io"
                attachLog = false
            }
        }
    }
}