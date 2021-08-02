# Processing image

Eden uses processing image for debugging purpose. It contains flamegraph drawer and git tool inside.

```bash
Usage for process perf script into svg:
docker run itmoeve/processing -i file -o file svg - to process output of perf script into svg
Usage for upload to git:
docker run itmoeve/processing -i file -o https://GIT_LOGIN:GIT_TOKEN@GIT_REPO -b branch [-d directory] git
```
