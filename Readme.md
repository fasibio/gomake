gomake is a command line tool that allows you to execute bash commands described in a yaml file using gotemplates (Helm Like). 
The yaml file syntax includes a section for variables and a list of commands (see [example](./gomake.yml)), each with a script section for the commands to be executed and an optional on_failure section to specify commands to be executed in case of failure.

```
NAME:
   gomake - A helm like makefile

USAGE:
   gomake [global options] command [command options] [arguments...]

COMMANDS:
   autocomplete  Set Autocomplete helper stuff to current shell session
   init          Crate a starter gomake.yml to current dir
   ls            List all commands described at gomake yaml file
   run           Run commands from gomake.yml file
   srun          Run commands from gomake.yml file  but it run all commands are inside the given stage and run this in parallel
   help, h       Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --makefile value, -f value    gomake file to use (default: "gomake.yml") [$GOMAKE_MAKEFILE]
   --executer value, --sh value  Shell to execute gomakefile config (default: "/bin/sh") [$GOMAKE_EXECUTER]
   --help, -h                    show help (default: false)
```

# Easy to write easy to read
Example Script: 
```yaml
{{$bval := "B"}}
vars: 
  A: {{$bval}}
  FC: {{ .Vars.f}}_included
  PUB_KEY: {{shell "cat ~/.ssh/id_rsa.pub"}}
  AB: {{$bval}}B
---
install:
  doc: I'm a description what this command do
  script: 
    - touch test.gomake.txt
    - echo "Hallo" > test.gomake.txt
    - export TEST1234=ShowME
    - echo $TEST1234
    {{- if eq .Vars.A "B" }}
    - echo "LINUX"
    {{- end}}
    - ls {{ .Env.ZDOTDIR}}
    {{include "execute"}}
    - echo {{.Bar}}
    - echo {{.Vars.PUB_KEY}}
    - lf
  on_failure: 
    - echo "Error"
    {{include "execute"}}
execute: 
  script:
    - echo "execute"
    {{include "extern"}}
  on_failure: 
    - echo "execute Error"
extern: 
  script:
    - echo "extern"
```


To run install commands: 
```bash
gomake run --var f=foo --var bar=baz install
```

to check how script lookslike after template execution: 

```bash
gomake run --var f=foo --var bar=baz --dry-run install
```

# How to Install

## With go

```
go install github.com/fasibio/gomake@latest
```

## Linux (Change version to current)
```bash
sudo wget -O /usr/bin/gomake https://github.com/fasibio/gomake/releases/download/1.0.27/gomake_1.0.27_linux_amd64 ; sudo chmod 755 /usr/bin/gomake
```


# Extra functions

**includeFile**
include all variables and commands from an other lokal file

Possible kind of usage: 
- path ==> example (```{{includeFile "./gomake_helper.yml"}}```)
- wildcard ==> example (```{{includeFile "./gomake_*.yml"}}```)
- url ==> example (```{{includeFile "https://raw.githubusercontent.com/fasibio/gomake/main/gomake_helper.yml"}}```)

**include**
other commands script or onFailure depands of position of command

**shell** 
command execution before the main script is running useful to fill Variables

And all sprig functions ==> [Documentation](http://masterminds.github.io/sprig/)


# Use inside Pipeline
There is a [Dockerimage](https://hub.docker.com/r/fasibio/gomake)

# parallel execution

You can use same stage for different commands

```yaml
buildBin:
  stage: build
  color: "{{$root.Colors.purple}}"
  script: 
    - echo build
buildDocker: 
  stage: build
  color: "{{$root.Colors.red}}"
  script: 
    - docker build -t {{$root.Vars.dockername}}:{{$root.Vars.version}} -f {{$root.Vars.dockerfile}} .
  on_failure: 
    - docker rmi {{$root.Vars.dockername}}:{{$root.Vars.version}}
```
Now you can run by stagename instand of Command name with: 

```
gomake srun build
```

or show the result after template execution: 

```
gomake srun --dry-run build
```

**Hint** 

Color can help you make it easier to read:

Possible Colors ({{.Colors.red}}): 
 - black
 - red
 - green
 - yellow
 - purple
 - magenta
 - teal
 - white



# Use Docker-Images

docker cli required

To execute inside a dockerimage you can use like this: 
```yaml
buildContainer: 
  image: 
    name: golang:latest # required
    volumes: #optional
      - {{.Env.PWD}}:/build
    entrypoint: "" #optional emtpy string is default
    executer: /bin/sh #optional /bin/sh is default
  script: 
    - cd /build
    - ls
    {{include "build"}}
  on_failure: 

```


# Missing
- Windows tests



