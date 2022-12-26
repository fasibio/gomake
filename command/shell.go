package command

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"text/template"
)

type ShellCommand struct {
	handler CommandHandler
}

func (i *ShellCommand) Name() string {
	return "shell"
}

func (i *ShellCommand) GetFuncMap() template.FuncMap {
	return template.FuncMap{
		i.Name(): func(name string) string {
			return fmt.Sprintf("__%s_%s=%s", i.handler.appName, i.Name(), name)
		},
	}
}

func (i *ShellCommand) Execute(cmd string, makefile MakeStruct, listType CommandListType) ([]string, error) {
	cmdRunner := exec.Command("/bin/sh", "-c", cmd)
	buf := new(bytes.Buffer)
	cmdRunner.Stdout = buf
	cmdRunner.Stderr = log.Writer()

	err := cmdRunner.Run()
	return []string{buf.String()}, err
}
