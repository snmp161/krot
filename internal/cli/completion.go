package cli

import (
	"fmt"
	"strings"

	"github.com/snmp161/krot/internal/awg"
)

var completionEnums = []struct {
	flag   string
	values []string
}{
	{"--type", []string{"wg", "awg", "awg2", "awg3"}},
	{"--mimic", awg.MimicProfiles()},
	{"--completion", []string{"bash", "zsh"}},
}

// CompletionScript возвращает скрипт автодополнения для shell (SPEC 4.6).
func CompletionScript(shell string) (string, error) {
	switch shell {
	case "bash":
		return bashCompletion(), nil
	case "zsh":
		return zshCompletion(), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}

func completionOptions(action Action) []string {
	var options []string
	if action == "" {
		for _, a := range allActions {
			options = append(options, "--"+string(a))
		}
		return append(options, "--help", "--version", "--completion")
	}
	for _, p := range paramTable {
		if p.matrix[action] != forbidden {
			options = append(options, "--"+p.name)
		}
	}
	return append(options, "--help")
}

func isCompletionEnum(flag string) bool {
	for _, e := range completionEnums {
		if e.flag == flag {
			return true
		}
	}
	return false
}

func completionPlainValueFlags() []string {
	var flags []string
	for _, p := range paramTable {
		flag := "--" + p.name
		if !p.isBool && !isCompletionEnum(flag) {
			flags = append(flags, flag)
		}
	}
	return flags
}

func actionPattern() string {
	actions := make([]string, 0, len(allActions))
	for _, action := range allActions {
		actions = append(actions, "--"+string(action))
	}
	return strings.Join(actions, "|")
}

func bashCompletion() string {
	var b strings.Builder
	b.WriteString(`# bash completion for krot
_krot() {
    local cur prev action word flag value opts
    local -a matches
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev=""
    if (( COMP_CWORD > 0 )); then
        prev="${COMP_WORDS[COMP_CWORD-1]}"
    fi

    if [[ "$cur" == --*=* ]]; then
        flag="${cur%%=*}"
        value="${cur#*=}"
        case "$flag" in
`)
	for _, e := range completionEnums {
		fmt.Fprintf(&b, "            %s)\n                matches=( $(compgen -W %q -- \"$value\") )\n                ;;\n", e.flag, strings.Join(e.values, " "))
	}
	b.WriteString(`            *)
                return 0
                ;;
        esac
        COMPREPLY=( "${matches[@]/#/$flag=}" )
        return 0
    fi

    case "$prev" in
`)
	for _, e := range completionEnums {
		fmt.Fprintf(&b, "        %s)\n            COMPREPLY=( $(compgen -W %q -- \"$cur\") )\n            return 0\n            ;;\n", e.flag, strings.Join(e.values, " "))
	}
	fmt.Fprintf(&b, "        %s)\n            return 0\n            ;;\n", strings.Join(completionPlainValueFlags(), "|"))
	b.WriteString(`    esac

    action=""
    for word in "${COMP_WORDS[@]:1:COMP_CWORD-1}"; do
        case "$word" in
`)
	fmt.Fprintf(&b, "            %s)\n                action=\"$word\"\n                ;;\n", actionPattern())
	b.WriteString(`            --help|--version|--completion)
                return 0
                ;;
        esac
    done

    case "$action" in
`)
	for _, action := range allActions {
		fmt.Fprintf(&b, "        --%s)\n            opts=%q\n            ;;\n", action, strings.Join(completionOptions(action), " "))
	}
	fmt.Fprintf(&b, "        *)\n            opts=%q\n            ;;\n", strings.Join(completionOptions(""), " "))
	b.WriteString(`    esac
    COMPREPLY=( $(compgen -W "$opts" -- "$cur") )
}

complete -F _krot krot
`)
	return b.String()
}

func zshCompletion() string {
	var b strings.Builder
	b.WriteString(`#compdef krot

_krot() {
    local cur prev action word flag value i
    local -a matches opts
    cur="${words[CURRENT]}"
    prev=""
    if (( CURRENT > 1 )); then
        prev="${words[CURRENT-1]}"
    fi

    if [[ "$cur" == --*=* ]]; then
        flag="${cur%%=*}"
        value="${cur#*=}"
        case "$flag" in
`)
	for _, e := range completionEnums {
		fmt.Fprintf(&b, "            %s)\n                matches=(%s)\n                ;;\n", e.flag, strings.Join(e.values, " "))
	}
	b.WriteString(`            *)
                return 0
                ;;
        esac
        compadd -P "${flag}=" -- "${matches[@]}"
        return 0
    fi

    case "$prev" in
`)
	for _, e := range completionEnums {
		fmt.Fprintf(&b, "        %s)\n            matches=(%s)\n            compadd -- \"${matches[@]}\"\n            return 0\n            ;;\n", e.flag, strings.Join(e.values, " "))
	}
	fmt.Fprintf(&b, "        %s)\n            return 0\n            ;;\n", strings.Join(completionPlainValueFlags(), "|"))
	b.WriteString(`    esac

    action=""
    for (( i = 2; i < CURRENT; i++ )); do
        word="${words[i]}"
        case "$word" in
`)
	fmt.Fprintf(&b, "            %s)\n                action=\"$word\"\n                ;;\n", actionPattern())
	b.WriteString(`            --help|--version|--completion)
                return 0
                ;;
        esac
    done

    case "$action" in
`)
	for _, action := range allActions {
		fmt.Fprintf(&b, "        --%s)\n            opts=(%s)\n            ;;\n", action, strings.Join(completionOptions(action), " "))
	}
	fmt.Fprintf(&b, "        *)\n            opts=(%s)\n            ;;\n", strings.Join(completionOptions(""), " "))
	b.WriteString(`    esac
    compadd -- "${opts[@]}"
}

_krot "$@"
`)
	return b.String()
}
