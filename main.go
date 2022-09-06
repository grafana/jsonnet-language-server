package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/go-jsonnet/formatter"
	"github.com/grafana/jsonnet-language-server/pkg/server"
	"github.com/grafana/jsonnet-language-server/pkg/utils"
	"github.com/jdbaldry/go-language-server-protocol/jsonrpc2"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	log "github.com/sirupsen/logrus"
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
  -t / --tanka       Create the jsonnet VM with Tanka (finds jpath automatically).
  -l / --log-level   Set the log level (default: info).
  --eval-diags       Try to evaluate files to find errors and warnings.
  --lint             Enable linting.
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
	config := server.Configuration{
		JPaths:            filepath.SplitList(os.Getenv("JSONNET_PATH")),
		FormattingOptions: formatter.DefaultOptions(),
	}
	log.SetLevel(log.InfoLevel)

	for i, arg := range os.Args {
		switch arg {
		case "-h", "--help":
			printHelp(os.Stdout)
			os.Exit(0)
		case "-v", "--version":
			printVersion(os.Stdout)
			os.Exit(0)
		case "-J", "--jpath":
			config.JPaths = append([]string{getArgValue(i)}, config.JPaths...)
		case "-t", "--tanka":
			config.ResolvePathsWithTanka = true
		case "-l", "--log-level":
			logLevel, err := log.ParseLevel(getArgValue(i))
			if err != nil {
				log.Fatalf("Invalid log level: %s", err)
			}
			log.SetLevel(logLevel)
		case "--lint":
			config.EnableLintDiagnostics = true
		case "--eval-diags":
			config.EnableEvalDiagnostics = true
		}
	}

	log.Infoln("Starting the language server")

	ctx := context.Background()
	stream := jsonrpc2.NewHeaderStream(utils.NewDefaultStdio())
	conn := jsonrpc2.NewConn(stream)
	client := protocol.ClientDispatcher(conn)

	s := server.NewServer(name, version, client, config)

	conn.Go(ctx, protocol.Handlers(
		protocol.ServerHandler(s, jsonrpc2.MethodNotFound)))
	<-conn.Done()
	if err := conn.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func getArgValue(i int) string {
	if i == len(os.Args)-1 {
		printHelp(os.Stdout)
		log.Fatalf("Expected value for option %s but found none.", os.Args[i])
	}
	return os.Args[i+1]
}
