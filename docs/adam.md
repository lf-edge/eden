# Adam

[adam](http://github.com/lf-edge/adam) is the reference open-source
EVE controller. adam can be run as a binary standalone or a docker
container. Rather than running it alone, eden coordinates it all for you.

adam itself stores all of its configuration as well as data that it
receives from eve devices in a backend. By default the backend
is just a local directory, but it is pluggable.

In the case of eden, Adam is run configured to use redis as a backend,
so that data can be shared and managed. This does mean, however,
that if you are looking for information, you _must_ use one of:

* `eden` command
* `adam admin` command
* `redis-cli` command

To use `adam admin`:

```sh
docker exec eden_adam adam admin <command>
```

To use `redis-cli` (with password from file `~/.eden/certs/redis.pass`):

```sh
docker exec -it eden_redis redis-cli -a $(cat ~/.eden/certs/redis.pass)
```

For the redis keys, you can run `keys *` inside the redis CLI.
Note that the `INFO_EVE_*`, `LOGS_EVE_*`, `METRICS_EVE_*` and `FLOW_MESSAGE_EVE_*` keys are of type `stream`,
and thus you need to use `xread stream`, `xinfo stream`
and their family of commands to read them.

It may be much easier to just use `adam admin` or `eden info`/`eden logs`/`eden metric`/`eden netstat`.
