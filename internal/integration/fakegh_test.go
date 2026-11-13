package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

const fields = "number,url,state,title,isDraft,headRefOid,updatedAt,statusCheckRollup"

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
	if reflect.DeepEqual(args, []string{"api", "repos/gkzz/renodash/issues/5"}) {
		name = "dashboard-issue.json"
	} else if len(args) == 7 && args[0] == "pr" && args[1] == "view" &&
		reflect.DeepEqual(args[3:], []string{"--repo", "gkzz/renodash", "--json", fields}) {
		switch args[2] {
		case "30", "51", "52", "53":
			name = "pr-" + args[2] + ".json"
		}
	}
	if name == "" {
		return fmt.Errorf("unsupported offline gh request: %q", args)
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
		case "empty":
			var pr map[string]json.RawMessage
			if err = json.Unmarshal(data, &pr); err != nil {
				return err
			}
			pr["statusCheckRollup"] = json.RawMessage("[]")
			data, err = json.Marshal(pr)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported offline mode: %s", mode)
		}
	}
	_, err = os.Stdout.Write(data)
	return err
}
