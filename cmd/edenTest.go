package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
)

var (
	testArgs     string
	testOpts     bool
	testRun      string
	testTimeout  string
	testList     string
	testProg     string
	testScenario string
	curDir       string
)

func runTest(testApp string, args []string, testArgs string) {
	if testApp != "" {
		log.Debug("testApp: ", testApp)
		vars, err := utils.InitVars()
		if err != nil {
			log.Fatalf("error reading config: %s\n", err)
			return
		}
		path, err := exec.LookPath(testApp)
		if err != nil {
			testApp = utils.ResolveAbsPath(vars.EdenBinDir + "/" + testApp)
		}

		_, err = os.Stat(testApp)
		if os.IsNotExist(err) {
			log.Fatalf("Test binary file %s does not exist\n", testApp)
			return
		} else {
			if err != nil {
				log.Fatalf("Error reading test binary %s\n", testApp, err)
				return
			}
		}

		path, err = exec.LookPath(testApp)
		if err != nil {
			log.Fatalf("Cannot find executable %s\n", testApp)
			return
		}

		log.Debug("testProg: ", path)
		if testTimeout != "" {
			args = append(args, "-test.timeout", testTimeout)
		}
		if verbosity != "info" {
			args = append(args, "-test.v")
		}

		resultArgs := append(args, strings.Fields(testArgs)...)
		log.Debugf("Test: %s %s", path, strings.Join(resultArgs, " "))
		tst := exec.Command(path, resultArgs...)
		tst.Stdout = os.Stdout
		tst.Stderr = os.Stderr
		err = tst.Run()
		if err != nil {
			log.Fatalf("Test running failed with %s\n", err)
		}
	}
}

func runScenario(testArgs string) {
	// is it path to file?
	_, err := os.Stat(testScenario)
	if os.IsNotExist(err) {
		testScenario = utils.ResolveAbsPath(testScenario)
		_, err = os.Stat(testScenario)
		if os.IsNotExist(err) {
			log.Fatalf("Scenario file %s is not exist\n", testScenario)
			return
		} else {
			if err != nil {
				log.Fatalf("Scenario file %s error reading: %s\n", testScenario, err)
				return
			}
		}
	}

	log.Debug("testScenario:", testScenario)

	tmpl, err := ioutil.ReadFile(testScenario)
	if err != nil {
		log.Fatal(err)
	}

	out, err := utils.RenderTemplate(configFile, string(tmpl))
	if err != nil {
		log.Fatal(err)
	}
	strs := strings.Split(out, "\n")
	var targs []string
	for _, str := range strs {
		// Handle line comments
		str = strings.Split(str, "#")[0]
		str = strings.Split(str, "//")[0]
		targs = strings.Split(str, " ")
		runTest(targs[0], targs[1:], testArgs)
	}
}

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
			testProg = vars.TestProg
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
		switch {
		case testList != "":
			runTest(testProg, []string{"-test.list", testList}, "")
			return
		case testOpts:
			runTest(testProg, []string{"-h"}, "")
			return
		case testRun != "":
			runTest(testProg, []string{"-test.run", testRun}, testArgs)
			return
		case testScenario != "":
			runScenario(testArgs)
			return
		default:
			runScenario(testArgs)
			return
		}

		if curDir != "" {
			err := os.Chdir(curDir)
			if err != nil {
				log.Fatal(err)
			}
		}
	},
}

func testInit() {
	testCmd.Flags().StringVarP(&testProg, "prog", "p", defaults.DefaultTestProg, "program binary to run tests")
	testCmd.Flags().StringVarP(&testRun, "run", "r", "", "run only those tests matching the regular expression")
	testCmd.Flags().StringVarP(&testTimeout, "timeout", "t", "", "panic if test exceded the timeout")
	testCmd.Flags().StringVarP(&testArgs, "args", "a", "", "Arguments for test binary")
	testCmd.Flags().StringVarP(&testList, "list", "l", "", "list tests matching the regular expression")
	testCmd.Flags().StringVarP(&testScenario, "scenario", "s", "", "scenario for tests bunch running")
	testCmd.Flags().BoolVarP(&testOpts, "opts", "o", false, "Options description for test binary which may be used in test scenarious and '-a|--args' option")
}
