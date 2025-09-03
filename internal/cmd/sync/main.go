// Copyright 2025 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main is the entrypoint to the syncer.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/google/go-github/v74/github"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/readability/internal/readability"
)

const (
	readabilityDirectory = "readability"
	org                  = "abcxyz"
)

var (
	githubToken = os.Getenv("GITHUB_TOKEN")
	dryRun, _   = strconv.ParseBool(os.Getenv("DRY_RUN"))
	debug, _    = strconv.ParseBool(os.Getenv("DEBUG"))
)

func main() {
	ctx, done := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer done()

	logLevel := slog.LevelError
	if debug {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:       logLevel,
		ReplaceAttr: replaceAttrs,
	}))
	ctx = logging.WithLogger(ctx, logger)

	if err := realMain(ctx); err != nil {
		done()
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func realMain(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	// Parse YAMLs
	files, err := os.ReadDir(readabilityDirectory)
	if err != nil {
		return fmt.Errorf("failed to read readability yaml directory: %w", err)
	}

	configs := make(map[string]map[string]string, len(files))
	for _, file := range files {
		// Ignore directories
		if file.IsDir() {
			continue
		}

		// Ignore non-YAMLs
		ext := filepath.Ext(file.Name())
		if ext != ".yaml" {
			continue
		}

		// Read the file
		contents, err := os.ReadFile(filepath.Join(readabilityDirectory, file.Name()))
		if err != nil {
			return fmt.Errorf("failed to read readability YAML for %s: %w", file.Name(), err)
		}

		var data map[string]string
		if err := yaml.Unmarshal(contents, &data); err != nil {
			return fmt.Errorf("failed to unmarshal YAML for %s: %w", file.Name(), err)
		}

		language := strings.TrimSuffix(filepath.Base(file.Name()), ext)
		configs[language] = data
	}

	logger.DebugContext(ctx, "found readability configs",
		"configs", readability.MapKeysLogAttr(configs))

	// Create the github client
	httpClient := newHTTPClient()
	defer httpClient.CloseIdleConnections()
	githubClient := github.NewClient(httpClient).WithAuthToken(githubToken)

	// Create the syncer
	syncer, err := readability.NewSyncer(ctx, githubClient, dryRun)
	if err != nil {
		return fmt.Errorf("failed to create readability syncer: %w", err)
	}

	if dryRun {
		fmt.Fprintf(os.Stdout, "⚠️ Operating in dry-run mode, changes will not be applied\n")
	}

	// Iterate over each YAML and sync the teams
	var merr error
	for _, language := range slices.Sorted(maps.Keys(configs)) {
		targetMemberships := configs[language]

		fmt.Fprintf(os.Stdout, "🔄 Synchronizing %s...\n", language)

		// Default readability
		readabilityTeam := language + "-readability"
		if err := syncer.Sync(ctx, org, readabilityTeam, targetMemberships); err != nil {
			merr = errors.Join(merr, fmt.Errorf("failed to sync %s: %w", language, err))
		}

		// Readability approvers
		targetApproverMemberships := make(map[string]string, len(targetMemberships))
		for k, v := range targetMemberships {
			if v == "maintainer" {
				targetApproverMemberships[k] = v
			}
		}
		readabilityApproversTeam := language + "-readability-approvers"
		if err := syncer.Sync(ctx, org, readabilityApproversTeam, targetApproverMemberships); err != nil {
			merr = errors.Join(merr, fmt.Errorf("failed to sync %s: %w", language, err))
		}
	}

	return merr
}

// newHTTPClient creates a new HTTP client that has no global state and uses a
// pooled transport since all connections are going to the same host.
//
// Callers should call CloseIdleConnections when shutting down the Client to
// avoid leaking file descriptors.
func newHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ForceAttemptHTTP2:     true,
			MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		},
	}
}

func replaceAttrs(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{}
	}
	if a.Key == slog.LevelKey && len(groups) == 0 {
		return slog.Attr{}
	}
	return a
}
