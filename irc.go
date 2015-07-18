package main

import "fmt"
import "strings"
import "regexp"

// :cameron.freenode.net NOTICE * :*** Looking up your hostname...
type Command struct {
    // cameron.freenode.net
    Prefix string

    // NOTICE
    Command string

    // [0]: *
    // [1]: *** Looking up your hostname...
    Parameters []string
}

func (this *Command) String() string {
    return fmt.Sprintf("%s %s %s", this.Prefix, this.Command, strings.Join(this.Parameters, " "))
}

var (
    spacesExpr = regexp.MustCompile(` +`)
)

func splitArg(line string) (arg string, rest string) {
    parts := spacesExpr.Split(line, 2)
    if len(parts) > 0 {
        arg = parts[0]
    }
    if len(parts) > 1 {
        rest = parts[1]
    }
    return
}

func ParseLine(line string) *Command {
    args := make([]string, 0)
    command := new(Command)

    if strings.HasPrefix(line, ":") {
        command.Prefix, line = splitArg(line)
        command.Prefix = command.Prefix[1:len(command.Prefix)]
    }
    arg, line := splitArg(line)
    command.Command = strings.ToUpper(arg)
    for len(line) > 0 {
        if strings.HasPrefix(line, ":") {
            args = append(args, line[len(":"):])
            break
        }
        arg, line = splitArg(line)
        args = append(args, arg)
    }
    command.Parameters = args
    return command
}

