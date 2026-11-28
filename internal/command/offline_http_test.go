package command

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func offlinePR(w http.ResponseWriter, req *http.Request) error {
	var request graphQLRequest
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		return err
	}
	target := request.Variables
	if req.Method != http.MethodPost || req.Host != "api.github.com" || req.URL.Path != "/graphql" ||
		request.Query != prQuery || target.Owner != "gkzz" || target.Repo != "renodash" || target.Cursor != "" {
		return fmt.Errorf("unexpected offline GraphQL request")
	}
	switch target.Number {
	case 30, 51, 52, 53:
	default:
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	if path := os.Getenv("OFFLINE_CALLS"); path != "" {
		log, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		err = json.NewEncoder(log).Encode(target.Number)
		log.Close()
		if err != nil {
			return err
		}
	}
	mode := os.Getenv("OFFLINE_MODE")
	if mode == "all-prs-fail" || (mode == "failure" && target.Number == 51) {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	name := fmt.Sprintf("pr-%d.json", target.Number)
	data, err := os.ReadFile(filepath.Join(root, "testdata/pr-metadata", name))
	if err != nil {
		return err
	}
	if name == "pr-51.json" {
		switch mode {
		case "":
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
	if target.Number != 51 || mode != "malformed" {
		var pr map[string]json.RawMessage
		if err := json.Unmarshal(data, &pr); err != nil {
			return err
		}
		// Use GraphQL-shaped fixtures, independently of the legacy output goldens.
		raw, err := os.ReadFile(filepath.Join(root, "testdata/pr-metadata", strings.TrimSuffix(name, ".json")+".graphql.json"))
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
		if target.Number == 51 && mode == "empty" {
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
	_, err = w.Write(data)
	return err
}
