package lazycli

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

func executeComplete(root *node, args []string, stdout io.Writer) int {
	selected := root
	prefix := ""
	if len(args) > 0 {
		prefix = args[len(args)-1]
		args = args[:len(args)-1]
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if child := selected.children[arg]; child != nil {
			selected = child
		}
	}
	type candidate struct{ value, help string }
	var candidates []candidate
	if strings.HasPrefix(prefix, "-") {
		for _, spec := range applicableHelpSpecs(selected) {
			value := "--" + spec.name
			if strings.HasPrefix(value, prefix) {
				candidates = append(candidates, candidate{value, spec.help})
			}
		}
	} else {
		for _, child := range sortedChildren(selected) {
			if strings.HasPrefix(child.name, prefix) {
				candidates = append(candidates, candidate{child.name, child.help})
			}
		}
		if selected == root && strings.HasPrefix("completion", prefix) {
			candidates = append(candidates, candidate{"completion", "Generate shell completion scripts"})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].value < candidates[j].value })
	for _, candidate := range candidates {
		fmt.Fprintf(stdout, "%s\t%s\n", candidate.value, candidate.help)
	}
	return 0
}

func executeCompletionScript(root *node, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintf(stderr, "usage: %s completion <bash|zsh|fish|powershell>\n", root.name)
		return 2
	}
	name := root.name
	switch args[0] {
	case "bash":
		fmt.Fprintf(stdout, `_lazy_complete() {
  local IFS=$'\n'
  COMPREPLY=($(%[1]s __complete "${COMP_WORDS[@]:1}" "${COMP_WORDS[COMP_CWORD]}" | cut -f1))
}
complete -F _lazy_complete %[1]s
`, name)
	case "zsh":
		fmt.Fprintf(stdout, `#compdef %[1]s
_%[1]s() {
  local -a values
  values=("${(@f)$(%[1]s __complete "${words[@]:1}")}")
  _describe '%[1]s' values
}
compdef _%[1]s %[1]s
`, name)
	case "fish":
		fmt.Fprintf(stdout, `function __%[1]s_complete
    %[1]s __complete (commandline -opc)[2..-1] (commandline -ct)
end
complete -c %[1]s -f -a '(__%[1]s_complete)'
`, name)
	case "powershell":
		fmt.Fprintf(stdout, `Register-ArgumentCompleter -Native -CommandName '%[1]s' -ScriptBlock {
  param($wordToComplete, $commandAst, $cursorPosition)
  $parts = $commandAst.CommandElements | Select-Object -Skip 1 | ForEach-Object { $_.Extent.Text }
  %[1]s __complete @parts $wordToComplete | ForEach-Object {
    $fields = $_ -split ([char]9), 2
    [System.Management.Automation.CompletionResult]::new($fields[0], $fields[0], 'ParameterValue', $fields[1])
  }
}
`, name)
	default:
		fmt.Fprintf(stderr, "unsupported shell %q\n", args[0])
		return 2
	}
	return 0
}
