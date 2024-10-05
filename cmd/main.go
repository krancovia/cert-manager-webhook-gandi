package main

import (
	"log"
	"os"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"

	"github.com/krancovia/cert-manager-webhook-gandi/internal/gandi"
	"github.com/krancovia/cert-manager-webhook-gandi/internal/version"
)

func main() {
	ver := version.GetVersion()
	log.Printf(
		"Starting cert-manager-webhook-gandi version=%s commit=%s\n",
		ver.Version, ver.GitCommit,
	)

	groupName := os.Getenv("GROUP_NAME")
	if groupName == "" {
		panic("GROUP_NAME must be specified")
	}

	cmd.RunWebhookServer(groupName, gandi.NewSolver())
}
