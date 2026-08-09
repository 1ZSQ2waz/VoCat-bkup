package main

import (
	"fmt"
	"io"

	"vocat/internal/buildinfo"
)

func runVersion() {
	fmt.Println("vocat " + buildinfo.Build())
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `vocat %s

Usage:
  vocat              Run the vocat server (default; same as no arguments).
  vocat version      Print the build version and exit.
  vocat update       Check GitHub for a newer release and self-update.
                     Flags:
                       --check           Only report whether an update is available.
                       --repo owner/name GitHub repository (default: $VOCAT_REPO).
                       --target path     Binary to replace (default: running exe).
                       --force           Reinstall even at the same version.
                     Environment:
                       VOCAT_REPO        Fallback for --repo.
                       GITHUB_TOKEN      Optional bearer token for private repos
                                         or higher rate limits.
  vocat menu         Interactive lifecycle menu (run as root on the host):
                       change password, restart service, uninstall.
  vocat help         Show this help message.

When run without a subcommand, vocat starts the HTTP server using
VOCAT_* environment variables or $VOCAT_CONFIG for configuration.
`, buildinfo.Version)
}
