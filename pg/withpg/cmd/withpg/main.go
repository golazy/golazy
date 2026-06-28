package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golazy.dev/pg/withpg"
)

func main() {
	var cfg withpg.Config
	var debug bool
	var port uint
	var versions versionFlag

	flag.StringVar(&cfg.SchemaFile, "schema", "", "SQL schema file to load before running the command")
	flag.Var(&versions, "version", "PostgreSQL version: 9, 10, 11, 12, 13, 14, 15, 16, 17, or 18; repeat to run the command once per version")
	flag.StringVar(&cfg.DataPath, "data", "", "data directory path to persist PostgreSQL data")
	flag.StringVar(&cfg.DBName, "db", "testdb", "database name")
	flag.UintVar(&port, "port", 0, "port, or 0 for a random available port")
	flag.BoolVar(&debug, "debug", false, "send embedded PostgreSQL logs to stderr")
	flag.Usage = usage
	flag.Parse()

	if debug {
		cfg.Logger = os.Stderr
	} else {
		cfg.Logger = io.Discard
	}
	cfg.Port = uint32(port)
	switch len(versions) {
	case 0:
	case 1:
		cfg.PgVersion = versions[0]
	default:
		cfg.PgVersions = []string(versions)
	}

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	err := withpg.WithPg(context.Background(), cfg, func(_ context.Context, db *withpg.DB) error {
		if debug {
			fmt.Fprintf(os.Stderr, "DATABASE_URL=%s\n", db.URL())
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = append(os.Environ(),
			"DATABASE_URL="+db.URL(),
			"GOLAZY_PG_DATABASE_URL="+db.URL(),
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	})
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: withpg [options] -- command [args...]")
	fmt.Fprintln(os.Stderr)
	flag.PrintDefaults()
}

type versionFlag []string

func (v *versionFlag) String() string {
	return strings.Join(*v, ",")
}

func (v *versionFlag) Set(value string) error {
	*v = append(*v, value)
	return nil
}
