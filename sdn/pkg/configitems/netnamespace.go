package configitems

import "os/exec"

// Symbolic name for the main network namespace (where SDN agent operates).
const mainNsName = "<main>"

func normNetNsName(name string) string {
	if name == "" {
		name = mainNsName
	}
	return name
}

func isMainNetNs(name string) bool {
	return name == mainNsName || name == ""
}

func namespacedCmd(netNs string, cmd string, args ...string) *exec.Cmd {
	if isMainNetNs(netNs) {
		return exec.Command(cmd, args...)
	}
	var newArgs []string
	newArgs = append(newArgs, "netns", "exec", normNetNsName(netNs))
	newArgs = append(newArgs, cmd)
	newArgs = append(newArgs, args...)
	return exec.Command("ip", newArgs...)
}