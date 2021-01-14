package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

$ source <(eden completion bash)

# To load completions for each session, execute once:
Linux:
  $ eden utils completion bash > /etc/bash_completion.d/eden
MacOS:
  $ eden utils completion bash > /usr/local/etc/bash_completion.d/eden

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ eden utils completion zsh > "${fpath[1]}/_eden"

# You will need to start a new shell for this setup to take effect.

Fish:

$ eden utils completion fish | source

# To load completions for each session, execute once:
$ eden utils completion fish > ~/.config/fish/completions/eden.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			err := cmd.Root().GenBashCompletion(os.Stdout)
			if err != nil {
				fmt.Fprintf(os.Stderr,
					"Completions generation error: %s",
					err.Error())
			}
		case "zsh":
			err := cmd.Root().GenZshCompletion(os.Stdout)
			if err != nil {
				fmt.Fprintf(os.Stderr,
					"Completions generation error: %s",
					err.Error())
			}
		case "fish":
			err := cmd.Root().GenFishCompletion(os.Stdout, true)
			if err != nil {
				fmt.Fprintf(os.Stderr,
					"Completions generation error: %s",
					err.Error())
			}
		}
	},
}
