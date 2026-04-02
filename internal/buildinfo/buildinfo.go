package buildinfo

import "strings"

var (
	Version = "0.1.4"
	Commit  = "none"
	Date    = "unknown"
)

const Copyright = "Copyright © 2026 RoversX / CloseX. Licensed under GPL-3.0."

func BinaryVersion() string {
	version := strings.TrimSpace(Version)
	if version == "" {
		return "dev"
	}
	return version
}

func HeaderVersion() string {
	version := BinaryVersion()
	if version == "dev" {
		return version
	}
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func Long() string {
	return "repodock " + BinaryVersion()
}

func LongWithCopyright() string {
	return Long() + "\n" + Copyright
}
