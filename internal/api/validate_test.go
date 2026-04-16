package api

import "testing"

func TestValidDownloadURL(t *testing.T) {
	hosts := DefaultDownloadHosts
	tests := []struct {
		name  string
		url   string
		valid bool
	}{
		{"forgecdn edge", "https://edge.forgecdn.net/files/1234/mod.jar", true},
		{"forgecdn mediafilez", "https://mediafilez.forgecdn.net/files/1234/mod.jar", true},
		{"github raw lobo235", "https://raw.githubusercontent.com/lobo235/repo/main/file.txt", true},
		{"github lobo235", "https://github.com/lobo235/repo/releases/download/v1/file.jar", true},
		{"modrinth cdn", "https://cdn.modrinth.com/data/abc/versions/1.0/mod.jar", true},
		{"feed-the-beast", "https://feed-the-beast.com/modpacks/download/pack.zip", true},
		{"api feed-the-beast", "https://api.feed-the-beast.com/v1/packs/123", true},
		{"api modpacks ch", "https://api.modpacks.ch/public/modpack/123", true},
		{"github other user", "https://github.com/otheruser/repo/releases/download/v1/file.jar", false},
		{"raw github other user", "https://raw.githubusercontent.com/otheruser/repo/main/file.txt", false},
		{"http not https", "http://edge.forgecdn.net/files/1234/mod.jar", false},
		{"evil domain", "https://evil.com/malware.zip", false},
		{"empty string", "", false},
		{"not a url", "not-a-url", false},
		{"ftp scheme", "ftp://edge.forgecdn.net/file.zip", false},
		{"forgecdn with port", "https://edge.forgecdn.net:8080/files/mod.jar", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validDownloadURL(tt.url, hosts)
			if got != tt.valid {
				t.Errorf("validDownloadURL(%q) = %v, want %v", tt.url, got, tt.valid)
			}
		})
	}
}

func TestValidDownloadURL_CustomHosts(t *testing.T) {
	hosts := append(DefaultDownloadHosts, AllowedHost{Host: "custom.example.com", PathPrefix: ""})
	if !validDownloadURL("https://custom.example.com/file.zip", hosts) {
		t.Error("expected custom host to be allowed")
	}
}

func TestValidDownloadURL_CustomHostWithPrefix(t *testing.T) {
	hosts := []AllowedHost{{Host: "cdn.example.com", PathPrefix: "/downloads/"}}
	if !validDownloadURL("https://cdn.example.com/downloads/file.zip", hosts) {
		t.Error("expected path with prefix to be allowed")
	}
	if validDownloadURL("https://cdn.example.com/other/file.zip", hosts) {
		t.Error("expected path without prefix to be rejected")
	}
}

func TestValidServerName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"mc-server", true},
		{"server1", true},
		{"a", true},
		{"minecraft/atm9", true},
		{"minecraft/horror-land", true},
		{"category/sub", true},
		{"project/env/job", true},
		{"a/b/c", true},
		{"chatbot/staging/api-1", true},
		{"INVALID", false},
		{"-starts-with-dash", false},
		{"has spaces", false},
		{"", false},
		{"a/b/c/d", false},
		{"project/env/job/extra", false},
		{"../etc", false},
		{"/absolute", false},
		{"minecraft/", false},
		{"/minecraft/atm9", false},
		{"a//b", false},
		{"a/b/", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validServerName(tt.name)
			if got != tt.valid {
				t.Errorf("validServerName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}
