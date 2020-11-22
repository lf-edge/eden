package tests

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

//RunTest -- single test runner.
func RunTest(testApp string, args []string, testArgs string, testTimeout string, failScenario string, configFile string, verbosity string) {
	if testApp != "" {
		log.Debug("testApp: ", testApp)
		vars, err := utils.InitVars()
		if err != nil {
			log.Fatalf("error reading config: %s\n", err)
			return
		}
		_, err = exec.LookPath(testApp)
		if err != nil {
			testApp = utils.ResolveAbsPath(vars.EdenBinDir + "/" + testApp)
		}

		_, err = os.Stat(testApp)
		if os.IsNotExist(err) {
			log.Fatalf("Test binary file %s does not exist\n", testApp)
			return
		}
		if err != nil {
			log.Fatalf("Error reading test binary %s: %s", testApp, err)
			return
		}

		path, err := exec.LookPath(testApp)
		if err != nil {
			log.Fatalf("Cannot find executable %s", testApp)
			return
		}

		log.Debug("testProg: ", path)
		if testTimeout != "" {
			args = append(args, "-test.timeout", testTimeout)
		}
		if verbosity != "info" {
			args = append(args, "-test.v")
		}
		done := make(chan bool, 1)
		go func() {
			ticker := time.NewTicker(defaults.DefaultRepeatTimeout * defaults.DefaultRepeatCount)
			for {
				select {
				case tickTime := <-ticker.C:
					//we need to log periodically to avoid stopping of ci/cd system
					log.Infof("Test is running: %s",
						tickTime.Format(time.RFC3339))
				case <-done:
					ticker.Stop()
					return
				}
			}
		}()
		resultArgs := append(args, strings.Fields(testArgs)...)
		log.Debugf("Test: %s %s", path, strings.Join(resultArgs, " "))
		tst := exec.Command(path, resultArgs...)
		tst.Stdout = os.Stdout
		tst.Stderr = os.Stderr
		err = tst.Run()
		close(done)
		if err != nil && failScenario != "" {
			log.Debug("failScenario: ", failScenario)
			RunScenario("", "", testTimeout, "",
				configFile, "")
			os.Exit(1)
		}
	}
}

//RunScenario -- run a scenario with a test suite
func RunScenario(testScenario string, testArgs string, testTimeout string, failScenario string, configFile string, verbosity string) {
	if testScenario == "" {
		return
	}
	// is it path to file?
	_, err := os.Stat(testScenario)
	if os.IsNotExist(err) {
		testScenario = utils.ResolveAbsPath(testScenario)
		_, err = os.Stat(testScenario)
		if os.IsNotExist(err) {
			log.Fatalf("Scenario file '%s' is not exist\n", testScenario)
			return
		}
		if err != nil {
			log.Fatalf("Scenario file '%s' error reading: %s\n", testScenario, err)
			return
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
		for i, part := range targs {
			// Handle defined args
			flagsParsed := make(map[string]string)
			// parse provided testArgs
			flags := strings.Split(strings.Trim(testArgs, "\""), ",")
			for _, el := range flags {
				fl := strings.TrimPrefix(el, "-")
				fl = strings.TrimPrefix(fl, "-")
				split := strings.SplitN(fl, "=", 2)
				if len(split) == 2 {
					flagsParsed[strings.TrimSpace(split[0])] = strings.TrimSpace(split[1])
				}
			}
			// parse args from scenario
			splitStr := strings.SplitN(part, "args=\"", 2)
			if len(splitStr) == 2 {
				flags := strings.Split(strings.SplitN(splitStr[1], "\"", 2)[0], ",")
				for _, el := range flags {
					fl := strings.TrimPrefix(el, "-")
					fl = strings.TrimPrefix(fl, "-")
					split := strings.SplitN(fl, "=", 2)
					if len(split) == 2 {
						if _, ok := flagsParsed[strings.TrimSpace(split[0])]; !ok { // do not overwrite flags from args
							flagsParsed[strings.TrimSpace(split[0])] = strings.TrimSpace(split[1])
						}
					}
				}

				// merge result map into args
				var resultArgs []string
				for k, v := range flagsParsed {
					resultArgs = append(resultArgs, fmt.Sprintf("%s=%s", k, v))
				}
				targs[i] = fmt.Sprintf("-args=\"%s\"", strings.Join(resultArgs, ","))
				log.Info(targs[i])
			}
		}
		RunTest(targs[0], targs[1:], testArgs, testTimeout,
			failScenario, configFile, verbosity)
	}
}
