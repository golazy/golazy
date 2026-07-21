package lazycli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"golazy.dev/lazytui"
)

func writeHelp(w io.Writer, command *node) {
	palette := effectivePalette(command)
	usage := command.path()
	if len(command.children) > 0 {
		usage += " <command>"
	}
	fmt.Fprintf(w, "%s %s\n", colorize(w, palette.Primary, "Usage:"), usage)
	if command.help != "" {
		fmt.Fprintf(w, "\n%s\n", command.help)
	}
	if command.description != "" {
		fmt.Fprintf(w, "\n%s\n", command.description)
	}
	children := sortedChildren(command)
	if command == rootOf(command) && command.autocomplete {
		children = append(children, &node{name: "completion", help: "Generate shell completion scripts"})
		sort.SliceStable(children, func(i, j int) bool { return children[i].name < children[j].name })
	}
	if len(children) > 0 {
		fmt.Fprintf(w, "\n%s\n", colorize(w, palette.Primary, "Commands:"))
		width := 0
		for _, child := range children {
			if len(child.name) > width {
				width = len(child.name)
			}
		}
		for _, child := range children {
			fmt.Fprintf(w, "  %s  %s\n", colorize(w, palette.Secondary, fmt.Sprintf("%-*s", width, child.name)), child.help)
		}
	}
	specs := applicableHelpSpecs(command)
	fmt.Fprintf(w, "\n%s\n", colorize(w, palette.Primary, "Parameters:"))
	if effectiveLogo(command) != "" {
		fmt.Fprintln(w, "  --logo  Print the command logo")
	}
	fmt.Fprintln(w, "  -h, --help  Print command help")
	for _, spec := range specs {
		name := "--" + spec.name
		if spec.short != "" {
			name = "-" + spec.short + ", " + name
		}
		fmt.Fprintf(w, "  %-18s %s\n", name, spec.help)
	}
}

func applicableHelpSpecs(selected *node) []flagSpec {
	plan, err := buildPlan(selected)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var specs []flagSpec
	for _, typeOf := range plan.paramTypes() {
		for _, spec := range specsFor(typeOf) {
			if spec.positional || seen[spec.name] {
				continue
			}
			seen[spec.name] = true
			specs = append(specs, spec)
		}
	}
	sort.SliceStable(specs, func(i, j int) bool { return specs[i].name < specs[j].name })
	return specs
}

func rootOf(n *node) *node {
	for n.parent != nil {
		n = n.parent
	}
	return n
}

func writeError(w io.Writer, palette lazytui.Palette, err error) {
	fmt.Fprintf(w, "%s %v\n", colorize(w, palette.Error, "error:"), err)
}

func colorize(w io.Writer, color lazytui.Color, text string) string {
	if color.IsZero() || !colorEnabled(w) {
		return text
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", color.R, color.G, color.B, text)
}

func colorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
