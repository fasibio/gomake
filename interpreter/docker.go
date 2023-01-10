package interpreter

import (
	"fmt"
	"strings"

	"github.com/fasibio/gomake/command"
)

func getDockerCmd(cmd string, docker *command.DockerOperation) string {
	executer := "/bin/sh"
	if docker.Executer != "" {
		executer = docker.Executer
	}

	var sb strings.Builder
	sb.WriteString("docker run --rm -it ")
	for _, v := range docker.Volumes {
		sb.WriteString(fmt.Sprintf("-v %s ", v))
	}
	sb.WriteString(fmt.Sprintf("%s ", docker.Name))
	sb.WriteString(fmt.Sprintf("%s -c '%s'", executer, cmd))

	cmd = sb.String()
	return cmd
}
