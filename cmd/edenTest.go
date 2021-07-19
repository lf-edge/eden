package cmd

import (
	"os"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/tests"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	testArgs     string
	testOpts     bool
	testEscript  string
	testRun      string
	testTimeout  string
	testList     string
	testProg     string
	testScenario string
	failScenario string
	curDir       string
)

var testCmd = &cobra.Command{
	Use:   "test [test_dir]",
	Short: "Run tests",
	Long: `Run tests from test binary. Verbose testing works with any level of general verbosity above "info"

test <test_dir> [-s <scenario>] [-t <timewait>] [-v <level>]
test <test_dir> -l <regexp>
test <test_dir> -o
test <test_dir> -r <regexp> [-t <timewait>] [-v <level>]

`,
	Args: cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			var err error
			log.Debug("DIR: ", args[0])
			curDir, err = os.Getwd()
			if err != nil {
				log.Fatal(err)
			}
			err = os.Chdir(args[0])
			if err != nil {
				log.Fatal(err)
			}
		}

		vars, err := utils.InitVars()
		if err != nil {
			log.Fatalf("error reading config: %s\n", err)
			return err
		}

		if testProg == "" {
			if vars.TestProg != "" {
				testProg = vars.TestProg
			} else {
				testProg = defaults.DefaultTestProg
			}
		}
		if testScenario == "" {
			testScenario = vars.TestScenario
		}

		if testScenario == "" && testProg == "" && testRun == "" {
			log.Fatal("Please set the --scenario option or environment variable eden.test-scenario in the EDEN configuration.")
			return err
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if curDir != "" {
				err := os.Chdir(curDir)
				if err != nil {
					log.Fatal(err)
				}
			}
		}()
		switch {
		case testList != "":
			tests.RunTest(testProg, []string{"-test.list", testList}, "", testTimeout, failScenario, configFile, verbosity)
			return
		case testOpts:
			tests.RunTest(testProg, []string{"-h"}, "", testTimeout, failScenario, configFile, verbosity)
			return
		case testEscript != "":
			tests.RunTest("eden.escript.test", []string{"-test.run", "TestEdenScripts/" + testEscript}, testArgs, testTimeout, failScenario, configFile, verbosity)
			return
		case testRun != "":
			tests.RunTest(testProg, []string{"-test.run", testRun}, testArgs, testTimeout, failScenario, configFile, verbosity)
			return
		default:
			tests.RunScenario(testScenario, testArgs, testTimeout, failScenario, configFile, verbosity)
			return
		}
	},
}

func testInit() {
	testCmd.Flags().StringVarP(&testEscript, "escript", "e", "", "run EScript matching the regular expression")
	testCmd.Flags().StringVarP(&testProg, "prog", "p", "", "program binary to run tests")
	testCmd.Flags().StringVarP(&testRun, "run", "r", "", "run only those tests matching the regular expression")
	testCmd.Flags().StringVarP(&testTimeout, "timeout", "t", "", "panic if test exceded the timeout")
	testCmd.Flags().StringVarP(&testArgs, "args", "a", "", "Arguments for test binary")
	testCmd.Flags().StringVarP(&testList, "list", "l", "", "list tests matching the regular expression")
	testCmd.Flags().StringVarP(&testScenario, "scenario", "s", "", "scenario for tests bunch running")
	testCmd.Flags().StringVarP(&failScenario, "fail_scenario", "f", "failScenario.txt", "scenario for test failing")
	testCmd.Flags().BoolVarP(&testOpts, "opts", "o", false, "Options description for test binary which may be used in test scenarious and '-a|--args' option")
}
