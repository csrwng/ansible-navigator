package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"gopkg.in/yaml.v2"
)

func NewAnsibleParserCmd() *cobra.Command {
	return &cobra.Command{
		Use: "ansible-parser PATH",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.Usage()
				return
			}
			parseYAMLFiles(args[0])
		},
	}
}

func parseYAMLFiles(dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml") {
			parseFile(path)
		}
		return nil
	})
}

func parseFile(path string) {
	result := ""
	var node yaml.YAMLNode
	filedata, err := ioutil.ReadFile(path)
	if err != nil {
		result = "cannotread"
	} else {
		listdata := []interface{}{}
		node, err = yaml.UnmarshalWithNode(filedata, &listdata)
		if err == nil {
			result = "list"
		} else {
			mapdata := map[interface{}]interface{}{}
			node, err = yaml.UnmarshalWithNode(filedata, &mapdata)
			if err == nil {
				result = "map"
			} else {
				result = "error"
			}
		}
	}
	fmt.Printf("%s => %s\n", path, result)
	describeNode("", node, os.Stdout)
}

const (
	documentNode = 1 << iota
	mappingNode
	sequenceNode
	scalarNode
	aliasNode
)

func kindToString(kind int) string {
	switch kind {
	case documentNode:
		return "document"
	case mappingNode:
		return "mapping"
	case sequenceNode:
		return "sequence"
	case scalarNode:
		return "scalar"
	case aliasNode:
		return "alias"
	default:
		return "unknown"
	}
}

func describeNode(indent string, node yaml.YAMLNode, out io.Writer) {
	if val := node.Value(); len(val) > 0 {
		fmt.Fprintf(out, "%s%s(%d,%d): %s\n", indent, kindToString(node.Kind()), node.Line(), node.Column(), node.Value())
	} else {
		fmt.Fprintf(out, "%s%s(%d,%d)\n", indent, kindToString(node.Kind()), node.Line(), node.Column())
	}

	for _, child := range node.Children() {
		describeNode(indent+" ", child, out)
	}
}

func main() {
	cmd := NewAnsibleParserCmd()
	cmd.Execute()
}
