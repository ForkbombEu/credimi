// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// codeHeaderKeys are the keys the header document may carry.
var codeHeaderKeys = []string{"appId", "env", "name", "tags"}

// codeAllowedCommands is the Maestro command allowlist shared with the runner's validator.
var codeAllowedCommands = map[string]bool{
	"launchApp":             true,
	"tapOn":                 true,
	"doubleTapOn":           true,
	"longPressOn":           true,
	"inputText":             true,
	"eraseText":             true,
	"hideKeyboard":          true,
	"pressKey":              true,
	"back":                  true,
	"swipe":                 true,
	"scroll":                true,
	"scrollUntilVisible":    true,
	"assertVisible":         true,
	"assertNotVisible":      true,
	"extendedWaitUntil":     true,
	"waitForAnimationToEnd": true,
	"takeScreenshot":        true,
	"runFlow":               true,
	"retry":                 true,
	"repeat":                true,
	"setPermissions":        true,
	"openLink":              true,
	"copyTextFrom":          true,
	"pasteText":             true,
}

// codeNestedCommands are the commands whose inline `commands` are walked.
var codeNestedCommands = map[string]bool{"runFlow": true, "retry": true, "repeat": true}

// codeInteractions are the commands that act on the app; the runner's
// validator uses the same list for the final assertion rule.
var codeInteractions = map[string]bool{
	"tapOn":              true,
	"doubleTapOn":        true,
	"longPressOn":        true,
	"inputText":          true,
	"eraseText":          true,
	"pressKey":           true,
	"back":               true,
	"swipe":              true,
	"scroll":             true,
	"scrollUntilVisible": true,
	"openLink":           true,
	"copyTextFrom":       true,
	"pasteText":          true,
}

var (
	// codeSubstitution matches a Maestro `${...}` substitution, unterminated ones included.
	codeSubstitution = regexp.MustCompile(`\$\{([^}]*)\}?`)
	codeIdentifier   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	// codeScreenshot bounds takeScreenshot names. The stored screenshot file
	// stem is Snakecase(<pipeline step id>-<name>), capped at 100 characters
	// by PocketBase, and evidence.screenshot_named
	// (pkg/fcaf/validators/evidence_screenshot_named.go) reports blocked when
	// a name was truncated: 64 leaves room for step ids and covers FCAF's
	// required names (up to 31 characters, e.g. fcaf-engagement-haip-vp-invoked).
	codeScreenshot = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
)

// PlayStoreAppID is the store app an Android install-app action may target
// besides the wallet.
const PlayStoreAppID = "com.android.vending"

// AllowedAppIDs returns the app IDs an action may target (header appId,
// launchApp, setPermissions): the package under test and, for an Android
// install-app action, the Play Store app. Web flows open the device's default
// browser with openLink instead of targeting it.
func AllowedAppIDs(packageID, slot, platform string) []string {
	if slot == SlotInstallApp && platform == PlatformAndroid {
		return []string{packageID, PlayStoreAppID}
	}
	return []string{packageID}
}

// StoreLinks returns the store pages an install-app action may open with
// openLink: none for another slot or platform; on Android the Play Store page
// of packageID in its two forms plus the wallet's own storeURL. An empty
// storeURL adds nothing.
func StoreLinks(slot, platform, packageID, storeURL string) []string {
	if slot != SlotInstallApp || platform != PlatformAndroid {
		return nil
	}
	links := []string{
		"market://details?id=" + packageID,
		"https://play.google.com/store/apps/details?id=" + packageID,
	}
	if storeURL != "" && !slices.Contains(links, storeURL) {
		links = append(links, storeURL)
	}
	return links
}

// CodeScope is what an action may use: AppIDs are the apps it may target,
// Parameters the session parameter names it may reference as `${name}`, and
// StoreLinks the store pages it may open. WaitSeconds is the wait of a wait
// action (0 for any other action): its last command must wait that long
// instead of asserting an outcome.
type CodeScope struct {
	AppIDs      []string
	Parameters  []string
	StoreLinks  []string
	WaitSeconds int
}

// ValidateCode returns every reason a Maestro action cannot be accepted, once
// each and in discovery order; an empty result means the action is valid.
func ValidateCode(code string, scope CodeScope) []string {
	parameters, waitSeconds := scope.Parameters, scope.WaitSeconds
	docs, err := codeDocuments(code)
	if err != nil {
		return []string{fmt.Sprintf("invalid YAML: %v", err)}
	}
	if problems := codeKeyProblems(docs); len(problems) > 0 {
		return problems
	}
	if len(docs) != 2 {
		return []string{fmt.Sprintf(
			"expected two YAML documents (header, `---`, command list), found %d", len(docs))}
	}
	v := codeValidator{appIDs: scope.AppIDs, parameters: parameters, storeLinks: scope.StoreLinks}
	header, commands := docs[0], docs[1]
	if header == nil || header.Kind != yaml.MappingNode {
		v.add("the first document must be the header mapping (appId, ...)")
		header = nil
	}
	if commands == nil || commands.Kind != yaml.SequenceNode || len(commands.Content) == 0 {
		v.add("the second document must be a non-empty list of commands")
		commands = nil
	}

	var foreign []string
	for _, doc := range docs {
		codeStrings(doc, func(text string) {
			for _, m := range codeSubstitution.FindAllStringSubmatch(text, -1) {
				ok := strings.HasSuffix(m[0], "}") && codeIdentifier.MatchString(m[1]) &&
					slices.Contains(parameters, m[1])
				if !ok && !slices.Contains(foreign, m[0]) {
					foreign = append(foreign, m[0])
				}
			}
		})
	}
	if len(foreign) > 0 {
		v.add(fmt.Sprintf("substitutions are not allowed: %s; write literal values or session "+
			"parameters (Maestro must not evaluate expressions or read the host environment)",
			codeList(foreign)))
	}

	v.header(header)

	v.walk(commands, "commands", v.command)

	var lastName string
	var lastArg *yaml.Node
	if commands != nil {
		for _, item := range commands.Content {
			if name, arg, ok := codeCommand(item); ok && name != "takeScreenshot" {
				lastName, lastArg = name, arg
			}
		}
	}
	switch {
	case waitSeconds > 0:
		if !codeIsWait(lastName, lastArg, waitSeconds) {
			v.add(fmt.Sprintf("the last command other than takeScreenshot must wait %d s: "+
				"extendedWaitUntil with optional: true and timeout: %d",
				waitSeconds, waitSeconds*1000))
		}
	case !codeIsOutcomeAssertion(lastName, lastArg):
		v.add("the last command other than takeScreenshot must assert the outcome: " +
			"`assertVisible`, or `extendedWaitUntil` with `visible`")
	default:
		if selector, ok := codeAssertedSelector(lastName, lastArg); ok &&
			codeReachedBeforeLastInteraction(commands, selector) {
			v.add("the final assertion checks \"" + selector.String() + "\", which the action " +
				"already reached before its last interaction; assert the screen the outcome shows")
		}
	}
	return v.problems
}

// codeValidator collects deduplicated problems for one action.
type codeValidator struct {
	appIDs     []string
	parameters []string
	storeLinks []string
	problems   []string
}

func (v *codeValidator) add(problem string) {
	if !slices.Contains(v.problems, problem) {
		v.problems = append(v.problems, problem)
	}
}

func (v *codeValidator) header(header *yaml.Node) {
	var extra []string
	var appID, env *yaml.Node
	if header != nil {
		for i := 0; i+1 < len(header.Content); i += 2 {
			key := codeResolve(header.Content[i])
			if !codeIsString(key) {
				v.add("header keys must be strings")
				continue
			}
			switch key.Value {
			case "appId":
				appID = codeResolve(header.Content[i+1])
			case "env":
				env = codeResolve(header.Content[i+1])
			case "name", "tags":
			default:
				extra = append(extra, key.Value)
			}
		}
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		v.add(fmt.Sprintf("header keys not allowed: %s; allowed: %s",
			codeList(extra), codeList(codeHeaderKeys)))
	}
	switch {
	case !codeIsNull(appID) && !codeIsString(appID):
		v.add("appId must be a string")
	case !codeIsString(appID) || !slices.Contains(v.appIDs, appID.Value):
		v.add(
			fmt.Sprintf(
				"header appId must be %s, found %s",
				codeExpected(v.appIDs),
				codeRepr(appID),
			),
		)
	}
	if env != nil && env.Kind == yaml.MappingNode {
		var shadowed []string
		for i := 0; i < len(env.Content); i += 2 {
			key := codeResolve(env.Content[i])
			if codeIsString(key) && slices.Contains(v.parameters, key.Value) &&
				!slices.Contains(shadowed, key.Value) {
				shadowed = append(shadowed, key.Value)
			}
		}
		if len(shadowed) > 0 {
			sort.Strings(shadowed)
			v.add("header env must not set session parameters: " + codeList(shadowed))
		}
	}
}

// walk visits every command, nested ones included, as (name, argument, location).
func (v *codeValidator) walk(
	commands *yaml.Node,
	path string,
	visit func(string, *yaml.Node, string),
) {
	if commands == nil {
		return
	}
	for i, item := range commands.Content {
		where := fmt.Sprintf("%s[%d]", path, i)
		name, arg, ok := codeCommand(item)
		if !ok {
			v.add(where + ": not a command (use `- name` or `- name: argument`)")
			continue
		}
		visit(name, arg, where)
		if !codeNestedCommands[name] {
			continue
		}
		if arg == nil || arg.Kind != yaml.MappingNode {
			v.add(fmt.Sprintf("%s %s: needs a mapping with inline `commands`", where, name))
			continue
		}
		if codeLookup(arg, "file") != nil {
			v.add(fmt.Sprintf("%s %s: `file` is not allowed; use inline `commands`", where, name))
		}
		nested := codeLookup(arg, "commands")
		if nested == nil || nested.Kind != yaml.SequenceNode || len(nested.Content) == 0 {
			v.add(fmt.Sprintf("%s %s: needs a non-empty inline `commands` list", where, name))
			continue
		}
		v.walk(nested, fmt.Sprintf("%s.%s.commands", where, name), visit)
	}
}

func (v *codeValidator) command(name string, arg *yaml.Node, where string) {
	if !codeAllowedCommands[name] {
		v.add(fmt.Sprintf("%s: command `%s` is not allowed", where, name))
		return
	}
	switch name {
	case "launchApp", "setPermissions":
		target := arg
		if arg != nil && arg.Kind == yaml.MappingNode {
			target = codeLookup(arg, "appId")
		} else if name == "setPermissions" {
			target = nil
		}
		switch {
		case codeIsNull(target):
		case !codeIsString(target):
			v.add(where + ": appId must be a string")
		case !slices.Contains(v.appIDs, target.Value):
			v.add(
				fmt.Sprintf(
					"%s: %s targets %s, not the app under test",
					where,
					name,
					codeRepr(target),
				),
			)
		}
	case "openLink":
		v.openLink(arg, where)
	case "takeScreenshot":
		path := arg
		if arg != nil && arg.Kind == yaml.MappingNode {
			path = codeLookup(arg, "path")
		}
		if !codeIsString(path) || !codeScreenshot.MatchString(path.Value) {
			v.add(fmt.Sprintf("%s: screenshot names must be 1-64 letters, digits, '-' or '_', "+
				"found %s", where, codeRepr(path)))
		}
	}
}

func (v *codeValidator) openLink(arg *yaml.Node, where string) {
	link := arg
	if arg != nil && arg.Kind == yaml.MappingNode {
		link = codeLookup(arg, "link")
	}
	if codeIsString(link) && (slices.Contains(v.storeLinks, link.Value) ||
		strings.HasPrefix(link.Value, "${") && strings.HasSuffix(link.Value, "}") &&
			slices.Contains(v.parameters, link.Value[2:len(link.Value)-1])) {
		return
	}
	if len(v.storeLinks) > 0 {
		links := slices.Clone(v.storeLinks)
		sort.Strings(links)
		v.add(fmt.Sprintf(
			"%s: openLink must open a session parameter or the wallet's store page: %s",
			where,
			strings.Join(links, ", "),
		))
		return
	}
	if len(v.parameters) == 0 {
		v.add(where + ": openLink is not allowed: this session has no parameters")
		return
	}
	names := slices.Clone(v.parameters)
	sort.Strings(names)
	if len(names) == 1 {
		v.add(
			fmt.Sprintf(
				"%s: openLink must open a session parameter: write link: ${%s}",
				where,
				names[0],
			),
		)
		return
	}
	v.add(
		fmt.Sprintf(
			"%s: openLink must open a session parameter: write link: ${NAME}, with NAME one of %s",
			where,
			strings.Join(names, ", "),
		),
	)
}

// codeDocuments parses every YAML document of code; an empty document is nil.
func codeDocuments(code string) ([]*yaml.Node, error) {
	decoder := yaml.NewDecoder(strings.NewReader(code))
	var docs []*yaml.Node
	for {
		var doc yaml.Node
		err := decoder.Decode(&doc)
		if errors.Is(err, io.EOF) {
			return docs, nil
		}
		if err != nil {
			return nil, err
		}
		var root *yaml.Node
		if len(doc.Content) > 0 {
			root = codeResolve(doc.Content[0])
			if root.Kind == yaml.ScalarNode && root.Tag == "!!null" {
				root = nil
			}
		}
		docs = append(docs, root)
	}
}

// codeKeyProblems refuses duplicate mapping keys and merge keys in every
// document, in document order: either makes the mapping Credimi checks differ
// from the one Maestro runs. Aliased nodes are checked where they are defined.
func codeKeyProblems(docs []*yaml.Node) []string {
	var problems []string
	add := func(problem string) {
		if !slices.Contains(problems, problem) {
			problems = append(problems, problem)
		}
	}
	var visit func(*yaml.Node)
	visit = func(node *yaml.Node) {
		if node == nil || node.Kind == yaml.AliasNode {
			return
		}
		if node.Kind != yaml.MappingNode {
			for _, child := range node.Content {
				visit(child)
			}
			return
		}
		seen := map[string]bool{}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := codeResolve(node.Content[i])
			switch {
			case key.Tag == "!!merge":
				add("merge keys are not allowed")
			case key.Kind == yaml.ScalarNode && seen[key.Value]:
				add(fmt.Sprintf("duplicate key '%s'", key.Value))
			case key.Kind == yaml.ScalarNode:
				seen[key.Value] = true
			}
			visit(node.Content[i])
			visit(node.Content[i+1])
		}
	}
	for _, doc := range docs {
		visit(doc)
	}
	return problems
}

// codeCommand returns the name and argument of one list item: `- back` or `- tapOn: ...`.
func codeCommand(item *yaml.Node) (string, *yaml.Node, bool) {
	item = codeResolve(item)
	if codeIsString(item) {
		return item.Value, nil, true
	}
	if item != nil && item.Kind == yaml.MappingNode && len(item.Content) == 2 {
		key := codeResolve(item.Content[0])
		if codeIsString(key) {
			arg := codeResolve(item.Content[1])
			if arg.Kind == yaml.ScalarNode && arg.Tag == "!!null" {
				arg = nil
			}
			return key.Value, arg, true
		}
	}
	return "", nil, false
}

func codeIsOutcomeAssertion(name string, arg *yaml.Node) bool {
	if name == "assertVisible" {
		return true
	}
	return name == "extendedWaitUntil" && arg != nil && arg.Kind == yaml.MappingNode &&
		codeLookup(arg, "visible") != nil
}

// codeSelector is the element an assertion targets: its text, or its id.
type codeSelector struct {
	id    bool
	value string
}

func (s codeSelector) String() string {
	if s.id {
		return "id: " + s.value
	}
	return s.value
}

// codeAssertedSelector returns the selector an `assertVisible`, or an
// `extendedWaitUntil` with `visible`, checks: a string is its text, a
// mapping its string `text`, else its string `id`. Other selectors, and
// selectors with a `${...}` substitution, have none.
func codeAssertedSelector(name string, arg *yaml.Node) (codeSelector, bool) {
	target := arg
	switch {
	case name == "assertVisible":
	case name == "extendedWaitUntil" && arg != nil && arg.Kind == yaml.MappingNode:
		target = codeLookup(arg, "visible")
	default:
		return codeSelector{}, false
	}
	var selector codeSelector
	switch {
	case codeIsString(target):
		selector = codeSelector{value: target.Value}
	case target != nil && target.Kind == yaml.MappingNode:
		if text := codeLookup(target, "text"); codeIsString(text) {
			selector = codeSelector{value: text.Value}
		} else if id := codeLookup(target, "id"); codeIsString(id) {
			selector = codeSelector{id: true, value: id.Value}
		} else {
			return codeSelector{}, false
		}
	default:
		return codeSelector{}, false
	}
	if strings.Contains(selector.value, "${") {
		return codeSelector{}, false
	}
	return selector, true
}

// codeReachedBeforeLastInteraction reports whether selector was asserted or
// waited for before the last interaction, over the top-level commands and the
// inline commands of `runFlow` (never `retry` or `repeat` bodies, and never a
// `when:` condition).
func codeReachedBeforeLastInteraction(commands *yaml.Node, selector codeSelector) bool {
	var flat []*yaml.Node
	var flatten func(*yaml.Node)
	flatten = func(list *yaml.Node) {
		if list == nil || list.Kind != yaml.SequenceNode {
			return
		}
		for _, item := range list.Content {
			flat = append(flat, item)
			if name, arg, ok := codeCommand(item); ok && name == "runFlow" &&
				arg != nil && arg.Kind == yaml.MappingNode {
				flatten(codeLookup(arg, "commands"))
			}
		}
	}
	flatten(commands)
	last := -1
	for i, item := range flat {
		if name, _, ok := codeCommand(item); ok && codeInteractions[name] {
			last = i
		}
	}
	for _, item := range flat[:max(last, 0)] {
		name, arg, ok := codeCommand(item)
		if !ok {
			continue
		}
		if reached, ok := codeAssertedSelector(name, arg); ok && reached == selector {
			return true
		}
	}
	return false
}

// codeIsWait reports whether a command is the wait of a wait action:
// extendedWaitUntil on a visible condition (notVisible would complete as soon
// as the text is absent, so it would not wait), with `optional` written as the
// plain scalar `true` and `timeout` as a plain decimal integer equal to
// waitSeconds * 1000. The raw text is checked, so YAML 1.1 forms such as `yes`
// or `240_000` are refused.
func codeIsWait(name string, arg *yaml.Node, waitSeconds int) bool {
	if name != "extendedWaitUntil" || arg == nil || arg.Kind != yaml.MappingNode {
		return false
	}
	if codeLookup(arg, "visible") == nil {
		return false
	}
	optional, ok := codePlainScalar(codeLookup(arg, "optional"), "!!bool")
	if !ok || optional != "true" {
		return false
	}
	timeout, ok := codePlainScalar(codeLookup(arg, "timeout"), "!!int")
	if !ok || !codeDecimal.MatchString(timeout) {
		return false
	}
	value, err := strconv.Atoi(timeout)
	return err == nil && value == waitSeconds*1000
}

// codeDecimal matches a plain decimal integer: no sign, `_`, or base prefix.
var codeDecimal = regexp.MustCompile(`^[0-9]+$`)

// codePlainScalar returns the raw text of an unquoted, non-block scalar whose
// resolved tag is tag.
func codePlainScalar(node *yaml.Node, tag string) (string, bool) {
	node = codeResolve(node)
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != tag ||
		node.Style&^yaml.TaggedStyle != 0 {
		return "", false
	}
	return node.Value, true
}

// codeStrings calls visit with every scalar, mapping keys included, at any depth.
func codeStrings(node *yaml.Node, visit func(string)) {
	node = codeResolve(node)
	if node == nil {
		return
	}
	if node.Kind == yaml.ScalarNode {
		visit(node.Value)
		return
	}
	for _, child := range node.Content {
		codeStrings(child, visit)
	}
}

// codeLookup returns the value of key in a mapping node, or nil.
func codeLookup(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if k := codeResolve(mapping.Content[i]); codeIsString(k) && k.Value == key {
			return codeResolve(mapping.Content[i+1])
		}
	}
	return nil
}

func codeResolve(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	return node
}

func codeIsString(node *yaml.Node) bool {
	return node != nil && node.Kind == yaml.ScalarNode && node.Tag == "!!str"
}

func codeIsNull(node *yaml.Node) bool {
	return node == nil || (node.Kind == yaml.ScalarNode && node.Tag == "!!null")
}

// codeRepr renders a YAML value like the runner's Python messages: 'text' or None.
func codeRepr(node *yaml.Node) string {
	switch {
	case node == nil || (node.Kind == yaml.ScalarNode && node.Tag == "!!null"):
		return "None"
	case codeIsString(node):
		return "'" + node.Value + "'"
	case node.Kind == yaml.ScalarNode:
		return node.Value
	default:
		out, err := yaml.Marshal(node)
		if err != nil {
			return "?"
		}
		return strings.TrimSpace(string(out))
	}
}

// codeList renders strings like a Python list: ['a', 'b'].
func codeList(items []string) string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = "'" + item + "'"
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// codeExpected renders the accepted app IDs: 'a' or one of ['a', 'b'].
func codeExpected(ids []string) string {
	if len(ids) == 1 {
		return "'" + ids[0] + "'"
	}
	return "one of " + codeList(ids)
}
