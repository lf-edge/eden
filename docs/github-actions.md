# Github Actions

Eden is a part of testing infrastructure of EVE and it's integrated in EVE CI/CD pipelines. EVE uses [test.yml](https://github.com/lf-edge/eden/blob/master/.github/workflows/test.yml) reusable workflow to run eden tests against specific EVE version in PR.
Eden reusable workflows are running using [BuildJet](https://buildjet.com/for-github-actions) runners provided by LF-EDGE. They provide both Arm and x86_64 architectures.
Currently we are provided with 4vCPU/16GBs of RAM and 8vCPU/32GBs of RAM runners. Maximum CPUs running in parallel is 64 for x86_64, that means with 4vCPUs we can have 16 jobs running in parallel.
In case one wants to run eden workflows locally in their own fork, `runner` and `repo` input variables for reusable workflow should be specified.

