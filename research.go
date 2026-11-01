package renodash

import (
	"regexp"
	"strings"
)

// Research contains facts extracted from the Dashboard, not a security verdict.
type Research struct {
	Package              string               `json:"package"`
	TargetVersion        string               `json:"target_version"`
	DetectedDependencies []DetectedDependency `json:"detected_dependencies"`
	ReferenceURLs        []string             `json:"reference_urls"`
}

// DetectedDependency preserves the version expression reported by Renovate.
type DetectedDependency struct {
	CurrentVersion string `json:"current_version"`
	Manager        string `json:"manager"`
	File           string `json:"file"`
}

type dependencyRecord struct {
	name string
	DetectedDependency
}

var (
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
