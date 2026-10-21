// Copyright 2026 Intrinsec
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"

	"github.com/alecthomas/kingpin/v2"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
)

var (
	vcPassword = kingpin.Flag("collector.vc.password", "vc api password").Envar("VC_PASSWORD").Required().String()
	vcUsername = kingpin.Flag("collector.vc.username", "vc api username").Envar("VC_USERNAME").Required().String()
	vcURL      = kingpin.Flag("collector.vc.url", "vc api url").Envar("VC_URL").Required().String()
	vcInsecure = kingpin.Flag("collector.vc.insecure", "Skip vCenter TLS certificate verification. Use --no-collector.vc.insecure to verify it against the system CAs.").Envar("VC_INSECURE").Default("true").Bool()
	vcCAFile   = kingpin.Flag("collector.vc.ca-file", "PEM file of the CAs used to verify the vCenter TLS certificate. Enables the verification.").Envar("VC_CA_FILE").Default("").String()

	sessions vcSession
)

// vcSession is the vCenter session shared by all scrapes. It logs in on first
// use and again only once the session has expired, instead of a login and
// logout per collector and per scrape.
type vcSession struct {
	mu     sync.Mutex
	client *govmomi.Client
}

// get returns a client with an active session.
func (s *vcSession) get(ctx context.Context, logger *slog.Logger) (*govmomi.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		// UserSession returns a nil session, not an error, once logged out.
		us, err := s.client.SessionManager.UserSession(ctx)
		if err != nil {
			return nil, err
		}
		if us != nil {
			return s.client, nil
		}
		logger.Info("vCenter session expired, logging in again")
	}

	client, err := login(ctx, logger)
	if err != nil {
		return nil, err
	}
	s.client = client
	return client, nil
}

var insecureWarning sync.Once

func login(ctx context.Context, logger *slog.Logger) (*govmomi.Client, error) {
	u, err := soap.ParseURL(*vcURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse url %q: %w", *vcURL, err)
	}

	insecure := *vcInsecure && *vcCAFile == ""
	if insecure {
		insecureWarning.Do(func() {
			logger.Warn("vCenter TLS certificate is not verified, use --collector.vc.ca-file or --no-collector.vc.insecure")
		})
	}
	sc := soap.NewClient(u, insecure)
	if *vcCAFile != "" {
		if err := sc.SetRootCAs(*vcCAFile); err != nil {
			return nil, fmt.Errorf("unable to load CA file: %w", err)
		}
	}

	vc, err := vim25.NewClient(ctx, sc)
	if err != nil {
		return nil, err
	}
	client := &govmomi.Client{Client: vc, SessionManager: session.NewManager(vc)}
	if err := client.Login(ctx, url.UserPassword(*vcUsername, *vcPassword)); err != nil {
		return nil, err
	}
	logger.Debug("logged in to vCenter", "url", *vcURL)
	return client, nil
}
