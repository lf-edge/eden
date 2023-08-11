// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testscript

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eden/tests/escript/go-internal/internal/textutil"
	"github.com/lf-edge/eden/tests/escript/go-internal/txtar"
)

// scriptCmds are the script command implementations.
// Keep list and the implementations below sorted by name.
//
// NOTE: If you make changes here, update doc.go.
var scriptCmds = map[string]func(*TestScript, bool, []string){
	"arg":     (*TestScript).cmdArg,
	"cd":      (*TestScript).cmdCd,
	"chmod":   (*TestScript).cmdChmod,
	"cmp":     (*TestScript).cmdCmp,
	"cmpenv":  (*TestScript).cmdCmpenv,
	"cp":      (*TestScript).cmdCp,
	"eden":    (*TestScript).cmdEden,
	"env":     (*TestScript).cmdEnv,
	"source":  (*TestScript).cmdSource,
	"exec":    (*TestScript).cmdExec,
	"exists":  (*TestScript).cmdExists,
	"grep":    (*TestScript).cmdGrep,
	"message": (*TestScript).cmdMsg,
	"mkdir":   (*TestScript).cmdMkdir,
	"rm":      (*TestScript).cmdRm,
	"unquote": (*TestScript).cmdUnquote,
	"skip":    (*TestScript).cmdSkip,
	"stdin":   (*TestScript).cmdStdin,
	"stderr":  (*TestScript).cmdStderr,
	"stdout":  (*TestScript).cmdStdout,
	"stop":    (*TestScript).cmdStop,
	"symlink": (*TestScript).cmdSymlink,
	"test":    (*TestScript).cmdTest,
	"wait":    (*TestScript).cmdWait,
}

var timewait time.Duration
var backgroundSpecifier = regexp.MustCompile(`^&(\w+&)?$`)

// cd changes to a different directory.
func (ts *TestScript) cmdCd(neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! cd")
	}
	if len(args) != 1 {
		ts.Fatalf("usage: cd dir")
	}

	dir := args[0]
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(ts.cd, dir)
	}
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		ts.Fatalf("directory %s does not exist", dir)
	}
	ts.Check(err)
	if !info.IsDir() {
		ts.Fatalf("%s is not a directory", dir)
	}
	ts.cd = dir
	ts.Logf("%s\n", ts.cd)
}

func (ts *TestScript) cmdChmod(neg bool, args []string) {
	if len(args) != 2 {
		ts.Fatalf("usage: chmod mode file")
	}
	mode, err := strconv.ParseInt(args[0], 8, 32)
	if err != nil {
		ts.Fatalf("bad file mode %q: %v", args[0], err)
	}
	if mode > 0777 {
		ts.Fatalf("unsupported file mode %.3o", mode)
	}
	err = os.Chmod(ts.MkAbs(args[1]), os.FileMode(mode))
	if neg {
		if err == nil {
			ts.Fatalf("unexpected chmod success")
		}
		return
	}
	if err != nil {
		ts.Fatalf("unexpected chmod failure: %v", err)
	}
}

// cmp compares two files.
func (ts *TestScript) cmdCmp(neg bool, args []string) {
	if len(args) != 2 {
		ts.Fatalf("usage: cmp file1 file2")
	}

	res := ts.doCmdCmp(args, false)
	if neg {
		if res {
			ts.Fatalf("unexpected cmp success")
		}
		return
	}
	if !res {
		ts.Fatalf("%s and %s differ", args[0], args[1])
	}
}

// cmpenv compares two files with environment variable substitution.
func (ts *TestScript) cmdCmpenv(neg bool, args []string) {
	if len(args) != 2 {
		ts.Fatalf("usage: cmpenv file1 file2")
	}
	res := ts.doCmdCmp(args, true)
	if neg {
		if res {
			ts.Fatalf("unexpected cmpenv success")
		}
		return
	}
	if !res {
		ts.Fatalf("%s and %s differ", args[0], args[1])
	}
}

func (ts *TestScript) doCmdCmp(args []string, env bool) bool {
	name1, name2 := args[0], args[1]
	text1 := ts.ReadFile(name1)

	absName2 := ts.MkAbs(name2)
	data, err := os.ReadFile(absName2)
	ts.Check(err)
	text2 := string(data)
	if env {
		text2 = ts.expand(text2)
	}
	if text1 == text2 {
		return true
	}
	if ts.params.UpdateScripts && !env && (args[0] == "stdout" || args[0] == "stderr") {
		if scriptFile, ok := ts.scriptFiles[absName2]; ok {
			ts.scriptUpdates[scriptFile] = text1
			return true
		}
		// The file being compared against isn't in the txtar archive, so don't
		// update the script.
	}

	ts.Logf("[diff -%s +%s]\n%s\n", name1, name2, textutil.Diff(text1, text2))
	return false
}

// cp copies files, maybe eventually directories.
func (ts *TestScript) cmdCp(neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! cp")
	}
	if len(args) < 2 {
		ts.Fatalf("usage: cp src... dst")
	}

	dst := ts.MkAbs(args[len(args)-1])
	info, err := os.Stat(dst)
	dstDir := err == nil && info.IsDir()
	if len(args) > 2 && !dstDir {
		ts.Fatalf("cp: destination %s is not a directory", dst)
	}

	for _, arg := range args[:len(args)-1] {
		var (
			src  string
			data []byte
			mode os.FileMode
		)
		switch arg {
		case "stdout":
			src = arg
			data = []byte(ts.stdout)
			mode = 0666
		case "stderr":
			src = arg
			data = []byte(ts.stderr)
			mode = 0666
		default:
			src = ts.MkAbs(arg)
			info, err := os.Stat(src)
			ts.Check(err)
			mode = info.Mode() & 0777
			data, err = os.ReadFile(src)
			ts.Check(err)
		}
		targ := dst
		if dstDir {
			targ = filepath.Join(dst, filepath.Base(src))
		}
		ts.Check(os.WriteFile(targ, data, mode))
	}
}

func (ts *TestScript) backgroundHelper(neg bool, bgName, command string, args []string) error {
	cmd, _, stdoutBuf, stderrBuf, err := ts.execBackground(command, args...)
	if err != nil {
		return err
	}
	wait := make(chan struct{})
	go func() {
		err = ctxWait(ts.ctxt, cmd)
		close(wait)
		if err == context.Canceled {
			return
		}
		if err == nil && neg {
			outString := stdoutBuf.String()
			errString := stderrBuf.String()
			if outString != "" {
				_, _ = fmt.Fprintf(&ts.log, "[stdout]\n%s", outString)
			}
			if errString != "" {
				_, _ = fmt.Fprintf(&ts.log, "[stderr]\n%s", errString)
			}
			ts.Fatalf("unexpected command success")
		}
	}()
	ts.background = append(ts.background, backgroundCmd{bgName, cmd, wait, neg})
	ts.stdout, ts.stderr = "", ""
	return nil
}

func (ts *TestScript) findBackground(bgName string) *backgroundCmd {
	if bgName == "" {
		return nil
	}
	for i := range ts.background {
		bg := &ts.background[i]
		if bg.name == bgName {
			return bg
		}
	}
	return nil
}

// eden execute EDEN's commands.
func (ts *TestScript) cmdEden(neg bool, args []string) {
	if len(args) < 1 || (len(args) == 1 && args[0] == "&") {
		ts.Fatalf("usage: eden [-t timewait] command [args...] [&]")
	}

	vars, err := utils.InitVars()
	if err != nil {
		ts.Fatalf("error reading config: %s\n", err)
	}

	edenProg := utils.ResolveAbsPath(vars.EdenBinDir + "/" + vars.EdenProg)

	_, err = exec.LookPath(edenProg)
	if err != nil {
		ts.Fatalf("can't find 'eden' executable: %s\n", err)
	}

	// timewait
	if len(args) > 0 && args[0] == "-t" {
		timewait, err = time.ParseDuration(args[1])
		if err != nil {
			ts.Fatalf("Incorrect time format in 'eden': %s\n", err)
		}
		args = args[2:]
	} else {
		timewait = 0
	}
	fmt.Printf("edenProg: %s timewait: %s\n", edenProg, timewait)

	if len(args) > 0 && backgroundSpecifier.MatchString(args[len(args)-1]) {
		bgName := strings.TrimSuffix(strings.TrimPrefix(args[len(args)-1], "&"), "&")
		if ts.findBackground(bgName) != nil {
			ts.Fatalf("duplicate background process name %q", bgName)
		}
		err = ts.backgroundHelper(neg, bgName, edenProg, args[:len(args)-1])
	} else {
		ts.stdout, ts.stderr, err = ts.exec(edenProg, args...)
		if ts.stdout != "" {
			fmt.Fprintf(&ts.log, "[stdout]\n%s", ts.stdout)
		}
		if ts.stderr != "" {
			fmt.Fprintf(&ts.log, "[stderr]\n%s", ts.stderr)
		}
		if err == nil && neg {
			ts.Fatalf("unexpected command success")
		}
	}

	if err != nil {
		fmt.Fprintf(&ts.log, "[%v]\n", err)
		if ts.ctxt.Err() != nil {
			ts.Fatalf("test interrupted while running command")
		} else if !neg {
			ts.Fatalf("command failure")
		}
	}
}

// eden execute EDEN's test commands.
func (ts *TestScript) cmdTest(neg bool, args []string) {
	if len(args) < 1 || (len(args) == 1 && args[0] == "&") {
		ts.Fatalf("usage: test [-t timewait] program [args...] [&]")
	}

	vars, err := utils.InitVars()
	if err != nil {
		ts.Fatalf("error reading config: %s\n", err)
	}

	// timewait
	if len(args) > 0 && args[0] == "-t" {
		timewait, err = time.ParseDuration(args[1])
		if err != nil {
			ts.Fatalf("Incorrect time format in 'test': %s\n", err)
		}
		args = args[2:]
	} else {
		timewait = 0
	}

	testProg := utils.ResolveAbsPath(vars.EdenBinDir + "/" + args[0])
	args = args[1:]

	fmt.Printf("testProg: %s timewait: %s\n", testProg, timewait)

	_, err = exec.LookPath(testProg)
	if err != nil {
		ts.Fatalf("can't find eden test executable: %s\n", err)
	} else {
		ts.Logf("testProg: %s\n", testProg)
	}

	if len(args) > 0 && backgroundSpecifier.MatchString(args[len(args)-1]) {
		bgName := strings.TrimSuffix(strings.TrimPrefix(args[len(args)-1], "&"), "&")
		if ts.findBackground(bgName) != nil {
			ts.Fatalf("duplicate background process name %q", bgName)
		}
		err = ts.backgroundHelper(neg, bgName, testProg, args[:len(args)-1])
	} else {
		ts.stdout, ts.stderr, err = ts.exec(testProg, args...)
		if ts.stdout != "" {
			fmt.Fprintf(&ts.log, "[stdout]\n%s", ts.stdout)
		}
		if ts.stderr != "" {
			fmt.Fprintf(&ts.log, "[stderr]\n%s", ts.stderr)
		}
		if err == nil && neg {
			ts.Fatalf("unexpected command success")
		}
	}

	if err != nil {
		fmt.Fprintf(&ts.log, "[%v]\n", err)
		if ts.ctxt.Err() != nil {
			ts.Fatalf("test interrupted while running command")
		} else if !neg {
			ts.Fatalf("command failure")
		}
	}
}

// env displays or adds to the environment.
func (ts *TestScript) cmdEnv(neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! env")
	}
	if len(args) == 0 {
		printed := make(map[string]bool) // env list can have duplicates; only print effective value (from envMap) once
		for _, kv := range ts.env {
			k := envvarname(kv[:strings.Index(kv, "=")])
			if !printed[k] {
				printed[k] = true
				ts.Logf("%s=%s\n", k, ts.envMap[k])
			}
		}
		return
	}
	for _, env := range args {
		i := strings.Index(env, "=")
		if i < 0 {
			// Display value instead of setting it.
			ts.Logf("%s=%s\n", env, ts.Getenv(env))
			continue
		}
		ts.Setenv(env[:i], env[i+1:])
	}
}

// cmdArg process args and set value into environment variable.
func (ts *TestScript) cmdArg(neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! env")
	}
	if len(args) != 2 {
		ts.Fatalf("usage: arg argument env_variable")
	}
	if val, ok := ts.params.Flags[args[0]]; !ok {
		ts.Logf("argument %s not found in passed argument", args[0])
	} else {
		ts.Setenv(args[1], val)
	}
}

// source parse variables from file to the environment.
func (ts *TestScript) cmdSource(neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! source")
	}
	if len(args) == 0 {
		ts.Fatalf("No file defined to parse variables")
	}
	if len(args) < 1 {
		ts.Fatalf("usage: source file...")
	}
	for _, arg := range args {
		currentFile := ts.MkAbs(arg)
		localViper := viper.New()
		localViper.SetConfigType("env") //we support only env type
		localViper.SetConfigFile(currentFile)
		if err := localViper.ReadInConfig(); err != nil {
			ts.Fatalf("Cannot process file %s: %v", currentFile, err)
		}
		for _, el := range localViper.AllKeys() {
			ts.Setenv(el, fmt.Sprint(localViper.Get(el)))
		}
	}
}

// exec runs the given command.
func (ts *TestScript) cmdExec(neg bool, args []string) {
	if len(args) < 1 || (len(args) == 1 && args[0] == "&") {
		ts.Fatalf("usage: exec [-t timewait] program [args...] [&]")
	}

	var err error
	if len(args) > 0 && args[0] == "-t" {
		timewait, err = time.ParseDuration(args[1])
		if err != nil {
			ts.Fatalf("Incorrect time format in 'exec': %s\n", err)
		}
		args = args[2:]
	} else {
		timewait = 0
	}
	fmt.Printf("exec timewait: %s\n", timewait)

	if len(args) > 0 && backgroundSpecifier.MatchString(args[len(args)-1]) {
		bgName := strings.TrimSuffix(strings.TrimPrefix(args[len(args)-1], "&"), "&")
		if ts.findBackground(bgName) != nil {
			ts.Fatalf("duplicate background process name %q", bgName)
		}
		err = ts.backgroundHelper(neg, bgName, args[0], args[1:len(args)-1])
	} else {
		ts.stdout, ts.stderr, err = ts.exec(args[0], args[1:]...)
		if ts.stdout != "" {
			fmt.Fprintf(&ts.log, "[stdout]\n%s", ts.stdout)
		}
		if ts.stderr != "" {
			fmt.Fprintf(&ts.log, "[stderr]\n%s", ts.stderr)
		}
		if err == nil && neg {
			ts.Fatalf("unexpected command success")
		}
	}

	if err != nil {
		fmt.Fprintf(&ts.log, "[%v]\n", err)
		if ts.ctxt.Err() != nil {
			ts.Fatalf("test interrupted while running command")
		} else if !neg {
			t := ts.ctxt.Value("kind")
			if t != nil {
				ts.Logf("exec timewait: %v\n", t)
			} else {
				ts.Fatalf("command failure")
			}
		}
	}
}

// exists checks that the list of files exists.
func (ts *TestScript) cmdExists(neg bool, args []string) {
	var readonly bool
	if len(args) > 0 && args[0] == "-readonly" {
		readonly = true
		args = args[1:]
	}
	if len(args) == 0 {
		ts.Fatalf("usage: exists [-readonly] file...")
	}

	for _, file := range args {
		file = ts.MkAbs(file)
		info, err := os.Stat(file)
		if err == nil && neg {
			what := "file"
			if info.IsDir() {
				what = "directory"
			}
			ts.Fatalf("%s %s unexpectedly exists", what, file)
		}
		if err != nil && !neg {
			ts.Fatalf("%s does not exist", file)
		}
		if err == nil && !neg && readonly && info.Mode()&0222 != 0 {
			ts.Fatalf("%s exists but is writable", file)
		}
	}
}

// stop stops execution of the test (marking it passed).
func (ts *TestScript) cmdMsg(neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! stop")
	}
	if len(args) == 1 {
		ts.Logf("message: %s\n", args[0])
	} else {
		ts.Fatalf("usage: message [msg]")
	}
}

// mkdir creates directories.
func (ts *TestScript) cmdMkdir(neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! mkdir")
	}
	if len(args) < 1 {
		ts.Fatalf("usage: mkdir dir...")
	}
	for _, arg := range args {
		ts.Check(os.MkdirAll(ts.MkAbs(arg), 0777))
	}
}

// unquote unquotes files.
func (ts *TestScript) cmdUnquote(neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! unquote")
	}
	for _, arg := range args {
		file := ts.MkAbs(arg)
		data, err := os.ReadFile(file)
		ts.Check(err)
		data, err = txtar.Unquote(data)
		ts.Check(err)
		err = os.WriteFile(file, data, 0666)
		ts.Check(err)
	}
}

// rm removes files or directories.
func (ts *TestScript) cmdRm(neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! rm")
	}
	if len(args) < 1 {
		ts.Fatalf("usage: rm file...")
	}
	for _, arg := range args {
		file := ts.MkAbs(arg)
		_ = removeAll(file)          // does chmod and then attempts rm
		ts.Check(os.RemoveAll(file)) // report error
	}
}

// skip marks the test skipped.
func (ts *TestScript) cmdSkip(neg bool, args []string) {
	if len(args) > 1 {
		ts.Fatalf("usage: skip [msg]")
	}
	if neg {
		ts.Fatalf("unsupported: ! skip")
	}

	// Before we mark the test as skipped, shut down any background processes and
	// make sure they have returned the correct status.
	for _, bg := range ts.background {
		interruptProcess(bg.cmd.Process)
	}
	ts.cmdWait(false, nil)

	if len(args) == 1 {
		ts.t.Skip(args[0])
	}
	ts.t.Skip()
}

func (ts *TestScript) cmdStdin(neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! stdin")
	}
	if len(args) != 1 {
		ts.Fatalf("usage: stdin filename")
	}
	data, err := os.ReadFile(ts.MkAbs(args[0]))
	ts.Check(err)
	ts.stdin = string(data)
}

// stdout checks that the last go command standard output matches a regexp.
func (ts *TestScript) cmdStdout(neg bool, args []string) {
	scriptMatch(ts, neg, args, ts.stdout, "stdout")
}

// stderr checks that the last go command standard output matches a regexp.
func (ts *TestScript) cmdStderr(neg bool, args []string) {
	scriptMatch(ts, neg, args, ts.stderr, "stderr")
}

// grep checks that file content matches a regexp.
// Like stdout/stderr and unlike Unix grep, it accepts Go regexp syntax.
func (ts *TestScript) cmdGrep(neg bool, args []string) {
	scriptMatch(ts, neg, args, "", "grep")
}

// stop stops execution of the test (marking it passed).
func (ts *TestScript) cmdStop(neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! stop")
	}
	if len(args) > 1 {
		ts.Fatalf("usage: stop [msg]")
	}
	if len(args) == 1 {
		ts.Logf("stop: %s\n", args[0])
	} else {
		ts.Logf("stop\n")
	}
	ts.stopped = true
}

// symlink creates a symbolic link.
func (ts *TestScript) cmdSymlink(neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! symlink")
	}
	if len(args) != 3 || args[1] != "->" {
		ts.Fatalf("usage: symlink file -> target")
	}
	// Note that the link target args[2] is not interpreted with MkAbs:
	// it will be interpreted relative to the directory file is in.
	ts.Check(os.Symlink(args[2], ts.MkAbs(args[0])))
}

// Tait waits for background commands to exit, setting stderr and stdout to their result.
func (ts *TestScript) cmdWait(neg bool, args []string) {
	if len(args) > 1 {
		ts.Fatalf("usage: wait [name]")
	}
	if neg {
		ts.Fatalf("unsupported: ! wait")
	}
	if len(args) > 0 {
		ts.waitBackgroundOne(args[0])
	} else {
		ts.waitBackground(true)
	}
}

func (ts *TestScript) waitBackgroundOne(bgName string) {
	bg := ts.findBackground(bgName)
	if bg == nil {
		ts.Fatalf("unknown background process %q", bgName)
		return
	}
	<-bg.wait
	ts.stdout = bg.cmd.Stdout.(*strings.Builder).String()
	ts.stderr = bg.cmd.Stderr.(*strings.Builder).String()
	if ts.stdout != "" {
		fmt.Fprintf(&ts.log, "[stdout]\n%s", ts.stdout)
	}
	if ts.stderr != "" {
		fmt.Fprintf(&ts.log, "[stderr]\n%s", ts.stderr)
	}
	// Note: ignore bg.neg, which only takes effect on the non-specific
	// wait command.
	if bg.cmd.ProcessState.Success() {
		if bg.neg {
			ts.Fatalf("unexpected command success")
		}
	} else {
		if ts.ctxt.Err() != nil {
			ts.Fatalf("test timed out while running command")
		} else if !bg.neg {
			ts.Fatalf("command failure")
		}
	}
	// Remove this process from the list of running background processes.
	for i := range ts.background {
		if bg == &ts.background[i] {
			ts.background = append(ts.background[:i], ts.background[i+1:]...)
			break
		}
	}
}

func (ts *TestScript) waitBackground(checkStatus bool) {
	var stdouts, stderrs []string
	for _, bg := range ts.background {
		<-bg.wait
		args := append([]string{filepath.Base(bg.cmd.Args[0])}, bg.cmd.Args[1:]...)
		fmt.Fprintf(&ts.log, "[background] %s: %v\n", strings.Join(args, " "), bg.cmd.ProcessState)
		cmdStdout := bg.cmd.Stdout.(*strings.Builder).String()
		if cmdStdout != "" {
			fmt.Fprintf(&ts.log, "[stdout]\n%s", cmdStdout)
			stdouts = append(stdouts, cmdStdout)
		}
		cmdStderr := bg.cmd.Stderr.(*strings.Builder).String()
		if cmdStderr != "" {
			fmt.Fprintf(&ts.log, "[stderr]\n%s", cmdStderr)
			stderrs = append(stderrs, cmdStderr)
		}
		if !checkStatus {
			continue
		}
		if bg.cmd.ProcessState.Success() {
			if bg.neg {
				ts.Fatalf("unexpected command success")
			}
		} else {
			if ts.ctxt.Err() != nil {
				ts.Fatalf("test timed out while running command")
			} else if !bg.neg {
				ts.Fatalf("command failure")
			}
		}
	}
	ts.stdout = strings.Join(stdouts, "")
	ts.stderr = strings.Join(stderrs, "")
	ts.background = nil
}

// scriptMatch implements both stdout and stderr.
func scriptMatch(ts *TestScript, neg bool, args []string, text, name string) {
	n := 0
	if len(args) >= 1 && strings.HasPrefix(args[0], "-count=") {
		if neg {
			ts.Fatalf("cannot use -count= with negated match")
		}
		var err error
		n, err = strconv.Atoi(args[0][len("-count="):])
		if err != nil {
			ts.Fatalf("bad -count=: %v", err)
		}
		if n < 1 {
			ts.Fatalf("bad -count=: must be at least 1")
		}
		args = args[1:]
	}

	extraUsage := ""
	want := 1
	if name == "grep" {
		extraUsage = " file"
		want = 2
	}
	if len(args) != want {
		ts.Fatalf("usage: %s [-count=N] 'pattern'%s", name, extraUsage)
	}

	pattern := args[0]
	re, err := regexp.Compile(`(?m)` + pattern)
	ts.Check(err)

	isGrep := name == "grep"
	if isGrep {
		name = args[1] // for error messages
		data, err := os.ReadFile(ts.MkAbs(args[1]))
		ts.Check(err)
		text = string(data)
	}

	if neg {
		if re.MatchString(text) {
			if isGrep {
				ts.Logf("[%s]\n%s\n", name, text)
			}
			ts.Fatalf("unexpected match for %#q found in %s: %s", pattern, name, re.FindString(text))
		}
	} else {
		if !re.MatchString(text) {
			if isGrep {
				ts.Logf("[%s]\n%s\n", name, text)
			}
			ts.Fatalf("no match for %#q found in %s", pattern, name)
		}
		if n > 0 {
			count := len(re.FindAllString(text, -1))
			if count != n {
				if isGrep {
					ts.Logf("[%s]\n%s\n", name, text)
				}
				ts.Fatalf("have %d matches for %#q, want %d", count, pattern, n)
			}
		}
	}
}
