package interpreter

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/fasibio/gomake/command"
	nearfinder "github.com/fasibio/gomake/nearFinder"
	"gopkg.in/yaml.v2"
)

type TemplateData struct {
	Var map[string]string
	Env map[string]string
}

func (t TemplateData) Bar() string {
	return "Bar"
}

type Interpreter struct {
	App            string
	cmdHandler     command.CommandHandler
	commandFile    []byte
	DryRun         bool
	ExecuteCommand string
	executer       string
	ExtraVariables map[string]string
}

func NewInterpreter(appName, executeCommand, executer string, dryRun bool, cmdHandler command.CommandHandler, commandFile []byte) Interpreter {
	return Interpreter{
		App:            appName,
		cmdHandler:     cmdHandler,
		commandFile:    commandFile,
		DryRun:         dryRun,
		ExecuteCommand: executeCommand,
		executer:       executer,
		ExtraVariables: make(map[string]string),
	}
}

func (r *Interpreter) GetMakeScripts() (command.MakeStruct, error) {
	explizitMakeFile, _, err := r.GetExecuteTemplate(string(r.commandFile), make(map[string]string))
	if err != nil {
		return nil, err
	}
	return r.getMakeScripts(explizitMakeFile)
}
func (r *Interpreter) getMakeScripts(yamlFileData []byte) (command.MakeStruct, error) {
	var c1 command.MakeStruct
	err := yaml.Unmarshal(yamlFileData, &c1)
	return c1, err
}

func (r *Interpreter) GetExecuteTemplate(file string, extraVariables map[string]string) ([]byte, map[string]map[string]string, error) {
	varCommandArr := strings.Split(file, "---")
	if len(varCommandArr) == 1 {
		varCommandArr = strings.Split("variables:\n---\n"+file, "---")
	}
	if len(varCommandArr) != 2 {
		return []byte{}, nil, fmt.Errorf("only variables and command as seperated yaml are allowed")
	}

	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		env[pair[0]] = pair[1]
	}
	tempVar := make(map[string]string)
	for k, v := range r.ExtraVariables {
		tempVar[k] = v
	}

	for k, v := range extraVariables {
		if _, ok := tempVar[k]; !ok {
			tempVar[k] = v
		}
	}

	varStr, err := r.getParsedTemplate("gomake_variables", varCommandArr[0], TemplateData{Env: env, Var: tempVar})

	if err != nil {
		return nil, nil, err
	}
	var variables map[string]map[string]string
	err = yaml.Unmarshal(varStr, &variables)
	if err != nil {
		return []byte{}, nil, err
	}
	for k, v := range r.ExtraVariables {
		variables["variables"][k] = v
	}

	for k, v := range extraVariables {
		if _, ok := variables["variables"][k]; !ok {
			variables["variables"][k] = v
		}
	}

	v, err := r.cmdHandler.ExecuteVariablesCommands(variables["variables"])
	if err != nil {
		return nil, nil, err
	}

	b, err := r.getParsedTemplate("gomake", varCommandArr[1], TemplateData{Var: v, Env: env})
	return b, variables, err
}

type DryRunOutput struct {
	Variables       map[string]string
	ExecutedCommand command.Operation
}

func (r *Interpreter) Run() error {
	explizitMakeFile, variables, err := r.GetExecuteTemplate(string(r.commandFile), make(map[string]string))
	if err != nil {
		return err
	}
	c1, err := r.getMakeScripts(explizitMakeFile)
	if err != nil {
		return err
	}

	if _, ok := c1[r.ExecuteCommand]; !ok {
		return fmt.Errorf("command %s not exist at makefile, did you mean \n%s", r.ExecuteCommand, nearfinder.ClosestMatch(r.ExecuteCommand, nearfinder.GetKeysOfMap(c1), 2))
	}

	command, err := r.cmdHandler.GetExecutedCommandMakeScript(r.ExecuteCommand, c1)
	if r.DryRun {
		if err != nil {
			return err
		}
		out, err := yaml.Marshal(DryRunOutput{
			Variables:       variables["variables"],
			ExecutedCommand: command[r.ExecuteCommand],
		})
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	err = r.execCmd(r.cmdHandler.SliceCommands(command[r.ExecuteCommand].Script))
	if err != nil {
		if len(command[r.ExecuteCommand].On_Failure) > 0 {
			fmt.Println("Script end with error so start onFailure Scripts ...")
			err = r.execCmd(r.cmdHandler.SliceCommands(command[r.ExecuteCommand].On_Failure))
		} else {
			fmt.Println("No onFailure Scripts found but got error")
		}
	}
	return err
}

func (r *Interpreter) execCmd(command string) error {
	cmd := exec.Command(r.executer, "-c", command)
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	return cmd.Run()
}

func (r *Interpreter) getParsedTemplate(templateName, tmpl string, data TemplateData) ([]byte, error) {
	t := template.New(templateName)
	var buf bytes.Buffer

	funcMap := r.cmdHandler.GetFuncMap()
	funcMap["includeFile"] = func(name string) string {
		f, err := os.ReadFile(name)
		if err != nil {
			log.Panic(err)
		}
		b, variables, err := r.GetExecuteTemplate(string(f), data.Var)
		if err != nil {
			log.Panic(err)
		}
		for k, v := range variables["variables"] {
			data.Var[k] = v
		}
		return string(b)
	}
	sprigFunc := sprig.FuncMap()
	for k, v := range sprigFunc {
		funcMap[k] = v
	}

	t, err := t.Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return []byte{}, err
	}
	t.Execute(&buf, data)
	return buf.Bytes(), nil
}
