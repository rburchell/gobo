/*
 * Copyright (C) 2015 Robin Burchell <robin+git@viroteck.net>
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 *  - Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 *  - Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE AUTHOR AND CONTRIBUTORS ``AS IS'' AND ANY
 * EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 * WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE AUTHOR OR CONTRIBUTORS BE LIABLE FOR ANY
 * DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 * LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 * ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF
 * THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package parser

import "fmt"
import "strings"

type IrcPrefix struct {
	Server string
	Nick   string
	User   string
	Host   string
}

func (this *IrcPrefix) String() string {
	if len(this.Server) > 0 {
		return this.Server
	} else {
		// we have two possible forms that are valid:
		// nick
		// nick!user@host
		if len(this.User) > 0 {
			return fmt.Sprintf("%s!%s@%s", this.Nick, this.User, this.Host)
		} else {
			return this.Nick
		}
	}
}

// :cameron.freenode.net NOTICE * :*** Looking up your hostname...
type IrcCommand struct {
	// cameron.freenode.net
	Prefix IrcPrefix

	// NOTICE
	Command string

	// [0]: *
	// [1]: *** Looking up your hostname...
	Parameters []string
}

func (this *IrcCommand) String() string {
	prefix := ""
	parameters := ""

	if len(this.Prefix.String()) > 0 {
		prefix = fmt.Sprintf(":%s ", this.Prefix.String())
	}

	if len(this.Parameters) > 0 {
		pcount := len(this.Parameters)
		if strings.Contains(this.Parameters[pcount-1], " ") {
			if pcount > 1 {
				parameters = strings.Join(this.Parameters[0:pcount-1], " ")
				parameters = fmt.Sprintf(" %s :%s", parameters, this.Parameters[pcount-1])
			} else {
				parameters = " :" + this.Parameters[pcount-1]
			}
		} else {
			parameters = " "
			parameters += strings.Join(this.Parameters, " ")
		}
	}

	return fmt.Sprintf("%s%s%s", prefix, this.Command, parameters)
}

func splitArg(line string) (arg string, rest string) {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) > 0 {
		arg = parts[0]
	}
	if len(parts) > 1 {
		rest = parts[1]
	}
	return
}

// TODO: communicate errors to caller
func ParseLine(line string) *IrcCommand {
	args := make([]string, 0)
	command := new(IrcCommand)

	if strings.HasPrefix(line, ":") {
		var pfx string
		pfx, line = splitArg(line)
		pfx = pfx[1:len(pfx)]

		// foo!bar@moo is nick!user@host
		// foo is nick
		// foo.bar.moo is a server
		//
		// therefore: if we see a ! OR we see no ., we enter this branch...
		if strings.Contains(pfx, "!") || !strings.Contains(pfx, ".") {
			var bang int = strings.Index(pfx, "!")
			var at int = strings.Index(pfx, "@") //  TODO: would LastIndex be faster? maybe not, host is usually long.
			if bang == -1 && at == -1 {
				command.Prefix.Nick = pfx
			} else if bang == -1 {
				// nick@host? invalid case, we empty the prefix
			} else if at == -1 {
				// nick!user? invalid case, we empty the prefix
			} else if bang > at {
				// nick@user!host or similar => automatically invalid
			} else {
				command.Prefix.Nick = pfx[0:bang]
				command.Prefix.User = pfx[bang+1 : at]
				command.Prefix.Host = pfx[at+1 : len(pfx)]
			}
		} else {
			// message from a server
			command.Prefix.Server = pfx
		}
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
