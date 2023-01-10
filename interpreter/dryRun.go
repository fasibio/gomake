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

func (i Interpreter) printDryRun(command command.MakeStruct, variables map[string]map[string]any) error {
	out, err := yaml.Marshal(DryRunOutput{
		Vars:            variables["vars"],
		ExecutedCommand: command[i.ExecuteCommand],
	})
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
