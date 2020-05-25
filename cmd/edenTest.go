package cmd

import (
	"bufio"
	"github.com/lf-edge/eden/pkg/defaults"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	//"github.com/spf13/viper"
	"github.com/lf-edge/eden/pkg/utils"
)

var (
	testRun     string
	testTimeout string
	testList    string
	testProg    string
	testScript  string
)

func runTest(args []string) {
	path, err := exec.LookPath(testProg)
	if err != nil {
		log.Fatalf("didn't find '%s' executable\n", testProg)
		return
	}
	if testTimeout != "" {
		args = append(args, "-test.timeout", testTimeout)
	}
	if verbosity != "info" {
		args = append(args, "-test.v")
	}
	tstr := path
	for _, arg := range args {
		tstr += " " + arg
	}
	log.Info("Test: ", tstr)
	tst := exec.Command(path, args...)
	tst.Stdout = os.Stdout
	tst.Stderr = os.Stderr
	err = tst.Run()
	if err != nil {
		log.Fatalf("Test running failed with %s\n", err)
	}
}

func runScript() {
	file, err := os.Open(testScript)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	log.Info("runScript: ")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var targs []string
		str := scanner.Text()
		targs = strings.Split(str, " ")
		runTest(targs)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run tests",
	Long: `Run tests from test binary. Verbose testing works with any level of general verbosity above "info"

test [-s <script>] [-t <timewait>] [-v <level>]
test -l <regexp>
test -r <regexp> [-t <timewait>] [-v <level>]

`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		vars, err := utils.InitVars()
		if err != nil {
			log.Fatalf("error reading config: %s\n", err)
			return err
		}

		if testProg == "" {
			testProg = vars.TestProg
		}
		if testScript == "" {
			testScript = vars.TestScript
		}

		_, err = exec.LookPath(testProg)
		if err != nil {
			testProg = utils.ResolveAbsPath(vars.EdenBinDir + "/" + testProg)
		}

		// is it path to file?
		_, err = os.Stat(testScript)
		if os.IsNotExist(err) {
			testScript = utils.ResolveAbsPath(testScript)
		}

		log.Debug("testProg: ", testProg)
		log.Debug("testScript:", testScript)
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		switch {
		case testList != "":
			runTest([]string{"-test.list", testList})
			return
		case testRun != "":
			runTest([]string{"-test.run", testRun})
			return
		case testScript != "":
			runScript()
			return
		}
	},
}

func testInit() {
	testCmd.Flags().StringVarP(&testProg, "prog", "p", defaults.DefaultTestProg, "program binary to run tests")
	testCmd.Flags().StringVarP(&testRun, "run", "r", "", "run only those tests matching the regular expression")
	testCmd.Flags().StringVarP(&testTimeout, "timeout", "t", "", "panic if test exceded the timeout")
	testCmd.Flags().StringVarP(&testList, "list", "l", "", "list tests matching the regular expression")
	testCmd.Flags().StringVarP(&testScript, "script", "s", "", "script for tests bunch running")
}
