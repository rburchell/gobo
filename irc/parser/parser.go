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

// Package parser provides a simple interface implementing a tolerant IRC
// message parser.
package parser

import "fmt"
import "strings"

// IrcPrefix represents the sender of an IrcMessage.
// A prefix may either be a server, or a user. If the prefix is representing a
// server, the Server member will be a non-empty string representing the
// server name. If the prefix is representing a user, the Nick, User and Host
// fields will be filled (as much as is possible from the given message).
type IrcPrefix struct {
	// The name of the server this prefix represents.
	Server string

	// The nickname of the user this prefix represents.
	Nick string

	// The username of the user this prefix represents.
	User string

	// The hostname of the user this prefix represents.
	Host string
}

// IrcTag represents a message tag.
// A message tag is an optional extension to the IRC protocol, defined in the
// IRCv3.2 specification (http://ircv3.net/specs/core/message-tags-3.2.html).
// Message tags provide additional metadata about the command in question.
type IrcTag struct {
	// For nonstandardised tags, a message tag has a vendor prefix, defining
	// which software vendor is responsible for this tag (for example, znc.in)
	VendorPrefix string

	// The key of the tag (e.g. server-time)
	Key string

	// Message tags may also have an optional value associated with them.
	Value string
}

// String converts an IrcPrefix to its string representation.
func (this *IrcPrefix) String() string {
	if len(this.Server) > 0 {
		return this.Server
	}

	// we have two possible forms that are valid:
	// nick
	// nick!user@host
	if len(this.User) > 0 {
		return fmt.Sprintf("%s!%s@%s", this.Nick, this.User, this.Host)
	}

	return this.Nick
}

// IrcMessage is the primary interface for interaction with the parser. The
// parser is responsible for generating IrcMessage instances after parsing from
// the input provided by the caller.
//
// An IrcMessage instance represents a full instance of an IRC protocol message,
// as defined by RFC1459 (and other optional extensions).
type IrcMessage struct {
	// Tags, if any - see the IRCv3.2 message tags extension.
	Tags []IrcTag

	// The sender of the message. See IrcPrefix for more information.
	Prefix IrcPrefix

	// An uppercased string containing the command.
	Command string

	// A slice of strings providing all the parameters to the command.
	// Note that the final parameter of IRC messages starting with a colon, e.g.
	// PRIVSG foo :bar moo cow) will coalesce the final three words into a
	// single entry in the slice when parsing - such that parameters in
	// this case would be:
	// 0: foo
	// 1: bar moo cow
	Parameters []string
}

// String converts an IrcMessage to its string representation.
func (this *IrcMessage) String() string {
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

// Given a string line, splits by a space delimiter and returns the first word
// in arg, and the rest of the string for further processing.
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

// ParseLine takes the given IRC protocol message in line and processes it.
//
// It returns a usable IrcMessage struct instance.
func ParseLine(line string) *IrcMessage {
	// BUG(w00t): ParseLine does not currently have a way of reporting errors.
	args := []string{}
	command := new(IrcMessage)

	// ircv3 message tags extension
	if strings.HasPrefix(line, "@") {
		var tagstr string
		tagstr, line = splitArg(line)
		tagstr = tagstr[1:len(tagstr)]

		// aaa=bbb;ccc;example.com/ddd=eee
		// split each tag and process seperately
		tags := strings.Split(tagstr, ";")
		for _, tag := range tags {
			eq := strings.Index(tag, "=")
			var key string
			var tagobj IrcTag

			if eq == -1 {
				// no equals sign means this is a tag with a key (and possibly
				// vendor prefix) only, no value.
				key = tag
			} else {
				// if we have an equals sign, we have a value too.
				key = tag[0:eq]

				// the value itself requires some string escaping.
				vstr := tag[eq+1 : len(tag)]
				vstr = strings.Replace(vstr, "\\:", ";", -1)
				vstr = strings.Replace(vstr, "\\s", " ", -1)
				vstr = strings.Replace(vstr, "\\\\", "\\", -1)
				vstr = strings.Replace(vstr, "\\r", "\r", -1)
				vstr = strings.Replace(vstr, "\\n", "\n", -1)
				tagobj.Value = vstr
			}

			// finally, find the vendor prefix - if any.
			slash := strings.Index(key, "/")
			if slash != -1 {
				tagobj.VendorPrefix = key[0:slash]
				tagobj.Key = key[slash+1 : len(key)]
			} else {
				tagobj.Key = key
			}

			// and save the tag.
			command.Tags = append(command.Tags, tagobj)
		}
	}

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
			bang := strings.Index(pfx, "!")
			at := strings.Index(pfx, "@") //  TODO: would LastIndex be faster? maybe not, host is usually long.
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
