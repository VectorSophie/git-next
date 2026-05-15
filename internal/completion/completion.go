package completion

// Bash returns a bash completion script for git-next.
func Bash() string {
	return bashScript
}

// Zsh returns a zsh completion script for git-next.
func Zsh() string {
	return zshScript
}

// Fish returns a fish completion script for git-next.
func Fish() string {
	return fishScript
}

const ruleIDs = "R001 R002 R003 R004 R005 R006 R007 R008 R009 R010 R011 R020 R021 R022 R030 R031 R032 R033 R034 R035 R036 R037 R038 R039 R040 R041 R042 R043 R044 R045 R046 R047 R048 R049 R050 R051 R052 R053 R054 R055 R056 R057 R058"

const bashScript = `# git-next bash completion
# Add to ~/.bashrc or source from your bash_completion.d directory:
#   eval "$(git-next completion bash)"

_git_next_complete() {
    local cur prev words cword
    _init_completion 2>/dev/null || {
        cur="${COMP_WORDS[COMP_CWORD]}"
        prev="${COMP_WORDS[COMP_CWORD-1]}"
    }

    local subcommands="guard hook ci explain rules completion"
    local global_flags="--version --all --json --compact --agent --explain --debug --action --config"

    # Which subcommand are we in?
    local sub=""
    for w in "${COMP_WORDS[@]:1}"; do
        case "$w" in
        guard|hook|ci|explain|rules|completion) sub="$w"; break ;;
        esac
    done

    case "$sub" in
    hook)
        case "$prev" in
        --hook) COMPREPLY=($(compgen -W "pre-commit pre-push commit-msg post-merge" -- "$cur")) ;;
        *)      COMPREPLY=($(compgen -W "install uninstall list --hook" -- "$cur")) ;;
        esac
        ;;
    ci)
        case "$prev" in
        --fail-on) COMPREPLY=($(compgen -W "critical high medium low all" -- "$cur")) ;;
        --format)  COMPREPLY=($(compgen -W "github human" -- "$cur")) ;;
        *)         COMPREPLY=($(compgen -W "--fail-on --format" -- "$cur")) ;;
        esac
        ;;
    explain)
        COMPREPLY=($(compgen -W "` + ruleIDs + `" -- "$cur"))
        ;;
    rules)
        COMPREPLY=($(compgen -W "list" -- "$cur"))
        ;;
    completion)
        COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur"))
        ;;
    guard)
        # After --, complete normal git subcommands
        COMPREPLY=($(compgen -W "push pull commit reset rebase merge checkout switch cherry-pick" -- "$cur"))
        ;;
    *)
        if [[ "$cur" == -* ]]; then
            COMPREPLY=($(compgen -W "$global_flags" -- "$cur"))
        else
            COMPREPLY=($(compgen -W "$subcommands" -- "$cur"))
        fi
        ;;
    esac
}

complete -F _git_next_complete git-next
`

const zshScript = `# git-next zsh completion
# Add to your ~/.zshrc:
#   eval "$(git-next completion zsh)"
# Or place this file in a directory on your $fpath.

_git_next() {
    local state

    _arguments \
        '1: :->subcommand' \
        '*: :->args'

    case "$state" in
    subcommand)
        local subcommands
        subcommands=(
            'guard:Check whether a git command is safe'
            'hook:Install or remove git hooks'
            'ci:Run as a CI safety gate'
            'explain:Explain a specific rule'
            'rules:List all rules'
            'completion:Print shell completion script'
        )
        _describe 'subcommand' subcommands
        _arguments \
            '(-v --version)'{-v,--version}'[Show version]' \
            '(-a --all)'{-a,--all}'[Show suppressed advice]' \
            '--json[Output in JSON format]' \
            '--compact[One-line summary]' \
            '--agent[Machine-readable JSON for agents]' \
            '--explain[Show explanation for each rule]' \
            '--debug[Show repo state]' \
            '--action[Interactive mode]'
        ;;
    args)
        case "${words[2]}" in
        hook)
            _arguments \
                '1: :(install uninstall list)' \
                '--hook[Hook name]:hook:(pre-commit pre-push commit-msg post-merge)'
            ;;
        ci)
            _arguments \
                '--fail-on[Minimum severity]:severity:(critical high medium low all)' \
                '--format[Output format]:format:(github human)'
            ;;
        explain)
            local rule_ids
            rule_ids=(` + ruleIDs + `)
            _describe 'rule ID' rule_ids
            ;;
        rules)
            _arguments '1: :(list)'
            ;;
        completion)
            _arguments '1: :(bash zsh fish)'
            ;;
        esac
        ;;
    esac
}

compdef _git_next git-next
`

const fishScript = `# git-next fish completion
# Add to your fish config:
#   git-next completion fish | source

# Disable file completion
complete -c git-next -f

# Subcommands
complete -c git-next -n '__fish_use_subcommand' -a guard       -d 'Check whether a git command is safe'
complete -c git-next -n '__fish_use_subcommand' -a hook        -d 'Install or remove git hooks'
complete -c git-next -n '__fish_use_subcommand' -a ci          -d 'Run as a CI safety gate'
complete -c git-next -n '__fish_use_subcommand' -a explain     -d 'Explain a specific rule'
complete -c git-next -n '__fish_use_subcommand' -a rules       -d 'List all rules with metadata'
complete -c git-next -n '__fish_use_subcommand' -a completion  -d 'Print shell completion script'

# Global flags
complete -c git-next -n '__fish_use_subcommand' -l version  -d 'Show version'
complete -c git-next -n '__fish_use_subcommand' -s v        -d 'Show version'
complete -c git-next -n '__fish_use_subcommand' -l all      -d 'Show suppressed advice'
complete -c git-next -n '__fish_use_subcommand' -s a        -d 'Show suppressed advice'
complete -c git-next -n '__fish_use_subcommand' -l json     -d 'JSON output'
complete -c git-next -n '__fish_use_subcommand' -l compact  -d 'One-line summary'
complete -c git-next -n '__fish_use_subcommand' -l agent    -d 'Machine-readable JSON for agents'
complete -c git-next -n '__fish_use_subcommand' -l explain  -d 'Show explanation for each rule'
complete -c git-next -n '__fish_use_subcommand' -l action   -d 'Interactive mode'
complete -c git-next -n '__fish_use_subcommand' -l debug    -d 'Show repo state'

# hook subcommand
complete -c git-next -n '__fish_seen_subcommand_from hook' -a install   -d 'Install a hook'
complete -c git-next -n '__fish_seen_subcommand_from hook' -a uninstall -d 'Remove a hook'
complete -c git-next -n '__fish_seen_subcommand_from hook' -a list      -d 'List hook status'
complete -c git-next -n '__fish_seen_subcommand_from hook' -l hook      -d 'Hook name' -a 'pre-commit pre-push commit-msg post-merge'

# ci subcommand
complete -c git-next -n '__fish_seen_subcommand_from ci' -l fail-on -d 'Minimum severity' -a 'critical high medium low all'
complete -c git-next -n '__fish_seen_subcommand_from ci' -l format  -d 'Output format'    -a 'github human'

# explain subcommand — list all rule IDs
complete -c git-next -n '__fish_seen_subcommand_from explain' -a '` + ruleIDs + `'

# rules subcommand
complete -c git-next -n '__fish_seen_subcommand_from rules' -a list -d 'List all rules'

# completion subcommand
complete -c git-next -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish'
`
