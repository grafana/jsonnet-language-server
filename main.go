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
	"github.com/jdbaldry/jsonnet-language-server/server"
	"github.com/jdbaldry/jsonnet-language-server/utils"
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
	fmt.Fprintf(w, `
Options:
  -h / --help        Print this help message.
  -J / --jpath <dir> Specify an additional library search dir
                     (right-most wins).
  -v / --version     Print version.

Environment variables:
  JSONNET_PATH is a %[2]q separated list of directories
  added in reverse order before the paths specified by --jpath
  These are equivalent:
    JSONNET_PATH=a%[2]cb %[1]s -J c -J d
    JSONNET_PATH=d%[2]cc%[2]ca%[2]cb %[1]s\n
    %[1]s -J b -J a -J c -J d
`, name, filepath.ListSeparator)
}

func main() {
	jpaths := filepath.SplitList(os.Getenv("JSONNET_PATH"))
	tankaMode := false

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
		} else if arg == "-t" || arg == "--tanka" {
			tankaMode = true
		}
	}

	if len(jpaths) > 0 && tankaMode {
		fmt.Println("Cannot set JPATH and use tanka mode at the same time")
		os.Exit(1)
	}

	log.Println("Starting the language server")

	ctx := context.Background()
	stream := jsonrpc2.NewHeaderStream(utils.Stdio{})
	conn := jsonrpc2.NewConn(stream)
	client := protocol.ClientDispatcher(conn)

	s := server.NewServer(name, version, client)
	if err := s.Init(); err != nil {
		log.Fatal(err)
	}
	if tankaMode {
		s = s.WithTankaVM()
	} else {
		s = s.WithStaticVM(jpaths)
	}

	conn.Go(ctx, protocol.Handlers(
		protocol.ServerHandler(s, jsonrpc2.MethodNotFound)))
	<-conn.Done()
	if err := conn.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
