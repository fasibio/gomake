package interpreter

import (
	"fmt"

	"github.com/fasibio/gomake/command"
	"gopkg.in/yaml.v2"
)

type DryRunOutput struct {
	Vars            map[string]any
	ExecutedCommand command.Operation
}

type SDryRunOutput map[string]any

func (i Interpreter) printDryRun(command []StageOperationWrapper, variables map[string]map[string]any) error {

	outPutData := make(SDryRunOutput)
	outPutData["vars"] = variables["vars"]
	for _, c := range command {
		outPutData[c.Name] = c.Command
	}
	out, err := yaml.Marshal(outPutData)
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
