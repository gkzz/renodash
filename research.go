package renodash

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
)

// Research contains facts extracted from the Dashboard, not a security verdict.
type Research struct {
	Packages             []GroupPackageResearch `json:"packages,omitempty"`
	Package              string                 `json:"package"`
	TargetVersion        string                 `json:"target_version"`
	DetectedDependencies []DetectedDependency   `json:"detected_dependencies"`
	ReferenceURLs        []string               `json:"reference_urls"`
}

// GroupPackageResearch lists explicit group members without inferring branch targets.
type GroupPackageResearch struct {
	Package              string                    `json:"package"`
	TargetVersion        string                    `json:"target_version"`
	DetectedDependencies []GroupDetectedDependency `json:"detected_dependencies"`
}

// GroupDetectedDependency keeps candidates attached to their source record.
type GroupDetectedDependency struct {
	DetectedDependency
	AvailableUpdates []string `json:"available_updates"`
}

// MarshalJSON preserves the single-update schema while adding a grouped shape.
func (r Research) MarshalJSON() ([]byte, error) {
	if r.Packages != nil {
		return json.Marshal(struct {
			Packages      []GroupPackageResearch `json:"packages"`
			ReferenceURLs []string               `json:"reference_urls"`
		}{r.Packages, r.ReferenceURLs})
	}
	type single Research
	return json.Marshal(single(r))
}

// DetectedDependency preserves the version expression reported by Renovate.
type DetectedDependency struct {
	CurrentVersion string `json:"current_version"`
	Manager        string `json:"manager"`
	File           string `json:"file"`
}

type dependencyRecord struct {
	name    string
	updates []string
	DetectedDependency
}

var (
	groupTitlePattern   = regexp.MustCompile(`(?i)^((?:.*?: )?update [^\r\n]+) \((.+)\)$`)
	groupMemberPattern  = regexp.MustCompile("^`([^` \\t]+)`$")
	updatesPattern      = regexp.MustCompile(" → \\[Updates: (`[^`]+`(?:, `[^`]+`)*)\\]$")
	candidatePattern    = regexp.MustCompile("`([^`]+)`")
	updateTitlePattern  = regexp.MustCompile("(?i)^(?:.*?: )?update (?:dependency )?(`?[^` \\t]+`?)(?: docker tag)? to (`?[^` \\t]+`?)$")
	summaryPattern      = regexp.MustCompile(`^<details><summary>(.+?)</summary>$`)
	summaryCountPattern = regexp.MustCompile(` \([0-9]+\)$`)
	dependencyPattern   = regexp.MustCompile("^ {0,3}- `([^` ]+) ([^`]+)`(?:.*)$")
	referencePattern    = regexp.MustCompile(`\[([^\]]*)\]\((https?://[^\s)]+)\)`)
)

// enrichResearch only associates exact package names. Group names, aliases and
// upstream repository URLs cannot be reliably inferred from Dashboard text.
func enrichResearch(d *Dashboard, records []dependencyRecord) {
	for i := range d.Items {
		item := &d.Items[i]
		plainTitle := referencePattern.ReplaceAllString(item.Title, "$1")
		match := updateTitlePattern.FindStringSubmatch(plainTitle)
		if match == nil {
			item.Research = groupedResearch(item.Title, records)
			continue
		}
		research := &Research{
			Package: strings.Trim(match[1], "`"), TargetVersion: strings.Trim(match[2], "`"),
			DetectedDependencies: []DetectedDependency{}, ReferenceURLs: []string{},
		}
		for _, record := range records {
			if record.name != research.Package {
				continue
			}
			found := false
			for _, existing := range research.DetectedDependencies {
				if existing == record.DetectedDependency {
					found = true
					break
				}
			}
			if !found {
				research.DetectedDependencies = append(research.DetectedDependencies, record.DetectedDependency)
			}
		}
		for _, link := range referencePattern.FindAllStringSubmatch(item.Title, -1) {
			research.ReferenceURLs = append(research.ReferenceURLs, link[2])
		}
		item.Research = research
	}
}

func groupedResearch(title string, records []dependencyRecord) *Research {
	match := groupTitlePattern.FindStringSubmatch(title)
	if match == nil {
		return nil
	}
	// A single-update title with a parenthetical note does not identify group members.
	plainPrefix := referencePattern.ReplaceAllString(match[1], "$1")
	if updateTitlePattern.MatchString(plainPrefix) {
		return nil
	}
	research := &Research{Packages: []GroupPackageResearch{}, ReferenceURLs: []string{}}
	seen := map[string]bool{}
	for _, member := range strings.Split(match[2], ",") {
		name := groupMemberPattern.FindStringSubmatch(strings.TrimSpace(member))
		if name == nil {
			return nil
		}
		if seen[name[1]] {
			continue
		}
		seen[name[1]] = true
		pkg := GroupPackageResearch{Package: name[1], TargetVersion: "unknown", DetectedDependencies: []GroupDetectedDependency{}}
		for _, record := range records {
			if record.name != pkg.Package {
				continue
			}
			dep := GroupDetectedDependency{DetectedDependency: record.DetectedDependency, AvailableUpdates: append([]string{}, record.updates...)}
			duplicate := false
			for _, existing := range pkg.DetectedDependencies {
				if existing.DetectedDependency == dep.DetectedDependency && slices.Equal(existing.AvailableUpdates, dep.AvailableUpdates) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				pkg.DetectedDependencies = append(pkg.DetectedDependencies, dep)
			}
		}
		research.Packages = append(research.Packages, pkg)
	}
	for _, link := range referencePattern.FindAllStringSubmatch(title, -1) {
		research.ReferenceURLs = append(research.ReferenceURLs, link[2])
	}
	return research
}
