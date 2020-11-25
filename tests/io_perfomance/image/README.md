# FIO autotests container

This container runs 36 FIO tests of different types and loads in turn. You can find out more about the test configuration by looking at the config.fio file in the container itself, or the test results in the specified GitHub repository in the directory. Each test is 2 minutes long. The total approximate testing time is ~ 1 hour 12 minutes. Run it and just wait for the result in the specified repository.

## How to

Before starting the container, you need to [get a token on GitHub.com](https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/creating-a-personal-access-token).

### How to deploy

```console
./eden pod deploy --metadata="EVE_VERSION=$(./eden config get --key=eve.tag)\nGIT_REPO=<git repository name>\nGIT_LOGIN=<your git login>\nGIT_TOKEN=<your git token>" -p 8029:80 docker://itmoeve/fio_tests --no-hyper
```

4 required parameters must be specified in the --metadata parameter:

1. **GIT_REPO** - is the name of the repository without .git. For example "eve-performance".
2. **GIT_LOGIN** - username on GitHub, where the repository specified in GIT_REPO is located.
3. **GIT_TOKEN** - GitHub token for authorization and adding a branch with results to your repository.
4. **EVE_VERSION** - EVE version. This parameter is required for naming a branch in GitHub.

### How to run tests

This test creates a virtual machine and starts testing.

```console
GIT_REPO=<git repository name> GIT_LOGIN=<your git login> GIT_TOKEN=<your git token> ./eden test ./tests/io_perfomance
```

>Before running the test, you need to add environmental variables: GIT_REPO, GIT_LOGIN, GIT_TOKEN.

## About results

At the moment, the test results will be posted to the GitHub repository (based on the specified parameters for the environment variables GIT_REPO, GIT_LOGIN, GIT_TOKEN) in a new branch of the specified repository. The new branch will have the following name: "FIO-tests-%date-eve-eve.tag" (Example FIO-tests-11-31-24-11-2020-EVE-0.0.0-st_storage-2dd213ca-new). The directory with the result will be located at the root. The directory name will have the same name as the branch. The results directory has the following structure:

- FIO-tests-%date-eve-version
  - README.md
  - HARDWARE.cfg
  - SUMMARY.csv
  - Configs
    - config.fio
    - Test-results
      - fio-results
      - Iostat
