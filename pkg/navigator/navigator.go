package navigator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/csrwng/ansible-navigator/pkg/parser"
)

// AnsibleNavigator attemps to determine a file that corresponds to the location
// specified by the Row and Column fields
type AnsibleNavigator struct {
	File   string
	Row    int
	Column int
	Debug  bool
}

func (n *AnsibleNavigator) Navigate() (string, error) {
	if _, err := os.Stat(n.File); err != nil {
		return "", fmt.Errorf("cannot stat file %s: %v", n.File, err)
	}
	fileContent, err := ioutil.ReadFile(n.File)
	if err != nil {
		return "", fmt.Errorf("cannot read file %s: %v", n.File, err)
	}
	docKind := determineDocKind(n.File, n.Debug)
	if docKind == parser.UnknownDoc {
		if n.Debug {
			fmt.Printf("Unknown document type\n")
		}
		return "", nil
	}

	ansibleNode, err := parser.Parse(fileContent, docKind, n.Debug)
	locationNode := locateNode(ansibleNode, n.Row-1)
	if locationNode == nil {
		if n.Debug {
			fmt.Printf("Could not find node that matches location %d\n", n.Row-1)
		}
		return "", nil
	}
	if n.Debug {
		fmt.Printf("Located node:\n")
		parser.DescribeAnsibleNode("", locationNode)
	}
	if locationNode.Reference == nil {
		return "", nil
	}
	referencedFile := getReferencedFile(n.File, locationNode.Reference)

	return referencedFile, nil
}

func determineDocKind(filePath string, debug bool) parser.AnsibleDocKind {
	dirs := splitPath(filePath)
	count := len(dirs)
	if debug {
		fmt.Printf("Doc dir parts: %v\n", dirs)
	}

	if count >= 4 {
		if dirs[count-4] == "roles" && dirs[count-2] == "tasks" {
			return parser.RoleDoc
		}
	}
	for i := 0; i < count-1; i++ {
		if dirs[i] == "playbooks" {
			return parser.PlaybookDoc
		}
	}
	return parser.UnknownDoc
}

func locateNode(root *parser.AnsibleNode, row int) *parser.AnsibleNode {
	node := root
	if inRange(row, node.StartLine, node.EndLine) {
		for i, _ := range node.Children {
			child := &node.Children[i]
			childNode := locateNode(child, row)
			if childNode != nil {
				return childNode
			}
		}
		return node
	}
	return nil
}

func inRange(num, low, high int) bool {
	return num >= low && num <= high
}

func getReferencedFile(path string, ref *parser.AnsibleReference) string {
	var referencedPath string
	var err error
	switch ref.Type {
	case parser.PlaybookReference, parser.TaskReference:
		referencedPath = filepath.Join(filepath.Dir(path), ref.Value)
	case parser.RoleReference:
		referencedPath = filepath.Join(filepath.Join(filepath.Dir(path), "roles"), ref.Value, "tasks", "main.yml")
		if _, err = os.Stat(referencedPath); err != nil {
			referencedPath = filepath.Join(filepath.Join(filepath.Dir(path), "roles"), ref.Value)
		}
		referencedPath, err = filepath.EvalSymlinks(referencedPath)
		if err != nil {
			referencedPath = ""
		}
	}
	if referencedPath != "" {
		if _, err = os.Stat(referencedPath); err == nil {
			return referencedPath
		}
	}
	return ""
}

func splitPath(filePath string) []string {
	result := []string{filepath.Base(filePath)}
	toSplit := filepath.Dir(filePath)
	for toSplit != "." && toSplit != "/" {
		base := filepath.Base(toSplit)
		if len(base) > 0 {
			result = append([]string{base}, result...)
		}
		toSplit = filepath.Dir(toSplit)
	}
	return result
}
