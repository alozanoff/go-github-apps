package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v52/github"
	"github.com/k0kubun/pp/v3"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func versionInfo() {
	fmt.Fprintf(os.Stderr, "Version: %s\nCommit: %s\nBuiltAt: %s\n", version, commit, date)
}

func main() {
	appID := flag.Int64("app-id", 0, "App ID")
	instID := flag.Int64("inst-id", 0, "Installation ID")
	export := flag.Bool("export", false, "show token as 'export GITHUB_TOKEN=...'")
	showVersion := flag.Bool("version", false, "show version info")
	showInsts := flag.Bool("show-insts", false, "show all of the installations for the app")

	origUsage := flag.Usage
	flag.Usage = func() {
		origUsage()
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "== Build Info ==\n")
		versionInfo()
	}

	flag.Parse()

	// See https://github.com/golang/go/issues/37533
	// I decided to implement -version flag to return 0
	if *showVersion {
		flag.Usage()
		os.Exit(0)
	}

	key := os.Getenv("GITHUB_PRIV_KEY")
	if key == "" {
		log.Fatal("Please populate GITHUB_PRIV_KEY environment variable with the private key for the App")
	}

	if *showInsts {
		if *appID == 0 {
			fmt.Fprintf(os.Stderr, "App ID is required to show the installations for the app.\n\n")
			flag.Usage()
			os.Exit(1)
		}

		showInstallations(*appID, []byte(key))

		return
	}

	if *appID == 0 || *instID == 0 {
		fmt.Fprintf(os.Stderr, "App ID and Installation ID are required.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Wrap the shared transport for use with the app ID 1 authenticating with installation ID 99.
	itr, err := ghinstallation.New(http.DefaultTransport, *appID, *instID, []byte(key))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	token, err := itr.Token(ctx)
	if err != nil {
		log.Fatalf("unable to get github token: %s", err)
	}

	if *export {
		showExport(token)
	} else {
		fmt.Println(token)
	}
}

func showExport(token string) {
	fmt.Printf("export GITHUB_TOKEN=%s\n", token)
}

func showInstallations(appID int64, key []byte) {
	atr, err := ghinstallation.NewAppsTransport(http.DefaultTransport, appID, key)
	if err != nil {
		log.Fatal(err)
	}

	client := github.NewClient(&http.Client{Transport: atr})

	opts := &github.ListOptions{
		PerPage: 10,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	for {
		inst, resp, err := client.Apps.ListInstallations(ctx, opts)
		if err != nil {
			log.Fatal(err)
		}

		pp.Println(inst)

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}
}
