package cmd

import "github.com/spf13/cobra"

type CommandGroup struct {
	Message  string
	Commands []*cobra.Command
}

type CommandGroups []CommandGroup

func (g CommandGroups) AddTo(c *cobra.Command) {
	for _, group := range g {
		c.AddCommand(group.Commands...)
	}
}
