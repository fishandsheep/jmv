package okm

import (
	"context"
	"html"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

var hrefPattern = regexp.MustCompile(`href="([^"]+)"`)

const userAgent = "okm"

type MirrorClient struct {
	BaseURL string
	HTTP    *http.Client
}

func NewMirrorClient(baseURL string) MirrorClient {
	return MirrorClient{BaseURL: trimSlash(baseURL), HTTP: http.DefaultClient}
}

func (m MirrorClient) Majors(ctx context.Context) ([]string, error) {
	body, err := m.getText(ctx, m.BaseURL+"/")
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	for _, href := range hrefs(body) {
		major := strings.TrimSuffix(href, "/")
		if major == "" {
			continue
		}
		allDigits := true
		for _, r := range major {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			seen[major] = true
		}
	}

	majors := make([]string, 0, len(seen))
	for major := range seen {
		majors = append(majors, major)
	}
	sort.Slice(majors, func(i, j int) bool {
		if len(majors[i]) != len(majors[j]) {
			return len(majors[i]) < len(majors[j])
		}
		return majors[i] < majors[j]
	})
	return majors, nil
}

func (m MirrorClient) List(ctx context.Context, rt Runtime, platform Platform) ([]Release, error) {
	majors, err := m.Majors(ctx)
	if err != nil {
		return nil, err
	}

	var releases []Release
	for _, major := range majors {
		release, err := m.Resolve(ctx, rt, major, platform)
		if err == nil {
			releases = append(releases, release)
		}
	}
	return releases, nil
}

func (m MirrorClient) Resolve(ctx context.Context, rt Runtime, major string, platform Platform) (Release, error) {
	indexURL := m.indexURL(rt, major, platform)
	body, err := m.getText(ctx, indexURL)
	if err != nil {
		return Release{}, err
	}

	var candidates []string
	for _, href := range hrefs(body) {
		name := strings.TrimPrefix(href, "./")
		if strings.Contains(name, "/") {
			continue
		}
		if strings.HasPrefix(name, "OpenJDK") &&
			strings.Contains(name, "-"+string(rt)+"_") &&
			strings.Contains(name, "_"+platform.Arch+"_"+platform.OS+"_") &&
			strings.HasSuffix(name, platform.Ext) {
			candidates = append(candidates, name)
		}
	}
	if len(candidates) == 0 {
		return Release{}, errf("no %s %s release found for %s/%s", rt, major, platform.Arch, platform.OS)
	}
	sort.Strings(candidates)
	name := candidates[len(candidates)-1]

	return Release{
		Runtime:  rt,
		Major:    major,
		FileName: name,
		URL:      indexURL + name,
		Platform: platform,
	}, nil
}

func (m MirrorClient) indexURL(rt Runtime, major string, platform Platform) string {
	return m.BaseURL + "/" + major + "/" + string(rt) + "/" + platform.Arch + "/" + platform.OS + "/"
}

func (m MirrorClient) getText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	setRequestHeaders(req)
	resp, err := m.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errf("GET %s returned %s", url, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func hrefs(s string) []string {
	matches := hrefPattern.FindAllStringSubmatch(s, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, html.UnescapeString(match[1]))
	}
	return out
}
