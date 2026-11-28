package command

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const fields = "number,url,state,title,isDraft,headRefOid,updatedAt,statusCheckRollup,mergeable,mergeStateStatus"

// serveGH is a test-only process adapter. It supports fixture requests only and
// never delegates to a real GitHub CLI or makes network requests.
func serveGH(args []string) error {
	log, err := os.OpenFile(os.Getenv("OFFLINE_CALLS"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	err = json.NewEncoder(log).Encode(args)
	log.Close()
	if err != nil {
		return err
	}
	name := ""
	if len(args) == 13 && slices.Equal(args[:5], []string{"api", "graphql", "--hostname", "github.com", "--include"}) && args[8] == "owner=gkzz" && args[10] == "repo=renodash" {
		switch args[12] {
		case "number=30", "number=51", "number=52", "number=53":
			name = "pr-" + strings.TrimPrefix(args[12], "number=") + ".json"
		}
	}

	if name == "" {
		return fmt.Errorf("unsupported offline gh request: %q", args)
	}
	if strings.HasPrefix(name, "pr-") && os.Getenv("OFFLINE_MODE") == "all-prs-fail" {
		return fmt.Errorf("offline fixture: requested lookup failure")
	}
	data, err := os.ReadFile(filepath.Join(os.Getenv("OFFLINE_FIXTURES"), name))
	if err != nil {
		return err
	}
	if name == "pr-51.json" {
		switch mode := os.Getenv("OFFLINE_MODE"); mode {
		case "":
		case "failure":
			return fmt.Errorf("offline fixture: requested lookup failure")
		case "malformed":
			data = []byte("{invalid JSON")
		case "missing":
			data = []byte("{}")
		case "merge", "empty":
			var pr map[string]json.RawMessage
			if err = json.Unmarshal(data, &pr); err != nil {
				return err
			}
			if mode == "empty" {
				pr["statusCheckRollup"] = json.RawMessage("[]")
			} else {
				pr["mergeable"] = json.RawMessage(`"MERGEABLE"`)
				pr["mergeStateStatus"] = json.RawMessage(`"BLOCKED"`)
			}
			data, err = json.Marshal(pr)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported offline mode: %s", mode)
		}
	}
	if strings.HasPrefix(name, "pr-") && !(name == "pr-51.json" && os.Getenv("OFFLINE_MODE") == "malformed") {
		var pr map[string]json.RawMessage
		if err := json.Unmarshal(data, &pr); err != nil {
			return err
		}
		// Use GraphQL-shaped fixtures, independently of the legacy output goldens.
		raw, err := os.ReadFile(filepath.Join(os.Getenv("OFFLINE_FIXTURES"), strings.TrimSuffix(name, ".json")+".graphql.json"))
		if err != nil {
			return err
		}
		var graph struct {
			Data struct {
				Repository struct {
					PullRequest struct{ Commits json.RawMessage }
				}
			}
		}
		if err := json.Unmarshal(raw, &graph); err != nil {
			return err
		}
		pr["commits"] = graph.Data.Repository.PullRequest.Commits
		if name == "pr-51.json" && os.Getenv("OFFLINE_MODE") == "empty" {
			commit := map[string]any{"oid": pr["headRefOid"], "statusCheckRollup": map[string]any{"contexts": map[string]any{"nodes": []any{}, "pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""}}}}
			pr["commits"], err = json.Marshal(map[string]any{"nodes": []any{map[string]any{"commit": commit}}})
			if err != nil {
				return err
			}
		}
		delete(pr, "statusCheckRollup")
		data, err = json.Marshal(map[string]any{"data": map[string]any{"repository": map[string]any{"pullRequest": pr}}})
		if err != nil {
			return err
		}
	}
	if strings.HasPrefix(name, "pr-") {
		fmt.Fprint(os.Stdout, "HTTP/2.0 200 OK\r\nContent-Type: application/json\r\n\r\n")
	}
	_, err = os.Stdout.Write(data)
	return err
}
