gomake is a command line tool that allows you to execute bash commands described in a yaml file using gotemplates (Helm Like). 
The yaml file syntax includes a section for variables and a list of commands (see [example](./gomake.yml)), each with a script section for the commands to be executed and an optional on_failure section to specify commands to be executed in case of failure.

# Easy to write easy to read

```yaml
{{$bval := "B"}}
variables: 
  A: {{$bval}}
  FC: {{ .Var.f}}_included
  PUB_KEY: {{shell "cat ~/.ssh/id_rsa.pub"}}
  AB: {{$bval}}B
---
install:
  script: 
    - touch test.gomake.txt
    - echo "Hallo" > test.gomake.txt
    - export TEST1234=ShowME
    - echo $TEST1234
    {{- if eq .Var.A "B" }}
    - echo "LINUX"
    {{- end}}
    - ls {{ .Env.ZDOTDIR}}
    {{include "execute"}}
    - echo {{.Bar}}
    - echo {{.Var.PUB_KEY}}
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

to check how script lookslike after template execute: 

```bash
gomake run --var f=foo --var bar=baz --dry-run install
```

```
NAME:
   gomake - A helm like makefile

USAGE:
   gomake [global options] command [command options] [arguments...]

COMMANDS:
   autocomplete  Set Autocomplete helper stuff to current shell session
   ls            List all commands described at gomake yaml file
   run           Run commands from gomake yml file
   help, h       Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --makefile value, -f value    gomake file to use (default: "./gomake.yml") [$GOMAKE_MAKEFILE]
   --executer value, --sh value  Shell to execute gomakefile config (default: "/bin/sh") [$GOMAKE_EXECUTER]
   --help, -h                    show help (default: false)
```
