// Copyright The Linux Foundation and each contributor to CommunityBridge.
// SPDX-License-Identifier: MIT

package main

import (
	"os"

	"github.com/communitybridge/easycla/cla-backend-go/cmd/functional_tests/cla_manager"
	"github.com/communitybridge/easycla/cla-backend-go/cmd/functional_tests/health"
	"github.com/communitybridge/easycla/cla-backend-go/cmd/functional_tests/test_models"
	log "github.com/communitybridge/easycla/cla-backend-go/logging"
	"github.com/verdverm/frisby"
)

const (
	// API_URL is the development API endpoint
	API_URL = "https://api.dev.lfcla.com/v3" // nolint
)

func init() {
	frisby.Global.PrintProgressName = true
}

func main() {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = API_URL
	}
	log.Debugf("API_URL: %s", apiURL)
	auth0Username := os.Getenv("AUTH0_USERNAME")
	if auth0Username == "" {
		log.Warnf("Unable to run tests - missing AUTH0_USERNAME environment variable")
		os.Exit(1)
	}
	auth0Password := os.Getenv("AUTH0_PASSWORD")
	if auth0Password == "" {
		log.Warnf("Unable to run tests - missing AUTH0_PASSWORD environment variable")
		os.Exit(1)
	}
	auth0ClientID := os.Getenv("AUTH0_CLIENT_ID")
	if auth0ClientID == "" {
		log.Warnf("Unable to run tests - missing AUTH0_CLIENT_ID environment variable")
		os.Exit(1)
	}
	auth0Config := test_models.Auth0Config{
		Auth0UserName: auth0Username,
		Auth0Password: auth0Password,
		Auth0ClientID: auth0ClientID,
	}

	health.NewTestBehaviour(apiURL, auth0Config).RunAllTests()
	cla_manager.NewTestBehaviour(apiURL, auth0Config).RunAllTests()
	frisby.Global.PrintReport()
}
