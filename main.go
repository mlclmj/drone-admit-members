// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package main

import (
	"context"
	"io"
	"net/http"

	"github.com/drone/drone-admit-members/plugin"
	"github.com/drone/drone-go/plugin/admission"

	"github.com/google/go-github/v28/github"
	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var (
	// Version is set at compile time.
	version string
	// Build revision is set at compile time.
	rev string
	// default context
	nocontext = context.Background()
)

// default github endpoint
const endpoint = "https://api.github.com/"

// spec provides the plugin settings.
type spec struct {
	Bind   string `envconfig:"DRONE_BIND"`
	Debug  bool   `envconfig:"DRONE_DEBUG"`
	Secret string `envconfig:"DRONE_SECRET"`

	Token     string `envconfig:"DRONE_GITHUB_TOKEN"`
	Endpoint  string `envconfig:"DRONE_GITHUB_ENDPOINT" default:"https://api.github.com/"`
	Org       string `envconfig:"DRONE_GITHUB_ORG"`
	OrgAdmins bool   `envconfig:"DRONE_GITHUB_ORG_ADMINS" default:"true"`
	OrgEnable bool   `envconfig:"DRONE_GITHUB_ORG_ENABLE" default:"true"`
	Team      string `envconfig:"DRONE_GITHUB_TEAM"`
}

func main() {

	if version == "" {
		version = "x.x.x"
	}

	if rev == "" {
		rev = "[unknown]"
	}

	// log as Json
	logrus.SetFormatter(&logrus.JSONFormatter{})

	spec := new(spec)
	err := envconfig.Process("", spec)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"version": version,
			"rev":     rev,
		}).Fatal(err)
	}

	if spec.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if spec.Secret == "" {
		logrus.WithFields(logrus.Fields{
			"version": version,
			"rev":     rev,
		}).Fatalln("missing secret key")
	}
	if spec.Bind == "" {
		spec.Bind = ":3000"
	}

	// creates the github client transport used
	// to authenticate API requests.
	trans := oauth2.NewClient(nocontext, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: spec.Token},
	))

	// create the github client
	client, err := github.NewEnterpriseClient(spec.Endpoint, spec.Endpoint, trans)
	if err != nil {
		logrus.Fatal(err)
	}

	// we need to lookup the github team name
	// to gets its unique system identifier
	var team int64
	if spec.Team != "" {
		result, _, err := client.Teams.GetTeamBySlug(nocontext, spec.Org, spec.Team)
		if err != nil {
			logrus.WithError(err).
				WithField("org", spec.Org).
				WithField("team", spec.Team).
				Fatalln("cannot find team")
		}
		team = result.GetID()
	}

	handler := admission.Handler(
		plugin.New(
			client,
			spec.Org,
			spec.OrgAdmins,
			spec.OrgEnable,
			team,
		),
		spec.Secret,
		logrus.StandardLogger(),
	)

	logrus.WithFields(logrus.Fields{
		"version": version,
		"rev":     rev,
	}).Infof("server listening on address %s", spec.Bind)

	http.Handle("/", handler)
	http.HandleFunc("/healthz", healthz)
	http.Handle("/metrics", promhttp.Handler())
	logrus.Fatal(http.ListenAndServe(spec.Bind, nil))
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "OK")
}
