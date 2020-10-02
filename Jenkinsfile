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
#./eden test ./tests/workflow
./eden setup
./eden start
./eden eve onboard
evebuildname=$(find ../dist/amd64 | grep kvm-amd64)
./eden controller edge-node eveimage-update file://$evebuildname -m adam://
./eden test tests/reboot
./eden test tests/networking
./eden test tests/lim
./eden test tests/docker
'''
      }
    }

  }
}
