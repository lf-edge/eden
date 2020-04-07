# Eden

Eden is where [EVE](https://github.com/lf-edge/eve) and [Adam](https://github.com/lf-edge/adam) get tried and tested.

Eden consists of a test harness and a series of integration tests implemented in Golang. Tests are structured as normal Golang tests by using ```_test.go``` nomenclature and be available for test runs using standard go test framework.

## Install Dependencies

Install requirements from [eve](https://github.com/lf-edge/eve#install-dependencies)

Also, you need to install ```uuidgen```

## Running

Recommended to run from superuser

To run harness use: ```make run```

To run tests use: ```make test```

To stop harness use: ```make stop```

## Help

You can see help by running ```make help```

## Utilites

Eden is controlled by a single command named (in a secret code) `eden`. It has multple sub-commands and options.
Run `eden help` to see sub-commands.

To build `eden`:

```
make build
```

You can build it for different computer architectures and operating systems by passing `OS` and `ARCH` options.
The default, however, is for the architecture and OS on which you are building it.

```
make build OS=linux
make build ARCH=arm64
```

The generated command is place in `./dist/bin/eden-<arch>-<os>`, for example `eden-darwin-amd64` or `eden-linux-arm64`.
To ease your life, a symlink is placed in the local directory named `eden` for your current architecture and OS.

The current sub-commands are:

   * `certs` -- SSL certificate generator;
   * `info` -- scans Info file accordingly by regular expression of requests to json fields;
   * `infowatch` -- Info-files monitoring tool with regular expression quering to json fields;
   * `log` -- scans Log file accordingly by regular expression of requests to json fields;
   * `logwatch` -- Log-files monitoring tool with regular expression quering to json fields;
   * `server` -- micro HTTP-server for providing of baseOS and Apps images.
