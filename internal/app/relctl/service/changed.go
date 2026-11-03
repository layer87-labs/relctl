package service

import (
	"fmt"
	"strconv"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/layer87-labs/relctl/internal/app/relctl/ces"
	"github.com/layer87-labs/relctl/internal/pkg/changedpaths"
	"github.com/layer87-labs/relctl/internal/pkg/tools"

	log "github.com/sirupsen/logrus"
)

// ChangedArgs configures `relctl release changed`.
type ChangedArgs struct {
	// ConfigFile is the path to .relctl.yaml; empty means default lookup.
	ConfigFile string
	// Component is the component whose paths are compared.
	Component string
	// Since overrides the commit or tag to compare with; empty means the latest
	// release of the component.
	Since string
}

// latestReleaseTag returns the highest tag that is a release: release candidates
// (X.Y.Z-rc.N) are not.
func latestReleaseTag(tags []tools.TaggedVersion) (tools.TaggedVersion, bool) {
	for i := len(tags) - 1; i >= 0; i-- {
		if tags[i].Version.PreRelease == "" {
			return tags[i], true
		}
	}
	return tools.TaggedVersion{}, false
}

// ReleaseChanged reports whether a file that belongs to a component changed since
// the component's latest release, and exports the answer.
func ReleaseChanged(args *ChangedArgs) {
	if args.Component == "" {
		log.Fatalln("release changed needs --component: it compares the paths of a component with its latest release")
	}
	eff, err := loadEffective(args.ConfigFile, args.Component)
	if err != nil {
		log.Fatalln(err)
	}
	if len(eff.Paths) == 0 {
		log.Fatalf("component %q has no paths: list them under components.%s.paths in .relctl.yaml", args.Component, args.Component)
	}

	cesCfg, _, _, err := ces.DetectCes()
	if err != nil {
		log.Fatalln(err)
	}

	repo, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		log.Fatalf("could not open the git repository: %v", err)
	}

	var (
		base        plumbing.Hash
		hasBase     bool
		baseLabel   string
		latestValue string
	)
	if args.Since != "" {
		h, err := repo.ResolveRevision(plumbing.Revision(args.Since))
		if err != nil {
			log.Fatalf("could not resolve --since %q: %v", args.Since, err)
		}
		base, hasBase, baseLabel = *h, true, args.Since
	} else {
		tags, err := tools.GetGitTagsUpToHeadWithPrefix(repo, eff.TagPrefix)
		if err != nil {
			log.Fatalln(err)
		}
		if latest, ok := latestReleaseTag(tags); ok {
			h, err := repo.ResolveRevision(plumbing.Revision(latest.Tag))
			if err != nil {
				log.Fatalf("could not resolve the tag %q: %v", latest.Tag, err)
			}
			base, hasBase, baseLabel = *h, true, latest.Tag
			latestValue = latest.Version.String()
		}
	}

	changed := true
	var matched []string
	if !hasBase {
		fmt.Printf("Component %s has no release yet: changed\n", args.Component)
	} else {
		files, err := changedpaths.Files(repo, base)
		if err != nil {
			log.Fatalln(err)
		}
		matched, err = changedpaths.Match(files, eff.Paths)
		if err != nil {
			log.Fatalln(err)
		}
		changed = len(matched) > 0
		fmt.Printf("Component %s since %s: %d of %d changed files belong to it\n", args.Component, baseLabel, len(matched), len(files))
		for _, f := range matched {
			fmt.Printf("  %s\n", f)
		}
	}
	fmt.Printf("Changed: %t\n", changed)

	vars := []ces.KeyValue{
		{Name: "RELCTL_CHANGED", Value: strconv.FormatBool(changed)},
		{Name: "RELCTL_CHANGED_FILES", Value: strconv.Itoa(len(matched))},
		{Name: "RELCTL_COMPONENT", Value: args.Component},
		{Name: "RELCTL_LATEST_VERSION", Value: latestValue},
	}
	if err := cesCfg.ExportAsEnv(vars); err != nil {
		log.Fatalf("could not export env variables: %v", err)
	}
}
