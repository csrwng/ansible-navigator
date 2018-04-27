package parser

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

// Kind constants for parser nodes (from yaml.v2)
const (
	documentNode = 1 << iota
	mappingNode
	sequenceNode
	scalarNode
	aliasNode
)

type AnsibleNode struct {
	StartLine int
	EndLine   int
	Type      AnsibleNodeType
	Children  []AnsibleNode
	Reference *AnsibleReference
}

type AnsibleReference struct {
	Type  ReferenceType
	Value string
}

type AnsibleNodeType int

const (
	NodeTypePlaybook AnsibleNodeType = iota + 1
	NodeTypeImportPlaybook
	NodeTypePlay
	NodeTypeTaskList
	NodeTypeTask
	NodeTypeImportRole
	NodeTypeRoleList
	NodeTypeRole
	NodeTypeIncludeTasks
)

type ReferenceType int

const (
	PlaybookReference ReferenceType = iota + 1
	RoleReference
	TaskReference
)

type AnsibleDocKind int

const (
	UnknownDoc AnsibleDocKind = iota + 1
	PlaybookDoc
	RoleDoc
)

type StringSet map[string]struct{}

func (s StringSet) Add(str string) {
	s[str] = struct{}{}
}

func (s StringSet) Has(str string) bool {
	_, exists := s[str]
	return exists
}

func Parse(doc []byte, kind AnsibleDocKind, debug bool) (*AnsibleNode, error) {
	var docNode yaml.YAMLNode
	var err error
	listdata := []interface{}{}
	docNode, err = yaml.UnmarshalWithNode(doc, &listdata)
	if err != nil {
		mapdata := map[interface{}]interface{}{}
		docNode, err = yaml.UnmarshalWithNode(doc, &mapdata)
		if err != nil {
			return nil, err
		}
	}
	var result *AnsibleNode
	switch kind {
	case PlaybookDoc:
		result, err = parseAnsiblePlaybook(docNode)
	case RoleDoc:
		result, err = parseTaskList(docNode.Children()[0])
	default:
		return nil, fmt.Errorf("Invalid document type: %v", kind)
	}
	if debug && err == nil {
		DescribeAnsibleNode("", result)
	}

	return result, err
}

func parseAnsiblePlaybook(node yaml.YAMLNode) (*AnsibleNode, error) {
	playbook := &AnsibleNode{
		Type: NodeTypePlaybook,
	}
	if err := validateNodeKind(node, documentNode); err != nil {
		return nil, err
	}
	if err := validateChildCount(node, 1); err != nil {
		return nil, err
	}
	listParent := node.Children()[0]
	if err := validateNodeKind(listParent, sequenceNode); err != nil {
		return nil, err
	}
	playbook.StartLine = listParent.Line()
	endLine := playbook.StartLine // Will be incremented as we find children
	children := []AnsibleNode{}
	for _, child := range listParent.Children() {
		if err := validateNodeKind(child, mappingNode); err != nil {
			return nil, err
		}
		keys := mapKeys(child)
		if keys.Has("import_playbook") {
			importPlaybook, err := parseImportPlaybook(child)
			if err != nil {
				return nil, err
			}
			children = append(children, *importPlaybook)
			if importPlaybook.EndLine > endLine {
				endLine = importPlaybook.EndLine
			}
		} else {
			play, err := parsePlay(child)
			if err != nil {
				return nil, err
			}
			children = append(children, *play)
			if play.EndLine > endLine {
				endLine = play.EndLine
			}
		}
	}
	playbook.Children = children
	playbook.EndLine = endLine
	return playbook, nil
}

func DescribeAnsibleNode(indent string, node *AnsibleNode) {
	fmt.Printf("%s[%d-%d] %s %s\n", indent, node.StartLine, node.EndLine, nodeType(node.Type), nodeRef(node))
	for i := range node.Children {
		DescribeAnsibleNode(indent+"  ", &node.Children[i])
	}
}

func nodeType(t AnsibleNodeType) string {

	switch t {
	case NodeTypePlaybook:
		return "Playbook"
	case NodeTypeImportPlaybook:
		return "ImportPlaybook"

	case NodeTypePlay:
		return "Play"

	case NodeTypeTaskList:
		return "TaskList"

	case NodeTypeTask:
		return "Task"

	case NodeTypeImportRole:
		return "ImportRole"

	case NodeTypeRoleList:
		return "RoleList"

	case NodeTypeRole:
		return "Role"

	case NodeTypeIncludeTasks:
		return "IncludeTasks"
	}
	return "Unknown"
}

func nodeRef(node *AnsibleNode) string {
	if node.Reference == nil {
		return ""
	}
	refType := ""
	switch node.Reference.Type {
	case PlaybookReference:
		refType = "Playbook"

	case RoleReference:
		refType = "Role"

	case TaskReference:
		refType = "Task"
	}
	return fmt.Sprintf(" --> %s(%s)", refType, node.Reference.Value)
}

func parseTaskList(node yaml.YAMLNode) (*AnsibleNode, error) {
	taskList := &AnsibleNode{
		Type: NodeTypeTaskList,
	}
	taskList.StartLine = node.Line()
	taskList.EndLine = lastLine(node)

	children := node.Children()
	tasks := []AnsibleNode{}
	for _, child := range children {
		task, err := parseTask(child)
		if err != nil {
			return nil, err
		}
		if task != nil {
			tasks = append(tasks, *task)
		}
	}
	taskList.Children = tasks
	return taskList, nil
}

func parseTask(node yaml.YAMLNode) (*AnsibleNode, error) {
	keys := mapKeys(node)
	switch {
	case keys.Has("block"):
		blockTasks := mapKeyValue(node, "block")
		return parseTaskList(blockTasks)
	case keys.Has("import_role"):
		importRoleNode := mapKeyValue(node, "import_role")
		return parseImportRole(importRoleNode)
	case keys.Has("include_role"):
		includeRoleNode := mapKeyValue(node, "include_role")
		return parseImportRole(includeRoleNode)
	case keys.Has("include_tasks"):
		includeTasksNode := mapKeyValue(node, "include_tasks")
		return parseIncludeTasks(includeTasksNode)
	default: // Tasks we don't care about (yet)
		return nil, nil
	}
}

func parseIncludeTasks(node yaml.YAMLNode) (*AnsibleNode, error) {
	includeTasks := &AnsibleNode{
		Type:      NodeTypeIncludeTasks,
		StartLine: node.Line(),
		EndLine:   lastLine(node),
	}
	if node.Kind() != scalarNode {
		return nil, nil
	}
	includeTasks.Reference = &AnsibleReference{
		Type:  TaskReference,
		Value: node.Value(),
	}
	return includeTasks, nil
}

func parseRoleList(node yaml.YAMLNode) (*AnsibleNode, error) {
	roleList := &AnsibleNode{
		Type:      NodeTypeRoleList,
		StartLine: node.Line(),
		EndLine:   lastLine(node),
	}
	childNodes := node.Children()
	roles := []AnsibleNode{}
	for _, childNode := range childNodes {
		role, err := parseRole(childNode)
		if err != nil {
			return nil, err
		}
		if role != nil {
			roles = append(roles, *role)
		}
	}
	roleList.Children = roles
	return roleList, nil
}

func parseRole(node yaml.YAMLNode) (*AnsibleNode, error) {
	role := &AnsibleNode{
		Type:      NodeTypeRole,
		StartLine: node.Line(),
		EndLine:   lastLine(node),
	}
	switch node.Kind() {
	case scalarNode:
		role.Reference = &AnsibleReference{
			Type:  RoleReference,
			Value: node.Value(),
		}
	case mappingNode:
		value := mapKeyValue(node, "role").Value()
		role.Reference = &AnsibleReference{
			Type:  RoleReference,
			Value: value,
		}
	default: // Other node types not supported
		return nil, nil
	}
	return role, nil
}

func parseImportRole(node yaml.YAMLNode) (*AnsibleNode, error) {
	keys := mapKeys(node)
	roleRef := mapKeyValue(node, "name").Value()
	if keys.Has("tasks_from") {
		tasksFile := mapKeyValue(node, "tasks_from").Value()
		roleRef = roleRef + "/tasks/" + tasksFile
	}
	importRole := &AnsibleNode{
		Type:      NodeTypeImportRole,
		StartLine: node.Line(),
	}
	importRole.EndLine = lastLine(node)
	importRole.Reference = &AnsibleReference{
		Type:  RoleReference,
		Value: roleRef,
	}
	return importRole, nil
}

func parseImportPlaybook(node yaml.YAMLNode) (*AnsibleNode, error) {
	importPlaybook := &AnsibleNode{
		Type: NodeTypeImportPlaybook,
	}
	importPlaybook.StartLine = node.Line()
	importPlaybook.EndLine = lastLine(node)
	playbookRefNode := mapKeyValue(node, "import_playbook")
	if err := validateNodeKind(playbookRefNode, scalarNode); err != nil {
		return nil, err
	}
	importPlaybook.Reference = &AnsibleReference{
		Type:  PlaybookReference,
		Value: playbookRefNode.Value(),
	}
	return importPlaybook, nil
}

func parsePlay(node yaml.YAMLNode) (*AnsibleNode, error) {
	play := &AnsibleNode{
		Type: NodeTypePlay,
	}
	play.StartLine = node.Line()
	play.EndLine = lastLine(node)
	keys := mapKeys(node)
	taskListKeys := []string{"tasks", "pre_tasks", "post_tasks"}
	for _, taskListKey := range taskListKeys {
		if !keys.Has(taskListKey) {
			continue
		}
		taskListNode := mapKeyValue(node, taskListKey)
		taskList, err := parseTaskList(taskListNode)
		if err != nil {
			return nil, err
		}
		play.Children = append(play.Children, *taskList)
	}
	if keys.Has("roles") {
		roleListNode := mapKeyValue(node, "roles")
		roleList, err := parseRoleList(roleListNode)
		if err != nil {
			return nil, err
		}
		play.Children = append(play.Children, *roleList)
	}
	return play, nil
}

func validateNodeKind(node yaml.YAMLNode, kind int) error {
	if node.Kind() != kind {
		return fmt.Errorf("unexpected node kind. Expected: %v, Got: %v", kind, node.Kind())
	}
	return nil
}

func lastLine(node yaml.YAMLNode) int {
	line := node.Line()
	switch node.Kind() {
	case scalarNode:
		line += (lineCount(node.Value()) - 1)
	default:
		for _, child := range node.Children() {
			childLine := lastLine(child)
			if childLine > line {
				line = childLine
			}
		}
	}
	return line
}

func lineCount(str string) int {
	result := strings.Split(str, "\n")
	return len(result)
}

func validateChildCount(node yaml.YAMLNode, count int) error {
	if len(node.Children()) != count {
		return fmt.Errorf("unexpected child count. Expected: %v, Got: %v", count, len(node.Children()))
	}
	return nil
}

func mapKeyValue(node yaml.YAMLNode, key string) yaml.YAMLNode {
	children := node.Children()
	for i := 0; i < len(children); i += 2 {
		if err := validateNodeKind(children[i], scalarNode); err == nil {
			if children[i].Value() == key {
				return children[i+1]
			}
		}
	}
	return nil
}

func mapKeys(node yaml.YAMLNode) StringSet {
	s := StringSet{}
	children := node.Children()
	for i := 0; i < len(children); i += 2 {
		if err := validateNodeKind(children[i], scalarNode); err == nil {
			s.Add(children[i].Value())
		}
	}
	return s
}
