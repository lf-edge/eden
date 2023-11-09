package sec_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eden/pkg/tests"
)

var (
	tc    *projects.TestContext
	rnode *remoteNode
)

// TestMain is used to provide setup and teardown for the rest of the
// tests. As part of setup we make sure that context has a slice of
// EVE instances that we can operate on. For any action, if the instance
// is not specified explicitly it is assumed to be the first one in the slice
func TestMain(m *testing.M) {
	log.Println("Security Test Suite started")

	tests.TestArgsParse()

	tc = projects.NewTestContext()

	projectName := fmt.Sprintf("%s_%s", "TestSecurity", time.Now())

	// Registering our own project namespace with controller for easy cleanup
	tc.InitProject(projectName)

	// Create representation of EVE instances (based on the names
	// or UUIDs that were passed in) in the context. This is the first place
	// where we're using zcli-like API:
	for _, node := range tc.GetNodeDescriptions() {
		edgeNode := node.GetEdgeNode(tc)
		if edgeNode == nil {
			// Couldn't find existing edgeNode record in the controller.
			// Need to create it from scratch now:
			// this is modeled after: zcli edge-node create <name>
			// --project=<project> --model=<model> [--title=<title>]
			// ([--edge-node-certificate=<certificate>] |
			// [--onboarding-certificate=<certificate>] |
			// [(--onboarding-key=<key> --serial=<serial-number>)])
			// [--network=<network>...]
			//
			// XXX: not sure if struct (giving us optional fields) would be better
			edgeNode = tc.NewEdgeNode(tc.WithNodeDescription(node), tc.WithCurrentProject())
		} else {
			// make sure to move EdgeNode to the project we created, again
			// this is modeled after zcli edge-node update <name> [--title=<title>]
			// [--lisp-mode=experimental|default] [--project=<project>]
			// [--clear-onboarding-certs] [--config=<key:value>...] [--network=<network>...]
			edgeNode.SetProject(projectName)
		}

		tc.ConfigSync(edgeNode)

		// finally we need to make sure that the edgeNode is in a state that we need
		// it to be, before the test can run -- this could be multiple checks on its
		// status, but for example:
		if edgeNode.GetState() == device.NotOnboarded {
			log.Fatal("Node is not onboarded now")
		}

		// this is a good node -- lets add it to the test context
		tc.AddNode(edgeNode)
	}

	tc.StartTrackingState(false)

	// create a remote node
	rnode = createRemoteNode()
	if rnode == nil {
		log.Fatal("Can't initlize the remote node")
	}

	// we now have a situation where TestContext has enough EVE nodes known
	// for the rest of the tests to run. So run them:
	res := m.Run()

	// Finally, we need to cleanup whatever objects may be in in the
	// project we created and then we can exit
	os.Exit(res)
}

func TestCordumpDisabled(t *testing.T) {
	log.Println("TestCordumpDisabled started")
	defer log.Println("TestCordumpDisabled finished")

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))
	tc.WaitForState(edgeNode, 60)

	// check if cordump is disabled
	out, err := rnode.runCommand("sysctl kernel.core_pattern")
	if err != nil {
		t.Fatal(err)
	}

	log.Println(string(out))
	if strings.Contains(string(out), "core") {
		t.Fatal("Core dumps are enabled")
	}
}

func TestProcessRunningAsRoot(t *testing.T) {
	// TODO : this is not a proper way to check, but good for now
	log.Println("TestProcessRunningAsRoot started")
	defer log.Println("TestProcessRunningAsRoot finished")

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))
	tc.WaitForState(edgeNode, 60)

	// check if there are any processes running as root
	out, err := rnode.runCommand("ps aux -U root -u root")
	if err != nil {
		t.Fatal(err)
	}

	if len(out) > 0 {
		log.Println(string(out))
		t.Fatal("There are processes running as root on the system")
	}
}

func TestAppArmorEnabled(t *testing.T) {
	log.Println("TestAppArmorEnabled started")
	defer log.Println("TestAppArmorEnabled finished")

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))
	tc.WaitForState(edgeNode, 60)

	out, err := rnode.readFile("/sys/module/apparmor/parameters/enabled")
	if err != nil {
		t.Fatal(err)
	}

	exits := strings.TrimSpace(string(out))
	if exits != "Y" {
		t.Fatal("AppArmor is not enabled")
	}
}

func TestCheckMountOptions(t *testing.T) {
	log.Println("TestCheckMountOptions started")
	defer log.Println("TestCheckMountOptions finished")

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))
	tc.WaitForState(edgeNode, 60)

	fail := false
	mounts, err := rnode.getMountPoints("")
	if err != nil {
		t.Fatal(err)
	}

	// checl of mounts of type proc are secure
	misconfig := checkMountOptionsByType("proc", mounts, []string{"nosuid", "nodev", "noexec"})
	if len(misconfig) > 0 {
		for _, msg := range misconfig {
			t.Logf("[FAIL] %s", msg)
		}
		fail = true
	}

	// TODO: set hidepid=2 on /proc and this to the above list
	misconfig = checkMountOptionsByType("proc", mounts, []string{"hidepid=2"})
	if len(misconfig) > 0 {
		for _, msg := range misconfig {
			t.Logf("[FAIL] %s", msg)
		}
	}

	// check of mounts of type tmpfs are secure
	misconfig = checkMountOptionsByType("tmpfs", mounts, []string{"nosuid", "nodev", "noexec"})
	if len(misconfig) > 0 {
		for _, msg := range misconfig {
			t.Logf("[FAIL] %s", msg)
		}
		fail = true
	}

	if fail {
		t.Fatal("Some mount options are not secure, see logs above")
	}
}

func TestCheckTmpIsSecure(t *testing.T) {
	log.Println("TestCheckTempIsSecure started")
	defer log.Println("TestCheckTempIsSecure finished")

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))
	tc.WaitForState(edgeNode, 60)

	mounts, err := rnode.getMountPoints("tmpfs")
	if err != nil {
		t.Fatal(err)
	}

	fail := false
	for _, mount := range mounts {
		p := perm{}
		if err := rnode.getPathPerm(mount.Path, &p); err != nil {
			t.Fatal(err)
		}

		if p.user != "root" || p.group != "root" {
			t.Logf("[FAIL] %s is not owned by root:root", mount.Path)
			fail = true
		}

		if !strings.Contains(p.perms, "t") {
			t.Logf("[FAIL] %s is not sticky", mount.Path)
			fail = true
		}
	}

	if fail {
		t.Fatal("Some tmpfs mounts are not secure, see logs above")
	}
}

func checkMountSecurityOptions(mount mount, secureOptions []string) []string {
	secOptNotFound := make([]string, 0)

	for _, option := range secureOptions {
		if !strings.Contains(mount.Options, option) {
			secOptNotFound = append(secOptNotFound, fmt.Sprintf("'%s' option is not set on %s", option, mount.Path))
		}
	}

	return secOptNotFound
}

func checkMountOptionsByType(mountType string, mounts []mount, options []string) []string {
	secOptNotFound := make([]string, 0)
	for _, mount := range mounts {
		if mount.Type == mountType {
			misses := checkMountSecurityOptions(mount, options)
			secOptNotFound = append(secOptNotFound, misses...)
		}
	}

	return secOptNotFound
}

func checkMountOptionsByPath(mountPath string, mounts []mount, options []string) []string {
	secOptNotFound := make([]string, 0)
	for _, mount := range mounts {
		if mount.Path == mountPath {
			misses := checkMountSecurityOptions(mount, options)
			secOptNotFound = append(secOptNotFound, misses...)
		}
	}

	return secOptNotFound
}
