# Debug EVE

You can debug EVE with [perf utility](https://perf.wiki.kernel.org/index.php/Main_Page)
with the following command of EDEN.

To start perf utility inside EVE (`perf record -F 99 -a -g -o /persist/perf.data`):

```bash
eden utils debug start
```

To stop perf utility inside EVE (`killall perf`):

```bash
eden utils debug stop
```

To process perf results into [Flame Graph](http://www.brendangregg.com/flamegraphs.html):

```bash
eden utils debug save flamegraph.svg
```

To obtain `lshw` info and store it into file with name `file_name`:

```bash
eden utils debug hw file_name
```

To upload file into git:

```bash
eden utils gitupload flamegraph.svg <git repo in notation https://GIT_LOGIN:GIT_TOKEN@GIT_REPO> <branch>
```

## Multiple perf on EVE with different parameters

To run multiple perf you must provide different directories to save on EVE with `--perf-location` flag
(default one is `/persist/perf.data`) to commands `eden utils debug start` and `eden utils debug save`.

Also, you can define different options to run perf with `--perf-options` flag (default one is `-F 99 -a -g`).

For example, you can run to save block_rq_insert events in the separate Flame Graph:

```bash
eden utils debug start --perf-location="/persist/perf1.data"
eden utils debug start --perf-options="-F 99 -a -g -e block:block_rq_insert" --perf-location="/persist/perf2.data"
sleep 5
eden utils stop
sleep 5
eden utils debug save flamegraph1.svg --perf-location="/persist/perf1.data"
eden utils debug save flamegraph2.svg --perf-location="/persist/perf2.data"
```

## Debug Go tests in VS Code

To debug Go tests, first ensure you have the debugger installed:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

Next, make sure you have a debug target in your test Makefile, this target compiles the test with debug information into a final binary named `debug`:

```bash
debug:
    CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go test -c -gcflags "all=-N -l" -o $@ *.go
    dlv dap --listen=:12345
```

Finally, ensure you have properly set up your `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Eden Test",
      "type": "go",
      "request": "launch",
      "mode": "exec",
      "program": "${fileDirname}/debug",
      "args": [],
      "showLog": true,
      "port": 12345,
    }
  ]
}
```

At this stage, simply open the test file in VS Code. Open a new terminal in VS Code, navigate to your test directory and run `make debug` :

```bash
$ cd tests/sec
$ make debug
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -c -gcflags "all=-N -l" -o debug *.go
dlv dap --listen=:12345
DAP server listening at: 0.0.0.0:12345
2024-08-28T14:59:52+03:00 warning layer=rpc Listening for remote connections (connections are not authenticated nor encrypted)
```

Finally, put a breakpoint in the part of the code you're interested in and press F5 or go to the menu "Run -> Start Debugging" to start debugging.
