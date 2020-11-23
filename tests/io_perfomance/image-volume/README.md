# FIO autotests container

This container runs 36 FIO tests of different types and loads in turn. You can find out more about the test configuration by looking at the config.fio file in the container itself, or the test results in the specified GitHub repository in the directory. Each test is 2 minutes long. The total approximate testing time is ~ 1 hour 12 minutes. Run it and just wait for the result in the specified repository.

## How to

Before starting the container, you need to [get a token on GitHub.com](https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/creating-a-personal-access-token).

### To deploy

```console
./eden pod deploy --metadata="EVE_VERSION=5.14\nGIT_REPO=eve-performance\nGIT_LOGIN=itmo-eve\nGIT_TOKEN=1111111111111111111111111aaaaaaaaaaaaaaa" -p 8029:80 docker://itmoeve/fio_tests --no-hyper
```

> If you want to start a container with the **VOLUME** parameter, you need to specify the tag **volume** in the container. For example: docker://itmoeve/fio_tests:volume

3 required parameters must be specified in the --metadata parameter:

1. **GIT_REPO** - is the name of the repository without .git. For example "eve-performance".
2. **GIT_LOGIN** - username on GitHub, where the repository specified in GIT_REPO is located.
    > **Attention!** if your login contains **_**, in this case, it is necessary to specify instead of **_** **-**
3. **GIT_TOKEN** - GitHub token for authorization and adding a branch with results to your repository.
4. **EVE_VERSION** - EVE version. This parameter is required for naming a branch in GitHub.

### How to run tests

This test creates a virtual machine and starts testing.

```console
./eden test ./tests/io_perfomance
```

>Before running the test, you need to add environmental variables: GIT_REPO, GIT_LOGIN, GIT_TOKEN.

## About results

At the moment, the test results will be posted to the GitHub repository in a new branch and will have the following tree:

- FIO-tests-%date
  - README.md
  - HARDWARE.cfg
  - SUMMARY.csv
  - Configs
    - config.fio
    - Test-results
      - fio-results
      - Iostat
