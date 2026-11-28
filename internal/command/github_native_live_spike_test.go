package command

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestNativeSpikeLive(t *testing.T) {
	if os.Getenv("RENODASH_NATIVE_SPIKE_LIVE") != "1" {
		t.Skip("set RENODASH_NATIVE_SPIKE_LIVE=1 to compare live read-only requests")
	}
	gh, err := exec.LookPath("gh")
	if err != nil {
		t.Fatal("gh is required for the comparison baseline")
	}
	// Ask gh for the credential it would use, without displaying or exporting it.
	authCtx, cancelAuth := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelAuth()
	credential, err := exec.CommandContext(authCtx, gh, "auth", "token", "--hostname", "github.com").Output()
	token := strings.TrimSpace(string(credential))
	if err != nil || token == "" {
		t.Fatal("GitHub authentication failed; run gh auth login --hostname github.com")
	}
	t.Setenv("GH_HOST", "github.com")
	t.Setenv("GH_DEBUG", "1") // Count request headlines without enabling verbose HTTP logging.
	t.Setenv("GH_NO_UPDATE_NOTIFIER", "1")
	t.Setenv("GH_TOKEN", token)
	const repo = "gkzz/renodash"
	client := newNativeSpike()
	client.token = token
	var requests int
	transport := client.http.Transport
	client.http.Transport = spikeRoundTrip(func(r *http.Request) (*http.Response, error) {
		requests++
		return transport.RoundTrip(r)
	})
	baseline := func(ctx context.Context, args ...string) ([]byte, time.Duration, int) {
		t.Helper()
		start := time.Now()
		var diagnostics bytes.Buffer
		cmd := exec.CommandContext(ctx, gh, args...)
		cmd.Stderr = &diagnostics
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("gh baseline failed for %s", args[0])
		}
		calls := 0
		for line := range strings.SplitSeq(diagnostics.String(), "\n") {
			if strings.HasPrefix(line, "* Request to https://api.github.com/") {
				calls++
			}
		}
		return out, time.Since(start).Round(time.Millisecond), calls
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	issueData, ghTime, ghCalls := baseline(ctx, "api", "--hostname", "github.com", "repos/"+repo+"/issues/5")
	start := time.Now()
	nativeData, err := client.FetchIssue(ctx, repo, 5)
	if err != nil {
		t.Fatal(err)
	}
	var wantIssue, gotIssue githubIssue
	if json.Unmarshal(issueData, &wantIssue) != nil || json.Unmarshal(nativeData, &gotIssue) != nil || wantIssue.Body == nil {
		t.Fatal("invalid Issue comparison response")
	}
	var issueDifferences []string
	if wantIssue.Number != gotIssue.Number {
		issueDifferences = append(issueDifferences, "number")
	}
	if wantIssue.HTMLURL != gotIssue.HTMLURL {
		issueDifferences = append(issueDifferences, "html_url")
	}
	if wantIssue.UpdatedAt != gotIssue.UpdatedAt {
		issueDifferences = append(issueDifferences, "updated_at")
	}
	if !cmp.Equal(wantIssue.Body, gotIssue.Body) {
		issueDifferences = append(issueDifferences, "body")
	}
	if len(issueDifferences) != 0 {
		t.Fatalf("Issue comparison inconclusive due to differing observations: fields=%v", issueDifferences)
	}
	t.Logf("Issue #5 matched: native=%s gh=%s native requests=%d gh requests=%d", time.Since(start).Round(time.Millisecond), ghTime, requests, ghCalls)

	const fields = "number,url,state,title,isDraft,headRefOid,updatedAt,mergeable,mergeStateStatus,statusCheckRollup"
	for _, number := range []int{30, 51, 52, 53} {
		var previous []map[string]any
		for attempt := range 2 {
			before := requests
			start := time.Now()
			nativeData, err := client.FetchPR(ctx, repo, number)
			nativeTime := time.Since(start).Round(time.Millisecond)
			if err != nil {
				t.Fatalf("native PR #%d: %v", number, err)
			}
			start = time.Now()
			productionData, err := (ghClient{}).FetchPR(ctx, repo, number)
			productionTime := time.Since(start).Round(time.Millisecond)
			if err != nil {
				t.Fatalf("ghClient PR #%d: %v", number, err)
			}
			viewData, viewTime, viewCalls := baseline(ctx, "pr", "view", fmt.Sprint(number), "--repo", repo, "--json", fields)

			sources := []string{"ghClient", "native", "gh pr view"}
			var observations []map[string]any
			var differences []string
			production, err := decodePullRequest(productionData, repo, number, "")
			if err != nil {
				t.Fatalf("ghClient PR #%d: invalid metadata", number)
			}
			for i, data := range [][]byte{productionData, nativeData, viewData} {
				var fieldsByName map[string]any
				if json.Unmarshal(data, &fieldsByName) != nil {
					t.Fatalf("%s PR #%d: invalid JSON", sources[i], number)
				}
				observations = append(observations, fieldsByName)
				if i == 0 {
					continue
				}
				metadata, err := decodePullRequest(data, repo, number, "")
				if err != nil {
					t.Fatalf("%s PR #%d: invalid metadata", sources[i], number)
				}
				var changed []string
				for key := range strings.SplitSeq(fields, ",") {
					got, gotPresent := fieldsByName[key]
					want, wantPresent := observations[0][key]
					if gotPresent != wantPresent || !cmp.Equal(got, want) {
						changed = append(changed, key)
					}
				}
				slices.Sort(changed)
				normalizedEqual := cmp.Equal(metadata, production, spikeJSONValue)
				if len(changed) != 0 || !normalizedEqual {
					differences = append(differences, fmt.Sprintf("%s vs ghClient: fields=%v normalized_equal=%t", sources[i], changed, normalizedEqual))
				}
			}
			if attempt == 1 && !cmp.Equal(observations, previous) {
				t.Errorf("PR #%d comparison inconclusive: live data changed between observations", number)
				break
			}
			if len(differences) != 0 {
				if attempt == 0 {
					previous = observations
					continue
				}
				left, right, view := observations[0], observations[1], observations[2]
				if left["headRefOid"] != right["headRefOid"] || left["headRefOid"] != view["headRefOid"] ||
					left["updatedAt"] != right["updatedAt"] || left["updatedAt"] != view["updatedAt"] {
					t.Errorf("PR #%d comparison inconclusive: different heads or update times (%s)", number, strings.Join(differences, "; "))
				} else {
					t.Errorf("PR #%d compatibility mismatch across two observations: %s", number, strings.Join(differences, "; "))
				}
				break
			}
			productionFields := observations[0]
			checks, _ := productionFields["statusCheckRollup"].([]any)
			t.Logf("PR #%d matched: state=%v draft=%v mergeable=%v mergeStateStatus=%v checks=%d native=%s ghClient=%s gh pr view=%s native requests=%d gh pr view requests=%d",
				number, productionFields["state"], productionFields["isDraft"], productionFields["mergeable"], productionFields["mergeStateStatus"], len(checks),
				nativeTime, productionTime, viewTime, requests-before, viewCalls)
			break
		}
	}
	// Compare the alternate REST merge fields at a matching head.
	restData, _, _ := baseline(ctx, "api", "--hostname", "github.com", "repos/"+repo+"/pulls/30")
	var rest struct {
		Head           struct{ SHA string }
		State          string
		Mergeable      *bool
		MergeableState string `json:"mergeable_state"`
	}
	if json.Unmarshal(restData, &rest) != nil {
		t.Fatal("invalid REST sample")
	}
	graphqlData, err := client.FetchPR(ctx, repo, 30)
	if err != nil {
		t.Fatal(err)
	}
	var graphql struct {
		HeadRefOID       string
		Mergeable        string
		MergeStateStatus string
	}
	if json.Unmarshal(graphqlData, &graphql) != nil || rest.Head.SHA == "" || graphql.HeadRefOID != rest.Head.SHA {
		t.Fatal("REST sample head changed or was missing")
	}
	mergeable := "null"
	if rest.Mergeable != nil {
		mergeable = fmt.Sprint(*rest.Mergeable)
	}
	t.Logf("PR #30 REST sample at matching head: state=%s mergeable=%s mergeable_state=%s; GraphQL mergeable=%s mergeStateStatus=%s",
		rest.State, mergeable, rest.MergeableState, graphql.Mergeable, graphql.MergeStateStatus)

	client.token = "renodash-intentionally-invalid-token"
	data, err := client.FetchIssue(ctx, repo, 5)
	assertSpikeError(t, data, err, "authentication", false, 0)
	if strings.Contains(err.Error(), client.token) {
		t.Fatal("authentication error contains token")
	}
	t.Log("GitHub rejected the deliberately invalid token with a sanitized authentication error")
}
