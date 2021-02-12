# FIO autotests container

IO_performance test allows you to run load testing of I/O. The test itself is a docker container. The container is flexible in settings for launching and building results and can accept environment variables described below as input.

## About the accepted environment

Environment parameters that this container can accept.

### For settings for uploading results to GitHub.com

1. **GIT_REPO** - Is the login/name of the repository without .git. For example "itmo-eve/eve-performance", and the result will be as follows: "github.com/itmo-eve/eve-performance".
2. **GIT_LOGIN** - Username on GitHub, where the repository specified in GIT_REPO is located.
3. **GIT_TOKEN** - GitHub token for authorization and adding a branch with results to your repository.
4. **GIT_BRANCH** - Branch name for results pushing. Optional parameter.
5. **GIT_PATH** - Path for placing results in the git repository. The path must already exist in the repository. Optional parameter.
6. **GIT_FOLDER** - Folder name for results on GitHub. Optional parameter.
7. **EVE_VERSION** - EVE version. This parameter is required for naming a folder in GitHub. Optional parameter.
8. **GIT_LOCAL** - Allows you to save test results to the volume passed to the container, via the -v parameter, without publishing the results to GitHub. This parameter can take the value - true. It is an optional parameter.

> If you want to upload results to GitHub.com, the GIT_REPO GIT_LOGIN GIT_TOKEN environment variables will be required. All other environment variables for GitHub are optional.
> Before starting the container, you need to [get a token on GitHub.com](https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/creating-a-personal-access-token).

In this configuration, the test results will be posted to the GitHub repository (based on the specified parameters for the environment variables GIT_REPO, GIT_LOGIN, GIT_TOKEN, and others). The results catalog has the following structure:

- FIO-tests-%date-eve-version
  - README.md
  - SUMMARY.csv
  - Configs
    - config.fio
    - Test-results
      - fio-results
      - Iostat

### To set up FIO

1. **FIO_OPTYPE** - Determining the type of operation. Default=read,write. Optional parameter. For example: read,write,randread
2. **FIO_BS** - Determining the block size. Default=4k,64k,1m. Optional parameter. For example: 4k,1m
3. **FIO_JOBS** - Determining the count jobs in test. Optional parameter. Default=1,8. For example: 1
4. **FIO_DEPTH** - Determining the io depth in test. Optional parameter. Default=1,8,16. For example: 1,8,32
5. **FIO_TIME** - Duration of each test in sec. Default=60. Optional parameter. For example: 60

According to these variables, the container generates a configuration file for the FIO utility. If no variables are set, a configuration file with default settings will be compiled.

## How to run

### How to run in Linux

To run the test without specifying additional parameters, run the following command (In this case, the output of the results will be displayed in the container console):

```console
docker run itmoeve/fio_tests:v.1.2.1
```

To run tests and upload results to GitHub, or configure the FIO utility, you need to start the container by passing the necessary environment variables to it:

```console
sudo docker run -e GIT_REPO='git_repository_name' -e GIT_LOGIN='your_git_login' -e GIT_TOKEN='your_git_token' -e GIT_BRANCH='fio_results' itmoeve/fio_tests:v.1.2.1
```

You can also write all the environment variables you need to a file and pass them to the container via the --env-file option. You can find more information at [docs.docker.com](https://docs.docker.com/engine/reference/commandline/run/#set-environment-variables--e---env---env-file).

If you want to store the test results on the volume passed to the container, you need:

1. Specify the -v parameter, which will take 2 values: the path to a directory or file that will act as a volume in the container, and the mount point in the container itself. The mount point must always be located along the path - **/data**.
2. You also need to pass the GIT_LOCAL environment variable (with the value true), it will save the result.
3. All other environment variables are added optionally.

```console
docker run -v ~/home/fio_results:/data -e GIT_LOCAL=true itmoeve/fio_tests:v.1.2.1
```

After executing this command, the test results will be saved in the directory - ~/home/fio_results

>Over time, the version of the container may be updated

### How to run in Eden

```console
./eden pod deploy --metadata="EVE_VERSION=$(./eden config get --key=eve.tag)\nGIT_REPO=<git repository name>\nGIT_LOGIN=<your git login>\nGIT_TOKEN=<your git token>" -p 8029:80 docker://itmoeve/fio_tests:v.1.2.1
```

> Note that the environment variables were also specified here.

### How to run tests in Eden

This test creates a virtual machine and starts testing.

```console
GIT_REPO=<git repository name> GIT_LOGIN=<your git login> GIT_TOKEN=<your git token> ./eden test ./tests/io_performance
```

>Before running the test, you need to add environmental variables: GIT_REPO, GIT_LOGIN, GIT_TOKEN.
