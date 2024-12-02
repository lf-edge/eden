package template

import (
	"os"
	"strings"
	"testing"

	"github.com/bloomberg/go-testgroup"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
	log "github.com/sirupsen/logrus"
)

// This is the struct that holds all the tests in this suite together,
// we can add members to this struct if necccasary to share state between tests.
type TemplateTest struct{}

// This is the name of the test suite, it is used to create the project name
// rename it according to the test suite you are writing.
const testSuiteName = "Template"
const projectName = testSuiteName + "-tests"

// the eve node to run the tests and the test group, no need to modify.
var eveNode *tk.EveNode
var group = &TemplateTest{}

// This is the main function that runs the tests, it initializes the test suite
// and runs the tests, usually there is no need to modify this function.
func TestMain(m *testing.M) {
	log.Printf("%s Test Suite started\n", testSuiteName)
	defer log.Printf("%s Test Suite finished\n", testSuiteName)

	node, err := tk.InitilizeTest(projectName, tk.WithControllerVerbosity("debug"))
	if err != nil {
		log.Fatalf("Failed to initialize test: %v", err)
	}

	eveNode = node
	res := m.Run()
	os.Exit(res)
}

// This is the main test function, it checks if the workflow is "template" and
// runs the tests that are grouped together with the TemplateTest struct. You
// usually don't need to modify this function.
func TestTemplate(t *testing.T) {
	if !strings.EqualFold(os.Getenv("WORKFLOW"), testSuiteName) {
		t.Skipf("Skip %s tests in non-%s workflow", testSuiteName, testSuiteName)
	}

	testgroup.RunSerially(t, group)
}

// This is a sample test, all the tests with signature
// func (grp *TemplateTest) Test*(*testgroup.T) are run by the test runner.
// it is better to keeo each test (or group of related tests) in a separate file.
// Please note that file name must end with _test.go to be recognized by go test.
func (grp *TemplateTest) TestSimpleTest(_ *testgroup.T) {
	eveNode.LogTimeInfof("TestSimpleTest started")
	defer eveNode.LogTimeInfof("TestSimpleTest finished")

	_, err := eveNode.EveRunCommand("echo \"Hello, World!\"")
	if err != nil {
		eveNode.LogTimeFatalf("Failed to execute command on eve: %v", err)
	}
}
