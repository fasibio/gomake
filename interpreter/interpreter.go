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
	Vars map[string]any
	Env  map[string]string
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
	explizitMakeFile, _, err := r.GetExecuteTemplate(string(r.commandFile), make(map[string]any))
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

func (r *Interpreter) GetExecuteTemplate(file string, extraVariables map[string]any) ([]byte, map[string]map[string]any, error) {
	varCommandArr := strings.Split(file, "---")
	if len(varCommandArr) == 1 {
		varCommandArr = strings.Split("vars:\n---\n"+file, "---")
	}
	if len(varCommandArr) != 2 {
		return []byte{}, nil, fmt.Errorf("only variables and command as seperated yaml are allowed")
	}

	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		env[pair[0]] = pair[1]
	}
	tempVar := make(map[string]any)
	for k, v := range r.ExtraVariables {
		tempVar[k] = v
	}

	for k, v := range extraVariables {
		if _, ok := tempVar[k]; !ok {
			tempVar[k] = v
		}
	}

	varStr, err := r.getParsedTemplate("gomake_vars", varCommandArr[0], TemplateData{Env: env, Vars: tempVar})

	if err != nil {
		return nil, nil, err
	}
	var variables map[string]map[string]any
	err = yaml.Unmarshal(varStr, &variables)
	if err != nil {
		return []byte{}, nil, err
	}
	for k, v := range r.ExtraVariables {
		variables["vars"][k] = v
	}

	for k, v := range extraVariables {
		if _, ok := variables["vars"][k]; !ok {
			variables["vars"][k] = v
		}
	}

	v, err := r.cmdHandler.ExecuteVariablesCommands(variables["vars"])
	if err != nil {
		return nil, nil, err
	}

	b, err := r.getParsedTemplate("gomake", varCommandArr[1], TemplateData{Vars: v, Env: env})
	return b, variables, err
}

func (r *Interpreter) Run() error {
	explizitMakeFile, variables, err := r.GetExecuteTemplate(string(r.commandFile), make(map[string]any))
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
	if err != nil {
		return err
	}
	if r.DryRun {
		return r.printDryRun(command, variables)
	}

	cmd := r.cmdHandler.SliceCommands(command[r.ExecuteCommand].Script)
	executer := r.executer
	if image := command[r.ExecuteCommand].Image; image != nil {
		cmd = getDockerCmd(cmd, image)
		log.Println(cmd)
	}

	err = r.execCmd(executer, cmd)
	if err != nil {
		if len(command[r.ExecuteCommand].On_Failure) > 0 {
			fmt.Println("Script end with error so start onFailure Scripts ...")
			err = r.execCmd(executer, r.cmdHandler.SliceCommands(command[r.ExecuteCommand].On_Failure))
		} else {
			fmt.Println("No onFailure Scripts found but got error")
		}
	}
	return err
}

func (r *Interpreter) execCmd(executer, command string) error {
	cmd := exec.Command(executer, "-c", command)
	cmd.Stdin = os.Stdin
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
		b, variables, err := r.GetExecuteTemplate(string(f), data.Vars)
		if err != nil {
			log.Panic(err)
		}
		for k, v := range variables["vars"] {
			data.Vars[k] = v
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
