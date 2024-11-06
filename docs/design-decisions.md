# Design decisions

This document is a collection of high-level decisions, that have been made in the project. Why do we have a section about how things use to be? In case if we want to revisit certain concept it will be useful to know why we chose one approach over another.

## Using Domain Specific Language (DSL) vs writing native golang tests

When talking about Eden tests till version 0.9.12, we are talking about `escript`, a Domain Specific Language (DSL) which describes test case and uses Eden to setup, environment. Escript looks like this

```bash
# 1 Setup environment variables
{{$port := "2223"}}
{{$network_name := "n1"}}
{{$app_name := "eclient"}}

# ...

# 2 run eden commands
eden -t 1m network create 10.11.12.0/24 -n {{$network_name}}

# 3 run escript commands
test eden.network.test -test.v -timewait 10m ACTIVATED {{$network_name}}

eden pod deploy -n {{$app_name}} --memory=512MB {{template "eclient_image"}} -p {{$port}}:22 --networks={{$network_name}}

# 4 execute shell script which are defined inside escript file
exec -t 5m bash ssh.sh
stdout 'Ubuntu'

# 5 overwrite configuration
-- eden-config.yml --
{{/* Test's config. file */}}
test:
    controller: adam://{{EdenConfig "adam.ip"}}:{{EdenConfig "adam.port"}}
    eve:
      {{EdenConfig "eve.name"}}:
        onboard-cert: {{EdenConfigPath "eve.cert"}}
        serial: "{{EdenConfig "eve.serial"}}"
        model: {{EdenConfig "eve.devmodel"}}

-- ssh.sh --
EDEN={{EdenConfig "eden.root"}}/{{EdenConfig "eden.bin-dist"}}/{{EdenConfig "eden.eden-bin"}}
for i in `seq 20`
do
  sleep 20
  # Test SSH-access to container
  echo $i\) $EDEN sdn fwd eth0 {{$port}} -- {{template "ssh"}} grep Ubuntu /etc/issue
  $EDEN sdn fwd eth0 {{$port}} -- {{template "ssh"}} grep Ubuntu /etc/issue && break
done
```

So you can

1) setup some environment variables
2) run eden commands
3) run escript commands with test
4) execute user-defined shell scripts
5) overwrite eden configuration

Escript file is fed as input to eden test command, which parses the variables using golang templates to substitute some variables using templates in golang, then it's going to read line by line and execute it via os.exec call in golang. test is actually a compiled golang binary, we compile it inside eden repo and then execute it, under the hood it is a manually forked version of standard golang test package with some added commands.

For more information on specific topics refer to following documents:

- Escript test structure [doc](./escript/test-anatomy-sample.md)
- Writing eden tasks for escript [doc](./escript/task-writing.md)
- Running tests [doc](./escript/test-running.md)

So there are several problems with that approach:

**You have to be an escript expert**: when new person comes to a project, they will have some industry skills, like C++ expertise, Rust, Golang,  Computer Networking, etc. but if you never worked on this project, you never heard about escript, its' sole purpose was to be part of Eden and be useful language to describe tests for Eden. One might argue, but that's just bash on steroids. Well, true, but do you know what `!` means? It's not equal parameter in escript. So it is like bash, but not exactly it, and it might take time to figure out other hidden features, that could be solved by proper documentation. But before spiraling down that conversation lets think about tools? Imagine, that you wrote new fancy eden test and it doesn't work. Usual thing, you would say. Test Driven Development is all about  writing failing tests first and then making them work.

**But how do you debug that test?** Because you're running an interpreter, which runs bash commands via os.exec plus you execute golang program, which is written and compiled inside the repository. Debug printing would work, but I find debuggers much more useful and efficient most of the times. It is a matter of taste, but better to debugger at your disposal, than not to have it in this case. Even worse, how do you debug escript problem? You need to know it's internals, there's no Stackoverflow or Google by your side, nor there are books about it.

**Bumping golang is actually manual action**: last but not least, we have to maintain custom test files which a basically copy-paste with one added function, bumping golang turns into some weird dances in that case.
