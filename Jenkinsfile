pipeline {
  agent any
  stages {
    stage('') {
      steps {
        sh '''#!/bin/bash
make clean
result=""
q=1
while [ "$result" == "" ]
do
        result=$(curl https://registry.hub.docker.com/v2/repositories/lfedge/eve/tags?page=$q | jq -r \'."results"[]["name"]\' | grep kvm-amd64 | head -1)
        q=$((q + 1))
done
result2=${result%-kvm-amd64}
echo $result2 
make build
./eden config add default
./eden config set default --key=eve.accel --value=false
./eden config set default --key=eve.tag --value=$result2
./eden test ./tests/workflow'''
      }
    }

  }
}