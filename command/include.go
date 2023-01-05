package command

import (
	"fmt"
	"log"
	"text/template"
)

type IncludeCommand struct {
	handler CommandHandler
}

func (i *IncludeCommand) Name() string {
	return "include"
}

func (i *IncludeCommand) GetFuncMap() template.FuncMap {
	return template.FuncMap{
		i.Name(): func(name string) string {
			return fmt.Sprintf("- __%s_%s=%s", i.handler.appName, i.Name(), name)
		},
	}
}

func (i *IncludeCommand) Execute(cmd string, makefile MakeStruct, listType CommandListType) ([]string, error) {
	if _, ok := makefile[cmd]; !ok {
		log.Println(makefile)
		return []string{}, fmt.Errorf("%s not exist, so can not include", cmd)
	}
	var list []string
	switch listType {
	case "script":
		list = makefile[cmd].Script
		break
	case "onFailer":
		list = makefile[cmd].On_Failure
		break
	}
	res := make([]string, 0)
	for _, c := range list {
		t, err := i.handler.commandExecuter(c, makefile, listType)
		if err != nil {
			return []string{}, err
		}
		res = append(res, t...)
	}
	return res, nil
}
