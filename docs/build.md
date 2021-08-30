# Building Eden

To build `eden`:

```console
make build
```

To build `eden` and tests inside eden
It's better to call `eden config add` first, so the build command
can build tests for the desired architecture

```console
make build-tests
```

You can build it for different computer architectures and
operating systems by passing `OS` and `ARCH` options.
The default, however, is for the architecture and OS on
which you are building it.

```console
make build OS=linux
make build ARCH=arm64
```

The generated command is place in `./dist/bin/eden-<arch>-<os>`,
for example `eden-darwin-amd64` or `eden-linux-arm64`.
To ease your life, a symlink is placed in the local directory named
`eden` for your current architecture and OS.
