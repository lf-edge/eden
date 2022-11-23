package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/openevec"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newUtilsCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}

	var utilsCmd = &cobra.Command{
		Use:               "utils",
		Short:             "Eden utilities",
		Long:              `Additional utilities for EDEN.`,
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newCompletionCmd(),
				newTemplateCmd(cfg),
				newDownloaderCmd(cfg),
				newOciImageCmd(),
				newCertsCmd(cfg),
				newGcpCmd(cfg),
				newSdInfoEveCmd(),
				newDebugCmd(cfg),
				newUploadGitCmd(),
				newImportCmd(cfg),
				newExportCmd(cfg),
			},
		},
	}

	groups.AddTo(utilsCmd)

	return utilsCmd
}

func newSdInfoEveCmd() *cobra.Command {
	var syslogOutput, eveReleaseOutput string

	var sdInfoEveCmd = &cobra.Command{
		Use:   "sd <SD_DEVICE_PATH>",
		Short: "get info from SD card",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			devicePath := args[0]
			if err := openevec.SdInfoEve(devicePath, syslogOutput, eveReleaseOutput); err != nil {
				log.Fatal(err)
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	sdInfoEveCmd.Flags().StringVar(&syslogOutput, "syslog-out", filepath.Join(currentPath, "syslog.txt"), "File to save syslog.txt")
	sdInfoEveCmd.Flags().StringVar(&eveReleaseOutput, "everelease-out", filepath.Join(currentPath, "eve-release"), "File to save eve-release")

	return sdInfoEveCmd
}

func newUploadGitCmd() *cobra.Command {
	var uploadGitCmd = &cobra.Command{
		Use: "gitupload <file or directory> " +
			"<git repo in notation https://GIT_LOGIN:GIT_TOKEN@GIT_REPO> <branch> [directory in git]",
		Long: "Upload file or directory to provided git branch into directory with the same name as branch " +
			"or into provided directory",
		Args: cobra.RangeArgs(3, 4),
		Run: func(cmd *cobra.Command, args []string) {
			if _, err := os.Stat(args[0]); os.IsNotExist(err) {
				log.Fatal(err)
			}

			absPath, err := filepath.Abs(args[0])
			if err != nil {
				log.Fatal(err)
			}
			directoryToSaveOnGit := args[2]
			if len(args) == 4 {
				directoryToSaveOnGit = args[3]
			}

			if err := openevec.UploadGit(absPath, args[1], args[2], directoryToSaveOnGit); err != nil {
				log.Fatal(err)
			}
		},
	}
	return uploadGitCmd
}

// this is circular dependency command
func newCompletionCmd() *cobra.Command {
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
	return completionCmd
}

func newTemplateCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var templateCmd = &cobra.Command{
		Use:   "template <file>",
		Short: "Render template",
		Long:  ``,
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			tmpl, err := ioutil.ReadFile(args[0])
			if err != nil {
				log.Fatal(err)
			}
			out, err := utils.RenderTemplate(cfg.ConfigFile, string(tmpl))
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(out)
		},
	}
	return templateCmd
}

func newExportCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var exportCmd = &cobra.Command{
		Use:   "export <filename>",
		Short: "export harness",
		Long:  `Export certificates and configs of harness into tar.gz file.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			tarFile := args[0]
			if err := openevec.EdenExport(tarFile, cfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	return exportCmd

}

func newImportCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var rewriteRoot bool

	var importCmd = &cobra.Command{
		Use:   "import <filename>",
		Short: "import harness",
		Long:  `Import certificates and configs of harness from tar.gz file.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			tarFile := args[0]
			if err := openevec.EdenImport(tarFile, rewriteRoot, cfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	importCmd.Flags().BoolVar(&rewriteRoot, "rewrite-root", true, "Rewrite eve.root with local value")

	return importCmd
}
