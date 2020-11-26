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

To upload file into git:

```bash
eden utils gitupload flamegraph.svg <git repo in notation https://GIT_LOGIN:GIT_TOKEN@GIT_REPO> <branch>
```
