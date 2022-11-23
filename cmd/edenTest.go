package cmd

import (
	"fmt"
	"os"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newTestCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var tstCfg openevec.TestArgs

	var testCmd = &cobra.Command{
		Use:   "test [test_dir]",
		Short: "Run tests",
		Long: `Run tests from test binary. Verbose testing works with any level of general verbosity above "info"

test <test_dir> [-s <scenario>] [-t <timewait>] [-v <level>]
test <test_dir> -l <regexp>
test <test_dir> -o
test <test_dir> -r <regexp> [-t <timewait>] [-v <level>]

`,
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if len(args) != 0 {
				log.Debug("DIR: ", args[0])
				tstCfg.CurDir, err = os.Getwd()
				if err != nil {
					return err
				}
				err = os.Chdir(args[0])
				if err != nil {
					return err
				}
			}

			vars, err := utils.InitVars()

			if err != nil {
				return fmt.Errorf("error reading config: %s\n", err)
			}

			if tstCfg.TestProg == "" {
				if vars.TestProg != "" {
					tstCfg.TestProg = vars.TestProg
				} else {
					tstCfg.TestProg = defaults.DefaultTestProg
				}
			}
			if tstCfg.TestScenario == "" {
				tstCfg.TestScenario = vars.TestScenario
			}

			if tstCfg.TestScenario == "" && tstCfg.TestProg == "" && tstCfg.TestRun == "" {
				return fmt.Errorf("Please set the --scenario option or environment variable eden.test-scenario in the EDEN configuration.")
			}
			tstCfg.ConfigFile = cfg.ConfigFile
			tstCfg.Verbosity = *verbosity
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {

			if err := openevec.Test(&tstCfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	testCmd.Flags().StringVarP(&tstCfg.TestEscript, "escript", "e", "", "run EScript matching the regular expression")
	testCmd.Flags().StringVarP(&tstCfg.TestProg, "prog", "p", "", "program binary to run tests")
	testCmd.Flags().StringVarP(&tstCfg.TestRun, "run", "r", "", "run only those tests matching the regular expression")
	testCmd.Flags().StringVarP(&tstCfg.TestTimeout, "timeout", "t", "", "panic if test exceded the timeout")
	testCmd.Flags().StringVarP(&tstCfg.TestArgs, "args", "a", "", "Arguments for test binary")
	testCmd.Flags().StringVarP(&tstCfg.TestList, "list", "l", "", "list tests matching the regular expression")
	testCmd.Flags().StringVarP(&tstCfg.TestScenario, "scenario", "s", "", "scenario for tests bunch running")
	testCmd.Flags().StringVarP(&tstCfg.FailScenario, "fail_scenario", "f", "cfg.FailScenario.txt", "scenario for test failing")
	testCmd.Flags().BoolVarP(&tstCfg.TestOpts, "opts", "o", false, "Options description for test binary which may be used in test scenarious and '-a|--args' option")

	return testCmd
}
