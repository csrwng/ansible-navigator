package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/csrwng/ansible-navigator/pkg/navigator"
)

func NewAnsibleNavigatorCmd() *cobra.Command {
	debug := false
	flag.CommandLine.BoolVar(&debug, "debug", false, "If true, turns on debug output")
	return &cobra.Command{
		Use: "ansible-navigator FILENAME ROW COLUMN",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				cmd.Usage()
				return
			}
			err := navigate(args[0], args[1], args[2], debug, os.Stdout)
			if err != nil && debug { // If not debug, don't print an error
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
}

func navigate(fileName, rowStr, colStr string, debug bool, out io.Writer) error {
	row, err := strconv.Atoi(rowStr)
	if err != nil {
		return err
	}

	col, err := strconv.Atoi(colStr)
	if err != nil {
		return err
	}

	nav := &navigator.AnsibleNavigator{
		File:   fileName,
		Row:    row,
		Column: col,
		Debug:  debug,
	}
	path, err := nav.Navigate()
	if err != nil {
		return err
	}
	if len(path) > 0 {
		fmt.Fprintf(out, "%s", path)
	}
	return nil
}

func main() {
	cmd := NewAnsibleNavigatorCmd()
	cmd.Execute()
}
