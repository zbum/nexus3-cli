package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var debugEnabled = os.Getenv("NEXUS_CLI_DEBUG") != ""

// Client talks to the Nexus Repository 3 REST API (/service/rest/v1/...).
// nexus_host is the main Nexus URL (e.g. http://nexus:8081) — no Docker
// connector is required because all operations go through the REST API.
type Client struct {
	baseURL    string // e.g. http://nexus:8081
	restURL    string // baseURL + /service/rest
	repository string
	username   string
	password   string
	http       *http.Client
}

func New(host, repository, username, password string) *Client {
	h := strings.TrimRight(host, "/")
	return &Client{
		baseURL:    h,
		restURL:    h + "/service/rest",
		repository: repository,
		username:   username,
		password:   password,
		http:       &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) Repository() string { return c.repository }

func (c *Client) do(method, full, accept string) (*http.Response, error) {
	req, err := http.NewRequest(method, full, nil)
	if err != nil {
		return nil, err
	}
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if debugEnabled {
		log.Printf("→ %s %s", method, full)
	}
	resp, err := c.http.Do(req)
	if debugEnabled && resp != nil {
		log.Printf("← %d %s", resp.StatusCode, full)
	}
	return resp, err
}

func hintFor(status int) string {
	switch status {
	case http.StatusNotFound:
		return "\n  hint: is nexus_host correct? It should be the main Nexus URL (e.g. http://nexus:8081)." +
			"\n        Is the repository name spelled exactly as it appears in the Nexus UI?"
	case http.StatusUnauthorized:
		return "\n  hint: credentials rejected. Check username/password with `nexus3-cli configure`."
	case http.StatusForbidden:
		return "\n  hint: the user lacks privileges — needs nx-repository-view-docker-<repo>-* (read for list/info, delete for delete)."
	case http.StatusUnprocessableEntity:
		return "\n  hint: malformed request. Common cause: a malformed component id or missing 'repository' parameter."
	}
	return ""
}

// errorFrom reads the body, closes it, and returns a formatted error with URL and hint.
func errorFrom(resp *http.Response, method, full string) error {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return fmt.Errorf("%s %s returned %d: %s%s",
		method, full, resp.StatusCode,
		strings.TrimSpace(string(body)), hintFor(resp.StatusCode))
}

// ListRepositories returns the list of all repositories visible to the user.
// Used to sanity-check that the target repository exists and is a docker format.
func (c *Client) ListRepositories() ([]Repository, error) {
	full := c.restURL + "/v1/repositories"
	resp, err := c.do(http.MethodGet, full, "application/json")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errorFrom(resp, http.MethodGet, full)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var repos []Repository
	if err := json.Unmarshal(body, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

// searchComponents paginates /v1/search scoped to the configured repository.
// Extra query params can narrow the result set (e.g. name, version).
func (c *Client) searchComponents(extra url.Values) ([]Component, error) {
	q := url.Values{}
	q.Set("repository", c.repository)
	q.Set("format", "docker")
	for k, vs := range extra {
		for _, v := range vs {
			q.Add(k, v)
		}
	}

	var all []Component
	token := ""
	for {
		if token != "" {
			q.Set("continuationToken", token)
		} else {
			q.Del("continuationToken")
		}
		full := c.restURL + "/v1/search?" + q.Encode()
		resp, err := c.do(http.MethodGet, full, "application/json")
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, errorFrom(resp, http.MethodGet, full)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		var page pageComponent
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Items...)
		if page.ContinuationToken == "" {
			break
		}
		token = page.ContinuationToken
	}
	return all, nil
}

// ListImages returns unique Docker image names in the repository.
func (c *Client) ListImages() ([]string, error) {
	comps, err := c.searchComponents(nil)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(comps))
	names := make([]string, 0, len(comps))
	for _, comp := range comps {
		if _, ok := seen[comp.Name]; ok {
			continue
		}
		seen[comp.Name] = struct{}{}
		names = append(names, comp.Name)
	}
	return names, nil
}

// ListTags returns the versions (Docker tags) for the given image name.
func (c *Client) ListTags(name string) ([]string, error) {
	q := url.Values{}
	q.Set("name", name)
	comps, err := c.searchComponents(q)
	if err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(comps))
	for _, comp := range comps {
		if comp.Name != name {
			continue // defensive: search is a wildcard-style match
		}
		tags = append(tags, comp.Version)
	}
	return tags, nil
}

// GetComponent returns the component matching (name, tag) exactly.
func (c *Client) GetComponent(name, tag string) (*Component, error) {
	q := url.Values{}
	q.Set("name", name)
	q.Set("version", tag)
	comps, err := c.searchComponents(q)
	if err != nil {
		return nil, err
	}
	for i := range comps {
		if comps[i].Name == name && comps[i].Version == tag {
			return &comps[i], nil
		}
	}
	return nil, fmt.Errorf("component %s:%s not found in repository %q", name, tag, c.repository)
}

// ListComponentsByName returns all components matching the given image name.
func (c *Client) ListComponentsByName(name string) ([]Component, error) {
	q := url.Values{}
	q.Set("name", name)
	comps, err := c.searchComponents(q)
	if err != nil {
		return nil, err
	}
	out := comps[:0]
	for _, comp := range comps {
		if comp.Name == name {
			out = append(out, comp)
		}
	}
	return out, nil
}

// DeleteComponent deletes the component (image:tag) by its Nexus id. All
// assets belonging to the component are removed. Blobs stay on disk until a
// compact-blob-store task runs.
func (c *Client) DeleteComponent(id string) error {
	full := c.restURL + "/v1/components/" + url.PathEscape(id)
	resp, err := c.do(http.MethodDelete, full, "application/json")
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return errorFrom(resp, http.MethodDelete, full)
	}
	resp.Body.Close()
	return nil
}
