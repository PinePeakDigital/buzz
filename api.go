package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const apiUsage = `Usage: buzz api [-X|--method <METHOD>] [-d|--data <key=value>]... <path>

Make an authenticated request to the Beeminder API using your buzz
credentials. <path> is relative to the API root, e.g. "users/me.json"
or "users/me/goals/<slug>.json". The auth_token is added automatically.

Options:
  -X, --method <METHOD>   HTTP method: GET (default), POST, PUT, PATCH, DELETE
  -d, --data <key=value>  Request parameter (repeatable). Sent as query
                          parameters for GET/DELETE, form body otherwise.

Examples:
  buzz api users/me.json
  buzz api users/me/goals/read.json
  buzz api -X POST -d value=1 -d "comment=via buzz" users/me/goals/read/datapoints.json`

// keyValueFlag collects repeatable -d/--data "key=value" pairs.
type keyValueFlag []string

func (k *keyValueFlag) String() string { return strings.Join(*k, ", ") }

func (k *keyValueFlag) Set(v string) error {
	*k = append(*k, v)
	return nil
}

// handleAPICommand makes a raw, authenticated request to the Beeminder API.
func handleAPICommand() {
	if !ConfigExists() {
		fmt.Fprintln(os.Stderr, "Error: No configuration found. Please run 'buzz auth login' to authenticate.")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load config: %s\n", redactError(err))
		os.Exit(1)
	}

	client := NewHTTPClient(config)
	os.Exit(runAPICommand(os.Args[2:], client, os.Stdout, os.Stderr))
}

// runAPICommand is the testable core of `buzz api`. It returns the process exit
// code and writes all output to the provided writers.
func runAPICommand(args []string, client Client, stdout, stderr io.Writer) int {
	apiFlags := flag.NewFlagSet("api", flag.ContinueOnError)
	// Silence the flag package's own error/usage output; we print our own
	// message and richer apiUsage on both --help and parse errors.
	apiFlags.SetOutput(io.Discard)
	apiFlags.Usage = func() {}

	var method string
	apiFlags.StringVar(&method, "method", "GET", "HTTP method")
	apiFlags.StringVar(&method, "X", "GET", "HTTP method (shorthand)")

	var data keyValueFlag
	apiFlags.Var(&data, "data", "Request parameter key=value (repeatable)")
	apiFlags.Var(&data, "d", "Request parameter (shorthand)")

	// Allow flags on either side of the positional <path>, mirroring the view
	// command: re-parse from the remainder until only non-flag args are left.
	var positional []string
	remaining := args
	for len(remaining) > 0 {
		if err := apiFlags.Parse(remaining); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				fmt.Fprintln(stdout, apiUsage)
				return 0
			}
			fmt.Fprintf(stderr, "Error parsing flags: %s\n", redactError(err))
			fmt.Fprintln(stderr, apiUsage)
			return 2
		}
		rest := apiFlags.Args()
		if len(rest) == 0 {
			break
		}
		positional = append(positional, rest[0])
		remaining = rest[1:]
	}

	if len(positional) != 1 {
		if len(positional) == 0 {
			fmt.Fprintln(stderr, "Error: Missing required <path> argument")
		} else {
			fmt.Fprintf(stderr, "Error: Too many arguments: %v\n", positional[1:])
		}
		fmt.Fprintln(stderr, apiUsage)
		return 1
	}
	path := positional[0]

	method = strings.ToUpper(strings.TrimSpace(method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		fmt.Fprintf(stderr, "Error: Unsupported method %q (use GET, POST, PUT, PATCH, or DELETE)\n", method)
		return 1
	}

	params := make(map[string]string)
	for _, kv := range data {
		key, val, found := strings.Cut(kv, "=")
		if !found || key == "" {
			fmt.Fprintf(stderr, "Error: Invalid --data %q (expected key=value)\n", kv)
			return 1
		}
		params[key] = val
	}

	status, body, err := client.APIRequest(context.Background(), method, path, params)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", redactError(err))
		return 1
	}

	// Pretty-print JSON responses; fall back to the raw body otherwise.
	var pretty bytes.Buffer
	if len(body) > 0 && json.Indent(&pretty, body, "", "  ") == nil {
		fmt.Fprintln(stdout, pretty.String())
	} else if len(body) > 0 {
		fmt.Fprintln(stdout, strings.TrimRight(string(body), "\n"))
	}

	// Surface non-2xx responses with a nonzero exit code while still printing
	// the body above, so error details from the API remain visible.
	if status < 200 || status >= 300 {
		fmt.Fprintf(stderr, "API returned status %d\n", status)
		return 1
	}

	return 0
}
