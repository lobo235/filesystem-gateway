package api

import (
	"net/url"
	"regexp"
	"strings"
)

// serverNameRegex validates server names: plain names ("myserver") or up to two
// levels of subdirectory nesting ("project/env/job"). Each segment must be
// lowercase alphanumeric with optional dashes, max 48 chars.
var serverNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,47}(/[a-z0-9][a-z0-9-]{0,47}){0,2}$`)

// AllowedHost represents a host (and optional path prefix) permitted for file downloads.
type AllowedHost struct {
	Host       string
	PathPrefix string // empty means any path on this host is allowed
}

// DefaultDownloadHosts lists the built-in hostnames (and optional path prefixes)
// permitted for file downloads.
var DefaultDownloadHosts = []AllowedHost{
	// CurseForge CDN
	{"edge.forgecdn.net", ""},
	{"mediafilez.forgecdn.net", ""},
	// Modrinth
	{"cdn.modrinth.com", ""},
	// FTB
	{"feed-the-beast.com", ""},
	{"api.feed-the-beast.com", ""},
	{"api.modpacks.ch", ""},
	// Forge / NeoForge
	{"maven.minecraftforge.net", ""},
	{"files.minecraftforge.net", ""},
	{"maven.neoforged.net", ""},
	// Paper / Bukkit / Spigot
	{"api.papermc.io", ""},
	{"download.getbukkit.org", ""},
	{"hub.spigotmc.org", ""},
	// Fabric
	{"maven.fabricmc.net", ""},
	{"meta.fabricmc.net", ""},
	// Mojang
	{"piston-data.mojang.com", ""},
	{"launchermeta.mojang.com", ""},
	{"launcher.mojang.com", ""},
	{"libraries.minecraft.net", ""},
	{"resources.download.minecraft.net", ""},
	// Trusted GitHub (operator scripts)
	{"raw.githubusercontent.com", "/lobo235/"},
	{"github.com", "/lobo235/"},
}

// validServerName checks if the server name matches the allowed pattern.
func validServerName(name string) bool {
	return serverNameRegex.MatchString(name)
}

// validDownloadURL checks that the URL is an allowed HTTPS download source.
func validDownloadURL(rawURL string, hosts []AllowedHost) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if u.Scheme != "https" {
		return false
	}
	for _, allowed := range hosts {
		if strings.EqualFold(u.Host, allowed.Host) {
			if allowed.PathPrefix == "" {
				return true
			}
			if strings.HasPrefix(u.Path, allowed.PathPrefix) {
				return true
			}
		}
	}
	return false
}
