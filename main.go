// jsonnet-language-server: A Language Server Protocol server for Jsonnet.
// Copyright (C) 2021 Jack Baldry

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/jdbaldry/go-language-server-protocol/jsonrpc2"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

const (
	name = "jsonnet-language-server"
)

var (
	// Set with `-ldflags="-X 'main.version=<version>'"`
	version = "dev"
)

// printVersion prints version text to the provided writer.
func printVersion(w io.Writer) {
	fmt.Fprintf(w, "%s version %s\n", name, version)
}

// printVersion prints help text to the provided writer.
func printHelp(w io.Writer) {
	printVersion(w)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Options:\n")
	fmt.Fprintf(w, "  -h / --help        Print this help message.\n")
	fmt.Fprintf(w, "  -J / --jpath <dir> Specify an additional library search dir\n")
	fmt.Fprintf(w, "                     (right-most wins).\n")
	fmt.Fprintf(w, "  -v / --version     Print version.\n")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Environment variables:\n")
	fmt.Fprintf(w, "  JSONNET_PATH is a %q separated list of directories\n", filepath.ListSeparator)
	fmt.Fprintf(w, "  added in reverse order before the paths specified by --jpath.\n")
	fmt.Fprintf(w, "  These are equivalent:\n")
	fmt.Fprintf(w, "    JSONNET_PATH=a:b %s -J c -J d\n", name)
	fmt.Fprintf(w, "    JSONNET_PATH=d:c:a:b %s\n", name)
	fmt.Fprintf(w, "    %s -J b -J a -J c -J d\n", name)
}

func main() {
	jpaths := filepath.SplitList(os.Getenv("JSONNET_PATH"))

	for i, arg := range os.Args {
		if arg == "-h" || arg == "--help" {
			printHelp(os.Stdout)
			os.Exit(0)
		} else if arg == "-v" || arg == "--version" {
			printVersion(os.Stdout)
			os.Exit(0)
		} else if arg == "-J" || arg == "--jpath" {
			if i == len(os.Args)-1 {
				fmt.Printf("Expected value for option %s but found none.\n", arg)
				fmt.Println()
				printHelp(os.Stdout)
				os.Exit(1)
			}

			jpaths = append([]string{os.Args[i+1]}, jpaths...)
		}
	}

	log.Println("Starting the language server")

	ctx := context.Background()
	stream := jsonrpc2.NewHeaderStream(stdio{})
	conn := jsonrpc2.NewConn(stream)
	client := protocol.ClientDispatcher(conn)

	s, err := newServer(client, jpaths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	conn.Go(ctx, protocol.Handlers(
		protocol.ServerHandler(s, jsonrpc2.MethodNotFound)))
	<-conn.Done()
	if err := conn.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
