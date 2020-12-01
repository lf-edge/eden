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
