// SPDX-License-Identifier: AGPL-3.0-only
package desktop

import "testing"

func TestPreviewDoesNotOfferOlderStable(t *testing.T) {
	const repo = "example/app"
	var releases []release
	for _, v := range []string{"1.2.4", "2.0.0"} {
		r := release{Tag: "v" + v, URL: "https://github.com/" + repo + "/releases/tag/v" + v}
		for _, suffix := range []string{"-Windows-x64-Setup.exe", "-Windows-x64-Portable.exe", "-macOS-arm64.dmg", "-macOS-x86_64.dmg"} {
			r.Assets = append(r.Assets, struct {
				Name  string `json:"name"`
				State string `json:"state"`
			}{Name: "UltraFile-" + v + suffix, State: "uploaded"})
		}
		releases = append(releases, r)
	}
	if u := newestRelease(releases[:1], repo, "2.0.0-beta.2"); u != nil {
		t.Fatalf("offered older version %s", u.Version)
	}
	if u := newestRelease(releases, repo, "2.0.0-beta.2"); u == nil || u.Version != "v2.0.0" {
		t.Fatal("matching stable upgrade missing")
	}
}
