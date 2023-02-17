package interpreter

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/fasibio/gomake/command"
	nearfinder "github.com/fasibio/gomake/nearFinder"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var colorsMap map[string]string = map[string]string{
	"black":   "\033[1;30m%s\033[0m",
	"red":     "\033[1;31m%s\033[0m",
	"green":   "\033[1;32m%s\033[0m",
	"yellow":  "\033[1;33m%s\033[0m",
	"purple":  "\033[1;34m%s\033[0m",
	"magenta": "\033[1;35m%s\033[0m",
	"teal":    "\033[1;36m%s\033[0m",
	"white":   "\033[1;37m%s\033[0m",
}

func getColorKeyMap() map[string]string {
	res := make(map[string]string)
	for k := range colorsMap {
		res[k] = k
	}
	return res
}

type TemplateData struct {
	Vars   map[string]any
	Env    map[string]string
	Colors map[string]string
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

	varStr, err := r.getParsedTemplate("gomake_vars", varCommandArr[0], TemplateData{Env: env, Vars: tempVar, Colors: getColorKeyMap()})

	log.Println(string(varStr))
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

	if len(variables["vars"]) == 0 {
		variables["vars"] = make(map[string]any)
	}

	v, err := r.cmdHandler.ExecuteVariablesCommands(variables["vars"])
	if err != nil {
		return nil, nil, err
	}

	b, err := r.getParsedTemplate("gomake", varCommandArr[1], TemplateData{Vars: v, Env: env, Colors: getColorKeyMap()})
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
		return r.printDryRun([]StageOperationWrapper{{Name: r.ExecuteCommand, Command: command[r.ExecuteCommand]}}, variables)
	}

	cmd := r.cmdHandler.SliceCommands(command[r.ExecuteCommand].Script)
	executer := r.executer
	if image := command[r.ExecuteCommand].Image; image != nil {
		cmd = getDockerCmd(cmd, image)
	}

	err = r.execCmd(executer, cmd, log.Writer())
	if err != nil {
		if len(command[r.ExecuteCommand].On_Failure) > 0 {
			fmt.Println("Script end with error so start onFailure Scripts ...")
			err = r.execCmd(executer, r.cmdHandler.SliceCommands(command[r.ExecuteCommand].On_Failure), log.Writer())
		} else {
			fmt.Println("No onFailure Scripts found but got error")
		}
	}
	return err
}

type StageOperationWrapper struct {
	Name    string
	Command command.Operation
}

type StageOperationWrapperError struct {
	StageOperationWrapper
	error
}

func (w *StageOperationWrapper) Write(data []byte) (n int, err error) {

	colorFunc := func(d ...interface{}) string { return fmt.Sprint(d...) }
	if w.Command.Color != "" {
		colorFunc = w.Color(colorsMap[w.Command.Color])

	}
	fmt.Print(colorFunc(w.Name + ":\t" + string(data)))
	// fmt.Printf("%s:\t%s", w.ServiceName, string(data))
	return len(data), nil
}

func (w *StageOperationWrapper) Color(colorString string) func(...interface{}) string {
	sprint := func(args ...interface{}) string {
		return fmt.Sprintf(colorString,
			fmt.Sprint(args...))
	}
	return sprint
}

func (r *Interpreter) GetStageMap() (map[string][]StageOperationWrapper, command.MakeStruct, map[string]map[string]any, error) {
	explizitMakeFile, variables, err := r.GetExecuteTemplate(string(r.commandFile), make(map[string]any))
	if err != nil {
		return nil, nil, nil, err
	}
	c1, err := r.getMakeScripts(explizitMakeFile)
	if err != nil {
		return nil, nil, nil, err
	}
	stagesMap := make(map[string][]StageOperationWrapper)
	for k, c := range c1 {
		if c.Stage != "" {
			_, ok := stagesMap[c.Stage]
			if !ok {
				stagesMap[c.Stage] = make([]StageOperationWrapper, 0)
			}
			stagesMap[c.Stage] = append(stagesMap[c.Stage], StageOperationWrapper{Name: k, Command: c})
		}
	}
	return stagesMap, c1, variables, nil
}

// Stage running
func (r *Interpreter) SRun() error {
	stagesMap, c1, variables, err := r.GetStageMap()
	if err != nil {
		return err
	}

	if _, ok := stagesMap[r.ExecuteCommand]; !ok {
		return fmt.Errorf("no command with stage %s found at makefile, did you mean \n%s", r.ExecuteCommand, nearfinder.ClosestMatch(r.ExecuteCommand, nearfinder.GetKeysOfMap(stagesMap), 2))
	}

	commands := make([]StageOperationWrapper, 0)
	for _, c := range stagesMap[r.ExecuteCommand] {
		tmpc, err := r.cmdHandler.GetExecutedCommandMakeScript(c.Name, c1)
		if err != nil {
			return err
		}
		commands = append(commands, StageOperationWrapper{Name: c.Name, Command: tmpc[c.Name]})
	}

	if r.DryRun {
		return r.printDryRun(commands, variables)
	}
	w := sync.WaitGroup{}
	errList := make([]StageOperationWrapperError, 0)
	for _, c := range commands {
		w.Add(1)
		go func(operator StageOperationWrapper) {
			cmd := r.cmdHandler.SliceCommands(operator.Command.Script)
			if image := operator.Command.Image; image != nil {
				cmd = getDockerCmd(cmd, image)
			}
			err := r.execCmd(r.executer, cmd, &operator)
			if err != nil {
				errList = append(errList, StageOperationWrapperError{
					error:                 err,
					StageOperationWrapper: operator,
				})
			}
			w.Done()
		}(c)
	}
	w.Wait()
	var errres error = nil
	if len(errList) > 0 {
		for _, e := range errList {
			if len(e.Command.On_Failure) > 0 {
				e.Write([]byte(fmt.Sprintf("%s end with error so start onFailure Scripts ...", e.Name)))
				err := r.execCmd(r.executer, r.cmdHandler.SliceCommands(e.Command.On_Failure), &e)
				if err != nil {
					errres = errors.Wrap(errres, err.Error())
				}
			} else {
				e.Write([]byte(fmt.Sprintf("No OnFailer Script found but %s got error ", e.Name)))
			}
		}
	}
	return errres
}

func (r *Interpreter) execCmd(executer, command string, writer io.Writer) error {
	cmd := exec.Command(executer, "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = writer
	cmd.Stderr = writer
	return cmd.Run()
}

func (r *Interpreter) getParsedTemplate(templateName, tmpl string, data TemplateData) ([]byte, error) {
	t := template.New(templateName)
	var buf bytes.Buffer

	funcMap := r.cmdHandler.GetFuncMap()
	funcMap["shell"] = func(cmd string) string {
		cmdRunner := exec.Command("/bin/sh", "-c", cmd)
		buf := new(bytes.Buffer)
		cmdRunner.Stdout = buf
		cmdRunner.Stderr = log.Writer()

		err := cmdRunner.Run()
		if err != nil {
			log.Panic(err)
		}
		return buf.String()
	}
	funcMap["includeFile"] = func(name string) string {
		files, err := GetContents(name)
		if err != nil {
			log.Panic(err)
		}
		res := strings.Builder{}

		for _, f := range files {
			b, variables, err := r.GetExecuteTemplate(string(f), data.Vars)
			if err != nil {
				log.Panic(err)
			}
			for k, v := range variables["vars"] {
				data.Vars[k] = v
			}
			res.Write(b)
			res.WriteString("\n")
		}
		return res.String()
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

func GetContents(path string) ([]string, error) {
	var contents []string
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		resp, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		contents = append(contents, string(bytes))
	} else if strings.Contains(path, "*") {
		files, err := filepath.Glob(path)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			bytes, err := ioutil.ReadFile(file)
			if err != nil {
				return nil, err
			}
			contents = append(contents, string(bytes))
		}
	} else {
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		contents = append(contents, string(bytes))
	}
	return contents, nil
}
