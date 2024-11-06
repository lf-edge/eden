# How to write tests with Eden

The eden golang test SDK for eve consists of two categories of functions:

1. `openevec`: infrastructure manager. `openevec` enables you to deploy a controller, a backend for that controller, deploy an edge node, etc. It does not change anything on an individual edge node.
2. `evetestkit`: provides useful collection of functions to describe expected state of the system (controller, EVE, AppInstances)

And those functions are used in context of standard golang test library. We do a setup in TestMain (for more info check [this](https://pkg.go.dev/testing#hdr-Main)) and then write test functions which interact with the environment we created. Setting up environment takes couple of minutes, so it makes sense to do it once and run tests within that environment. 

Source for the example below can be found [here](../tests/sec/sec_test.go)

In order to test a feature in Eden you need to

## 1. Create a configuration file and describe your environment

The glue between `openevec` and `evetestkit` is a configuration structure called `EdenSetupArgs`. It contains all the necessary information to setup all the components of the test: controller, EVE, etc. You can fill in the structure manually, however `openevec` provides you with a convenient function to create default configuration structure providing project root path:

```go
// ...
func TestMain(m *testing.M) {
  currentPath, err := os.Getwd()
  if err != nil {
    log.Fatal(err)
  }
  twoLevelsUp := filepath.Dir(filepath.Dir(currentPath))
  
  cfg := openevec.GetDefaultConfig(twoLevelsUp)
  
  if err = openevec.ConfigAdd(cfg, cfg.ConfigName, "", false); err != nil {
    log.Fatal(err)
  }
  // ...
}
```

Due to backward compatibility with escript as of time of writing, the project root path should be Eden repository folder. So if you add tests in the `eden/tests/my-awesome-test` you need to go two levels up, will be removed with escript
Also we need to write configuration file to file system, because there are components (like changer) which read configuration from file system, will be removed with escript

## 2. Initialize `openevec` Setup, Start and Onboard EVE node

When configuration you need `openevec` to create all needed certificates, start backend. For that we create `openevec` object based on a configuration provided, it is just a convinient wrapper.

```go
// ... 
func TestMain(m *testing.M) {
	// ...
	evec := openvec.CreateOpenEVEC(cfg)

	evec.SetupEden(/* ... */)
	evec.StartEden(/* ... */)
	evec.OnboardEve(/* ... */)

  // ...
}
```

## 3. Initialize `evetestkit` and run test suite

`evetestkit` provides an abstraction over EveNode, which is used to describe expected state of the system. Each EveNode is running within a project You can create a global object within one test file to use it across multiple tests. Note that EveNode is not threadsafe, since controller is stateful, so tests should be run consequently (no t.Parallel())

```go
const projectName = "security-test"
var eveNode *evetestkit.EveNode

func TestMain(m *testing.M) {
  // ...
	node, err := evetestkit.InitializeTestFromConfig(projectName, cfg, evetestkit.WithControllerVerbosity("debug"))
	if err != nil {
		log.Fatalf("Failed to initialize test: %v", err)
	}

	eveNode = node
	res := m.Run()
	os.Exit(res)  
}
```

## 4. Write your test

Below is an example of test, which check if AppArmor is enabled (specific file on EVE exists). It uses `EveReadFile` function from `evetestkit`

```go
const appArmorStatus = "/sys/module/apparmor/parameters/enabled"
// ...
func TestAppArmorEnabled(t *testing.T) {
	log.Println("TestAppArmorEnabled started")
	defer log.Println("TestAppArmorEnabled finished")

	out, err := eveNode.EveReadFile(appArmorStatus)
	if err != nil {
		t.Fatal(err)
	}

	exits := strings.TrimSpace(string(out))
	if exits != "Y" {
		t.Fatal("AppArmor is not enabled")
	}
```
