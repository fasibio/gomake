package main

import (
	"embed"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/fasibio/gomake/command"
	"github.com/fasibio/gomake/interpreter"
	nearfinder "github.com/fasibio/gomake/nearFinder"
	"github.com/urfave/cli/v2"
)

const (
	MakeFileCli            = "makefile"
	CommandCli             = "command"
	ExecuterCli            = "executer"
	App                    = "GOMAKE"
	DryRunCli              = "dry-run"
	VarsCli                = "var"
	ShellAutocompleteCli   = "shell"
	PersistAutocompleteCli = "persist"
)

const (
	GomakeDefaultFile = "gomake.yml"
)

func getFlagEnvByFlagName(flagName string) string {
	return fmt.Sprintf("%s_%s", App, strings.ToUpper(flagName))
}

var VariableFlagRegex *regexp.Regexp

func init() {
	VariableFlagRegex = regexp.MustCompile(`^[0-9aA-zZ]*=[0-9,aA-zZ]*$`)

}

//go:embed autocomplete/*
var autocompleteFiles embed.FS

//go:embed gomake_ini.yml
var gomakeIni []byte

func main() {
	runner := Runner{
		cmdHandler: command.NewCommandHandler(App),
	}

	app1 := &cli.App{
		Usage:                "A helm like makefile",
		EnableBashCompletion: true,
		CommandNotFound:      runner.CommandNotFound,
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:    MakeFileCli,
				Aliases: []string{"f"},
				EnvVars: []string{getFlagEnvByFlagName(MakeFileCli)},
				Value:   GomakeDefaultFile,
				Usage:   "gomake file to use",
			},
			&cli.StringFlag{
				Name:    ExecuterCli,
				Aliases: []string{"sh"},
				EnvVars: []string{getFlagEnvByFlagName(ExecuterCli)},
				Value:   "/bin/sh",
				Usage:   "Shell to execute gomakefile config",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "autocomplete",
				Usage:  "Set Autocomplete helper stuff to current shell session",
				Action: runner.Autocomplete,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    ShellAutocompleteCli,
						EnvVars: []string{"SHELL"},
						Usage:   "Shell inside this terminal",
					},
					&cli.BoolFlag{
						Name:    PersistAutocompleteCli,
						Aliases: []string{"p"},
						EnvVars: []string{getFlagEnvByFlagName(PersistAutocompleteCli)},
						Usage:   "To make autocomplete for this programm persist (Linux only)",
					},
				},
			},
			{
				Name:   "init",
				Usage:  fmt.Sprintf("Crate a starter %s to current dir", GomakeDefaultFile),
				Action: runner.Init,
			},
			{
				Name:   "ls",
				Usage:  "List all commands described at gomake yaml file",
				Action: runner.List,
				Before: runner.Before,
			},
			{
				ArgsUsage:    "{executed command name}",
				Name:         "run",
				Usage:        fmt.Sprintf("Run commands from %s file", GomakeDefaultFile),
				BashComplete: runner.RunBashComplete,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    DryRunCli,
						EnvVars: []string{getFlagEnvByFlagName(DryRunCli)},
						Value:   false,
						Usage:   fmt.Sprintf("Only show template paresed %s but not execute it", GomakeDefaultFile),
					},
					&cli.StringSliceFlag{
						Name:    VarsCli,
						Aliases: []string{"v"},
						EnvVars: []string{getFlagEnvByFlagName(VarsCli)},
						Action:  runner.ExtraVariables,
					},
				},
				Action: runner.Run,
				Before: runner.RunBefore,
			},
			{
				ArgsUsage:    "{executed command name}",
				Name:         "srun",
				Usage:        fmt.Sprintf("Run commands from %s file  but it run all commands are inside the given stage and run this in parallel", GomakeDefaultFile),
				BashComplete: runner.SRunBashComplete,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    DryRunCli,
						EnvVars: []string{getFlagEnvByFlagName(DryRunCli)},
						Value:   false,
						Usage:   fmt.Sprintf("Only show template paresed %s but not execute it", GomakeDefaultFile),
					},
					&cli.StringSliceFlag{
						Name:    VarsCli,
						Aliases: []string{"v"},
						EnvVars: []string{getFlagEnvByFlagName(VarsCli)},
						Action:  runner.ExtraVariables,
					},
				},
				Action: runner.SRun,
				Before: runner.RunBefore,
			},
		},
	}

	if err := app1.Run(os.Args); err != nil {
		fmt.Println("Error: ", err)
	}
}

type Runner struct {
	cmdHandler  command.CommandHandler
	interpreter interpreter.Interpreter
}

func (r *Runner) ExtraVariables(ctx *cli.Context, s []string) error {
	for _, a := range s {
		if ok := VariableFlagRegex.MatchString(a); !ok {
			return fmt.Errorf("Variable does not match: %s only \"key=value\" are allowed ", a)
		}
		splittetVar := strings.SplitN(a, "=", 2)
		r.interpreter.ExtraVariables[splittetVar[0]] = splittetVar[1]
	}
	return nil
}

func (r *Runner) RunBefore(c *cli.Context) error {
	r.Before(c)
	neededCommand := c.Args().Get(0)
	dryRun := c.Bool(DryRunCli)
	if neededCommand == "" {
		return fmt.Errorf("need name of executing command")
	}
	r.interpreter.ExecuteCommand = neededCommand
	r.interpreter.DryRun = dryRun
	return nil
}

func (r *Runner) CommandNotFound(c *cli.Context, cmd string) {
	possibleCommands := []string{}
	for _, v := range c.App.Commands {
		possibleCommands = append(possibleCommands, v.Names()...)
	}
	fmt.Printf("Command \"%s\" not found did you mean \n%s\n", cmd, nearfinder.ClosestMatch(cmd, possibleCommands, 1))
}

func (r *Runner) Before(c *cli.Context) error {
	makefile := c.Path(MakeFileCli)
	executer := c.String(ExecuterCli)
	f, err := os.ReadFile(makefile)
	if err != nil {
		return err
	}
	r.interpreter = interpreter.NewInterpreter(App, "", executer, c.Bool(DryRunCli), r.cmdHandler, f)
	return nil
}

func isFlagAtUseList(used []string, test string) bool {
	for _, v := range used {
		if v == test {
			return true
		}
	}
	return false
}

func (r *Runner) SRunBashComplete(c *cli.Context) {
	r.Before(c)
	r.RunBefore(c)
	autoCompleteHelp := make([]string, 0)

	for _, cs := range c.App.Commands {
		if cs.Name == "srun" {
			for _, csf := range cs.Flags {
				for _, csfn := range csf.Names() {
					if csfn == "help" || csfn == "h" {
						continue
					}
					if csfn == DryRunCli {
						if isFlagAtUseList(c.FlagNames(), csfn) {
							continue
						}
					}
					minusStr := "-"
					if len(csfn) > 1 {
						minusStr = "--"
					}
					autoCompleteHelp = append(autoCompleteHelp, fmt.Sprintf("%s%s", minusStr, csfn))
				}
			}
		}
	}
	list, _, _, err := r.interpreter.GetStageMap()
	if err != nil {
		return
	}
	for k := range list {
		autoCompleteHelp = append(autoCompleteHelp, k)
	}

	for _, v := range autoCompleteHelp {
		fmt.Println(v)
	}
}

func (r *Runner) RunBashComplete(c *cli.Context) {
	r.Before(c)
	r.RunBefore(c)
	autoCompleteHelp := make([]string, 0)

	for _, cs := range c.App.Commands {
		if cs.Name == "run" {
			for _, csf := range cs.Flags {
				for _, csfn := range csf.Names() {
					if csfn == "help" || csfn == "h" {
						continue
					}
					if csfn == DryRunCli {
						if isFlagAtUseList(c.FlagNames(), csfn) {
							continue
						}
					}
					minusStr := "-"
					if len(csfn) > 1 {
						minusStr = "--"
					}
					autoCompleteHelp = append(autoCompleteHelp, fmt.Sprintf("%s%s", minusStr, csfn))
				}
			}
		}
	}
	list, err := r.interpreter.GetMakeScripts()
	if err != nil {
		return
	}
	for k := range list {
		autoCompleteHelp = append(autoCompleteHelp, k)
	}

	for _, v := range autoCompleteHelp {
		fmt.Println(v)
	}
}

func (r *Runner) Autocomplete(c *cli.Context) error {
	persist := c.Bool(PersistAutocompleteCli)
	shell := c.String(ShellAutocompleteCli)

	if strings.HasSuffix(shell, "zsh") {
		err := os.Setenv("PROG", c.App.Name)
		if err != nil {
			return err
		}

		f, err := autocompleteFiles.ReadFile("autocomplete/zsh_autocomplete")
		if err != nil {
			return err
		}
		if persist {
			path := fmt.Sprintf("/etc/bash_completion.d/%s", c.App.Name)
			err = os.WriteFile(path, f, 0644)
			if err != nil {
				return err
			}
			fmt.Printf("Autocomplete was added to %s \n reload shell to activate or \n export PROG=%s \n source %s", path, c.App.Name, path)
		} else {
			err = os.WriteFile("/usr/tmp/zsh_autocomplete", f, 0644)
			if err != nil {
				return err
			}
			fmt.Printf("Execute Command \n export PROG=%s \n source /usr/tmp/zsh_autocomplete", c.App.Name)
		}
	}
	if strings.HasSuffix(shell, "bash") {
		err := os.Setenv("PROG", c.App.Name)
		if err != nil {
			return err
		}

		f, err := autocompleteFiles.ReadFile("autocomplete/bash_autocomplete")
		if err != nil {
			return err
		}
		err = os.WriteFile("/usr/tmp/bash_autocomplete", f, 0644)
		if err != nil {
			return err
		}
		fmt.Printf("Execute Command \n export PROG=%s \n source /usr/tmp/bash_autocomplete", c.App.Name)
	}

	return nil
}

func (r *Runner) Run(c *cli.Context) error {
	return r.interpreter.Run()
}

func (r *Runner) SRun(c *cli.Context) error {
	return r.interpreter.SRun()
}

func (r *Runner) Init(c *cli.Context) error {
	_, err := os.Stat(GomakeDefaultFile)
	if err != nil {
		f, err := os.Create(GomakeDefaultFile)
		defer f.Close()
		if err != nil {
			return err
		}
		_, err = f.Write(gomakeIni)
		if err != nil {
			return err
		}
		fmt.Printf("%s was created\nyou can execute with \ngomake run run", GomakeDefaultFile)
		return nil
	}
	return fmt.Errorf("%s already exist", GomakeDefaultFile)

}

func (r *Runner) List(c *cli.Context) error {
	list, err := r.interpreter.GetMakeScripts()
	if err != nil {
		return err
	}
	fmt.Println("List of executed Commands (for run):")
	for k := range list {
		fmt.Println(k)
	}

	fmt.Println("\nList of executed Stages (for srun):")
	stages, _, _, err := r.interpreter.GetStageMap()
	if err != nil {
		return err
	}
	for k := range stages {
		fmt.Println(k)
	}
	return nil
}
