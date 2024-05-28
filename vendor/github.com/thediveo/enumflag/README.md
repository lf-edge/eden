# CLI Enumeration Flags

[![GoDoc](https://godoc.org/github.com/thediveo/enumflag?status.svg)](http://godoc.org/github.com/thediveo/enumflag)
[![GitHub](https://img.shields.io/github/license/thediveo/enumflag)](https://img.shields.io/github/license/thediveo/enumflag)
![build and test](https://github.com/thediveo/enumflag/workflows/build%20and%20test/badge.svg?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thediveo/enumflag)](https://goreportcard.com/report/github.com/thediveo/enumflag)
![Coverage](https://img.shields.io/badge/coverage-135.72%25-darkcyan)

`enumflag` is a Golang package which supplements the Golang CLI flag packages
[spf13/cobra](https://github.com/spf13/cobra) and
[spf13/pflag](https://github.com/spf13/pflag) with enumeration flags, including
support for enumeration slices.

For instance, users can specify enum flags as `--mode=foo` or `--mode=bar`,
where `foo` and `bar` are valid enumeration values. Other values which are not
part of the set of allowed enumeration values cannot be set and raise CLI flag
errors. In case of an enumeration _slice_ flag users can specify multiple
enumeration values either with a single flag `--mode=foo,bar` or multiple flag
calls, such as `--mode=foo --mode=bar`.

Application programmers then simply deal with enumeration values in form of
uints (or ints), liberated from parsing strings and validating enumeration
flags.

## Alternatives

In case you are just interested in string-based one-of-a-set flags, then the
following packages offer you a minimalist approach:

- [hashicorp/packer/helper/enumflag](https://godoc.org/github.com/hashicorp/packer/helper/enumflag)
  really is a reduced-to-the-max version without any whistles and bells.

- [creachadair/goflags/enumflag](https://godoc.org/github.com/creachadair/goflags/enumflag)
  has a similar, but slightly more elaborate API with additional "indices" for
  enumeration values.

But if you instead want to handle one-of-a-set flags as properly typed
enumerations instead of strings, or if you need (multiple-of-a-set) slice
support, then please read on.

## How To Use

- [start with your own enum types](#start-with-your-own-enum-types),
- [use existing enum types and non-zero defaults](#use-existing-enum-types),
- [CLI flag with default](#cli-flag-with-default),
- [slice of enums](#slice-of-enums).

### Start With Your Own Enum Types

Without further ado, here's how to define and use enum flags in your own
applications...

```go
import (
    "fmt"

    "github.com/spf13/cobra"
    "github.com/thediveo/enumflag"
)

// ① Define your new enum flag type. It can be derived from enumflag.Flag, but
// it doesn't need to be as long as it is compatible with enumflag.Flag, so
// either an int or uint.
type FooMode enumflag.Flag

// ② Define the enumeration values for FooMode.
const (
    Foo FooMode = iota
    Bar
)

// ③ Map enumeration values to their textual representations (value
// identifiers).
var FooModeIds = map[FooMode][]string{
    Foo: {"foo"},
    Bar: {"bar"},
}

// ④ Now use the FooMode enum flag. If you want a non-zero default, then simply
// set it here, such as in "foomode = Bar".
var foomode FooMode

func main() {
    rootCmd := &cobra.Command{
        Run: func(cmd *cobra.Command, _ []string) {
            fmt.Printf("mode is: %d=%q\n",
                foomode,
                cmd.PersistentFlags().Lookup("mode").Value.String())
        },
    }
    // ⑤ Define the CLI flag parameters for your wrapped enum flag.
    rootCmd.PersistentFlags().VarP(
        enumflag.New(&foomode, "mode", FooModeIds, enumflag.EnumCaseInsensitive),
        "mode", "m",
        "foos the output; can be 'foo' or 'bar'")

    rootCmd.SetArgs([]string{"--mode", "bAr"})
    _ = rootCmd.Execute()
}
```

The boilerplate pattern is always the same:

1. Define your own new enumeration type, such as `type FooMode enumflag.Flag`.
2. Define the constants in your enumeration.
3. Define the mapping of the constants onto enum values (textual
   representations).
4. Somewhere, declare a flag variable of your enum flag type.
   - If you want to use a non-zero default enum value, just go ahead and set
     it: `var foomode = Bar`. It will be used correctly.
5. Wire up your flag variable to its flag long and short names, et cetera.

### Use Existing Enum Types

A typical example might be your application using a 3rd party logging package
and you want to offer a `-v` log level CLI flag. Here, we use the existing 3rd
party enum values and set a non-zero default for our logging CLI flag.

Considering the boiler plate shown above, we can now leave out steps ① and ②,
because these definitions come from a 3rd party package. We only need to
supply the textual enum names as ③.

```go
import (
    "fmt"
    "os"

    log "github.com/sirupsen/logrus"
    "github.com/spf13/cobra"
    "github.com/thediveo/enumflag"
)

func main() {
    // ①+② skip "define your own enum flag type" and enumeration values, as we
    // already have a 3rd party one.

    // ③ Map 3rd party enumeration values to their textual representations
    var LoglevelIds = map[log.Level][]string{
        log.TraceLevel: {"trace"},
        log.DebugLevel: {"debug"},
        log.InfoLevel:  {"info"},
        log.WarnLevel:  {"warning", "warn"},
        log.ErrorLevel: {"error"},
        log.FatalLevel: {"fatal"},
        log.PanicLevel: {"panic"},
    }

    // ④ Define your enum flag value and set the your logging default value.
    var loglevel log.Level = log.WarnLevel

    rootCmd := &cobra.Command{
        Run: func(cmd *cobra.Command, _ []string) {
            fmt.Printf("logging level is: %d=%q\n",
                loglevel,
                cmd.PersistentFlags().Lookup("log").Value.String())
        },
    }

    // ⑤ Define the CLI flag parameters for your wrapped enum flag.
    rootCmd.PersistentFlags().Var(
        enumflag.New(&loglevel, "log", LoglevelIds, enumflag.EnumCaseInsensitive),
        "log",
        "sets logging level; can be 'trace', 'debug', 'info', 'warn', 'error', 'fatal', 'panic'")

    // Defaults to what we set above: warn level.
    _ = rootCmd.Execute()

    // User specifies a specific level, such as log level. 
    rootCmd.SetArgs([]string{"--log", "debug"})
    _ = rootCmd.Execute()
}
```

### CLI Flag With Default

Sometimes you might want a CLI enum flag to have a default value when the user
specifies the CLI flag, but not its value. A good example is the `--color`
flag of the `ls` command:

- if just specified as `--color` without a value, it
will default to the value of `auto`;
- otherwise, as specific value can be given, such as
  - `--color=always`,
  - `--color=never`,
  - or even `--color=auto`.

In such situations, use spf13/pflags's
[`NoOptDefVal`](https://godoc.org/github.com/spf13/pflag#Flag) to set the
flag's default value *as text*, if the flag is on the command line without any
options.

The gist here is as follows, please see also
[colormode.go](https://github.com/TheDiveO/lxkns/blob/master/cmd/internal/pkg/style/colormode.go)
from my [lxkns](https://github.com/TheDiveO/lxkns) Linux namespaces discovery
project:

```go
rootCmd.PersistentFlags().VarP(
    enumflag.New(&colorize, "color", colorModeIds, enumflag.EnumCaseSensitive),
    "color", "c",
    "colorize the output; can be 'always' (default if omitted), 'auto',\n"+
        "or 'never'")
rootCmd.PersistentFlags().Lookup("color").NoOptDefVal = "always"
```

### Slice of Enums

For a slice of enumerations, simply declare your variable to be a slice of your
enumeration type and then use `enumflag.NewSlice(...)` instead of
`enumflag.New(...)`.

```go
import (
    "fmt"

    "github.com/spf13/cobra"
    "github.com/thediveo/enumflag"
)

// ① Define your new enum flag type. It can be derived from enumflag.Flag, but
// it doesn't need to be as long as it is compatible with enumflag.Flag, so
// either an int or uint.
type MooMode enumflag.Flag

// ② Define the enumeration values for FooMode.
const (
    Moo MooMode = (iota + 1) * 111
    Møø
    Mimimi
)

// ③ Map enumeration values to their textual representations (value
// identifiers).
var MooModeIds = map[MooMode][]string{
    Moo:    {"moo"},
    Møø:    {"møø"},
    Mimimi: {"mimimi"},
}

// User-defined enum flag types should be derived from "enumflag.Flag"; however
// this is not strictly necessary as long as they can be converted into the
// "enumflag.Flag" type. Actually, "enumflag.Flag" is just a fancy name for an
// "uint". In order to use such user-defined enum flags as flag slices, simply
// wrap them using enumflag.NewSlice.
func Example_slice() {
    // ④ Define your enum slice flag value.
    var moomode []MooMode
    rootCmd := &cobra.Command{
        Run: func(cmd *cobra.Command, _ []string) {
            fmt.Printf("mode is: %d=%q\n",
                moomode,
                cmd.PersistentFlags().Lookup("mode").Value.String())
        },
    }
    // ⑤ Define the CLI flag parameters for your wrapped enumm slice flag.
    rootCmd.PersistentFlags().VarP(
        enumflag.NewSlice(&moomode, "mode", MooModeIds, enumflag.EnumCaseInsensitive),
        "mode", "m",
        "can be any combination of 'moo', 'møø', 'mimimi'")

    rootCmd.SetArgs([]string{"--mode", "Moo,møø"})
    _ = rootCmd.Execute()
}
```

## Copyright and License

`lxkns` is Copyright 2020 Harald Albrecht, and licensed under the Apache
License, Version 2.0.
