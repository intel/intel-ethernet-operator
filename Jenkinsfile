# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2021 Intel Corporation
pipeline {
    agent {
        kubernetes {
            label 'go'
        }
    }

    stages {  

        stage('Prerequisites') {
            steps {
                container('go') {
                    dir("$WORKSPACE") {
                        sh '''
                            curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b $(go env GOPATH)/bin
                            curl -fsSL https://deb.nodesource.com/setup_12.x | bash -
                            apt-get install -y yamllint pciutils
                            mkdir /usr/share/hwdata
                            ln -s /usr/share/misc/pci.ids /usr/share/hwdata/pci.ids
                            update-pciids
                        '''
                    }
                }
            }
        }
    
 
        stage('Build') {
            steps {
                container('go') {
                    sh '''
                        echo "BUILD STEP: Build code"
                        rm -rf bin
                        if ! make all; then
                            echo "ERROR: Failed to build the intel-ethernet-operator code"
                            exit 1
                        fi
                    '''
                }
            }
        }
    
        stage('test') {
            steps {
                container('go') {
                    sh '''
                        echo "Run unit tests and gather code coverage"
                        rm -rf testbin
                        if ! make test; then
                            echo "ERROR: Failed to run unit tests"
                            exit 1
                        fi
                        go tool cover -func cover.out
                    '''
                }
            }
        }
    
        stage('code_style'){
            steps{
                container('go') {
                    sh '''
                        DIFFCHECK=$(gofmt)
                        if [ -n "$DIFFCHECK" ]; then
                            echo "WARNING: code style issues found"
                        fi
                    '''
                }
            }
        }
        
        stage('go lint'){
            steps{
                container('go') {
                    sh '''
                        echo "Run Go linter tools"
                        set -x
                        golangci-lint version
                        set +x
                        if ! golangci-lint run -c ${WORKSPACE}/.golangci.yml; then
                            echo "ERROR: Go lint check found errors"
                            exit 1
                        fi
                    '''
                }
            }
        }

        stage('yaml lint'){
            steps{
                container('go') {
                    sh '''
                        echo "Run yaml linter tools"
                        set -x
                        yamllint --version
                        set +x
                        yamllint .
                        if [ $? -ne 0 ]; then
                            exit 1
                        fi
                    '''
                }
            }
        }
    } 
}

