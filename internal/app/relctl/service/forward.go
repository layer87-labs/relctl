package service

import (
	"fmt"

	"github.com/layer87-labs/relctl/internal/app/relctl/ces"
	scmportal "github.com/layer87-labs/relctl/internal/app/relctl/scm-portal"
	"github.com/layer87-labs/relctl/internal/pkg/relctlconfig"

	log "github.com/sirupsen/logrus"
)

// ForwardArgs configures `relctl release forward`.
type ForwardArgs struct {
	// ConfigFile is the path to .relctl.yaml; empty means default lookup.
	ConfigFile string
	// From overrides production_branch of the config.
	From string
	// To overrides development_branch of the config.
	To string
	// DryRun only reports what would be done.
	DryRun bool
}

// forwardAction is what has to be done to bring the production branch into the
// development branch.
type forwardAction int

const (
	forwardNothing     forwardAction = iota // development already contains production
	forwardFastForward                      // development is an ancestor of production
	forwardPullRequest                      // the branches have diverged
)

func (a forwardAction) String() string {
	switch a {
	case forwardFastForward:
		return "fast-forward"
	case forwardPullRequest:
		return "pull-request"
	default:
		return "nothing"
	}
}

// decideForward maps GitHub's comparison of production (head) with development
// (base) to an action.
func decideForward(status string) forwardAction {
	switch status {
	case "identical", "behind":
		// development has everything production has
		return forwardNothing
	case "ahead":
		// development is an ancestor of production: a plain fast-forward
		return forwardFastForward
	default:
		// diverged: a merge is needed
		return forwardPullRequest
	}
}

// forwardBranches returns the production and development branch from flags and config.
func forwardBranches(args *ForwardArgs, defaultBranch func() (string, error)) (from, to string, err error) {
	cfgPath := args.ConfigFile
	if cfgPath == "" {
		cfgPath = relctlconfig.DefaultConfigFile
	}
	cfg, err := relctlconfig.Load(cfgPath)
	if err != nil {
		return "", "", fmt.Errorf("loading relctl config: %w", err)
	}
	from, to = args.From, args.To
	if from == "" {
		from = cfg.ProductionBranch
	}
	if to == "" {
		to = cfg.DevelopmentBranch
	}
	if to == "" {
		to, err = defaultBranch()
		if err != nil {
			return "", "", err
		}
	}
	if from == to {
		return "", "", fmt.Errorf("the production branch and the development branch are both %q: nothing to forward. "+
			"Set development_branch in .relctl.yaml if you work on a different branch", from)
	}
	return from, to, nil
}

// ReleaseForward brings the production branch back into the development branch
// after a release. It fast-forwards the development branch if possible. If not
// (the development branch has moved on, or the token may not push to it), it opens
// a pull request, which has to be merged with a merge commit, not squashed.
// The command is idempotent.
func ReleaseForward(args *ForwardArgs) {
	scmLayer, err := scmportal.LoadSCMPortalLayer()
	if err != nil {
		log.Fatalln(err)
	}

	from, to, err := forwardBranches(args, scmLayer.RepositoryDefaultBranch)
	if err != nil {
		log.Fatalln(err)
	}

	status, err := scmLayer.CompareStatus(to, from)
	if err != nil {
		log.Fatalln(err)
	}
	action := decideForward(status)
	log.Infof("comparing %s with %s: %s -> %s", from, to, status, action)

	export := func(result, pr string) {
		vars := []ces.KeyValue{{Name: "RELCTL_FORWARD_RESULT", Value: result}}
		if pr != "" {
			vars = append(vars, ces.KeyValue{Name: "RELCTL_FORWARD_PR", Value: pr})
		}
		if err := scmLayer.CES.ExportAsEnv(vars); err != nil {
			log.Fatalf("could not export env variables: %v", err)
		}
	}

	if action == forwardNothing {
		fmt.Printf("%s already contains %s, nothing to forward\n", to, from)
		export("nothing", "")
		return
	}

	if args.DryRun {
		fmt.Printf("Would forward %s into %s by: %s\n", from, to, action)
		export(action.String(), "")
		return
	}

	if action == forwardFastForward {
		sha, err := scmLayer.BranchSHA(from)
		if err != nil {
			log.Fatalln(err)
		}
		if err := scmLayer.FastForwardBranch(to, sha); err == nil {
			fmt.Printf("Fast-forwarded %s to %s (%s)\n", to, from, shortSha(sha))
			export("fast-forward", "")
			return
		} else {
			// typically a ruleset: only a bypass actor may push to the branch
			log.Warnf("fast-forward of %s not possible (%v), opening a pull request instead", to, err)
		}
	}

	number, err := scmLayer.FindOpenPullRequest(from, to)
	if err != nil {
		log.Fatalln(err)
	}
	if number != 0 {
		fmt.Printf("Pull request #%d from %s into %s is already open\n", number, from, to)
		export("pull-request", fmt.Sprintf("%d", number))
		return
	}
	number, err = scmLayer.CreatePullRequest(from, to, fmt.Sprintf("chore: merge %s back into %s", from, to),
		fmt.Sprintf("Brings the released state of `%s` back into `%s`.\n\n"+
			"**Merge this pull request with a merge commit, do not squash it.** A squash creates a commit that "+
			"`%s` never had and causes conflicts with the next release.", from, to, from))
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("Opened pull request #%d from %s into %s\n", number, from, to)
	export("pull-request", fmt.Sprintf("%d", number))
}
