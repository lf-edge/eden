# FIO autotests container

This container runs 36 FIO tests of different types and loads in turn. You can find out more about the test configuration by looking at the config.fio file in the container itself, or the test results in the specified GitHub repository in the directory. Each test is 2 minutes long. The total approximate testing time is ~ 1 hour 12 minutes. Run it and just wait for the result in the specified repository.

## How to

Before starting the container, you need to [get a token on GitHub.com](https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/creating-a-personal-access-token).

### To deploy

```console
./eden pod deploy --metadata="GITREPO=eve-performance\nLOGIN=itmo-eve\nTOKEN=1111111111111111111111111aaaaaaaaaaaaaaa" -p 8029:80 docker://vken/fio_tests --no-hyper
```

3 required parameters must be specified in the --metadata parameter:

1. **GITREPO** - is the name of the repository without .git. For example "eve-performance".
2. **LOGIN** - username on GitHub, where the repository specified in GITREPO is located
    > **Attention!** if your login contains **_**, in this case, it is necessary to specify instead of _ **-**
3. **TOKEN** - GitHub token for authorization and adding a branch with results to your repository

## About results

At the moment, the test results will be posted to the GitHub repository in a new branch and will have the following tree:

- FIO-tests-%date
  - README.md
  - HARDWARE.cfg
  - SUMMARY.csv
  - configs
    - config.fio
    - tets-result
      - fio-result
