package command

import (
	"fmt"
	"log"
	"strings"
	"text/template"
)

type MakeStruct map[string]Operation

type Operation struct {
	Script     []string
	Image      *DockerOperation
	On_Failure []string
	Stage      string
	Color      string
}

type DockerOperation struct {
	Name string
	//default is /bin/sh
	Executer string
	Volumes  []string
	// default is nil
	Entrypoint string
}

type CommandListType string

const (
	CommandListTypeScript   CommandListType = "script"
	CommandListTypeOnFailer CommandListType = "onFailer"
)

type Command interface {
	Name() string
	Execute(cmd string, makefile MakeStruct, listType CommandListType) ([]string, error)
	GetFuncMap() template.FuncMap
}

type CommandHandler struct {
	appName string
	handler map[string]Command
}

func NewCommandHandler(appName string) CommandHandler {
	res := CommandHandler{appName: appName, handler: make(map[string]Command)}
	res.registerStandardHandler()
	return res
}
func (c *CommandHandler) registerStandardHandler() {
	c.RegisterHandler(&IncludeCommand{
		handler: *c,
	})
}

func (c *CommandHandler) ExecuteVariablesCommands(variabels map[string]any) (map[string]any, error) {
	res := variabels
	for k, v := range variabels {
		prefix := fmt.Sprintf("__%s_", c.appName)
		vstr, ok := v.(string)
		if !ok {
			continue
		}
		if strings.HasPrefix(vstr, prefix) {
			command := strings.SplitN(strings.TrimLeft(vstr, prefix), "=", 2)
			if len(command) > 2 {
				log.Println("Problem wtf")
			}
			for k1, v := range c.handler {

				if k1 == command[0] {
					tmp, err := v.Execute(command[1], nil, "")
					if err != nil {
						return nil, err
					}
					res[k] = tmp[0]
				}
			}
		} else {
			res[k] = v

		}
	}

	return res, nil

}

func (c *CommandHandler) RegisterHandler(cmd Command) error {
	if _, ok := c.handler[cmd.Name()]; ok {
		return fmt.Errorf("%s allready exist as Handler", cmd.Name())
	}
	c.handler[cmd.Name()] = cmd
	return nil
}

func (c *CommandHandler) GetExecutedCommandMakeScript(cmd string, data MakeStruct) (MakeStruct, error) {

	commands := make([]string, 0)

	for _, commandLine := range data[cmd].Script {
		t, err := c.commandExecuter(commandLine, data, CommandListTypeScript)
		if err != nil {
			return nil, err
		}
		commands = append(commands, t...)
	}

	onFailer := make([]string, 0)
	for _, commandLine := range data[cmd].On_Failure {
		t, err := c.commandExecuter(commandLine, data, CommandListTypeOnFailer)
		if err != nil {
			return nil, err
		}
		onFailer = append(onFailer, t...)
	}
	res := make(MakeStruct)
	res[cmd] = Operation{
		Script:     commands,
		On_Failure: onFailer,
		Image:      data[cmd].Image,
		Color:      data[cmd].Color,
		Stage:      data[cmd].Stage,
	}
	return res, nil
}

func (c *CommandHandler) GetFuncMap() template.FuncMap {
	res := make(template.FuncMap)
	for _, v := range c.handler {
		tmpMap := v.GetFuncMap()
		for fmk, fmv := range tmpMap {
			res[fmk] = fmv
		}
	}
	return res
}

func (c *CommandHandler) commandExecuter(cmd string, data MakeStruct, listType CommandListType) ([]string, error) {
	prefix := fmt.Sprintf("__%s_", c.appName)
	if strings.HasPrefix(cmd, prefix) {
		command := strings.SplitN(strings.TrimLeft(cmd, prefix), "=", 2)
		if len(command) > 2 {
			log.Println("Problem wtf")
		}
		for k, v := range c.handler {

			if k == command[0] {
				return v.Execute(command[1], data, listType)
			}
		}
	}
	return []string{cmd}, nil
}

func (c *CommandHandler) SliceCommands(cmdList []string) string {
	res := ""
	for _, c := range cmdList {
		res += fmt.Sprintf("echo \"\\$ %s\";%s; ", c, c)
	}
	return res
}
