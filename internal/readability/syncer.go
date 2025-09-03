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

// Package readability implements a team synchronization for readability.
package readability

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"sync"

	"github.com/google/go-github/v74/github"

	"github.com/abcxyz/pkg/logging"
)

// ignoreOrgAdmins are the list of enterprise-installed org admins to ignore for
// computations.
var ignoredOrgAdmins = map[string]struct{}{
	"google-admin":     {},
	"google-ospo-team": {},
	"googlebot":        {},
}

// Syncer is a wrapper around a GitHub API client and some context.
type Syncer struct {
	githubClient *github.Client
	dryRun       bool

	orgAdminsCacheLock sync.RWMutex
	orgAdminsCache     map[string]map[string]struct{}
}

// NewSyncer creates a new readability syncer.
func NewSyncer(ctx context.Context, githubClient *github.Client, dryRun bool) (*Syncer, error) {
	return &Syncer{
		githubClient: githubClient,
		dryRun:       dryRun,
	}, nil
}

// Sync iterates over the upstream team memberships and ensures the given
// targetMemberships exactly match the upstream team memberships. The
// targetMemeberships is a map of GitHub username => team permission (e.g.
// "member", "maintainer").
func (s *Syncer) Sync(ctx context.Context, org, team string, targetMemberships map[string]string) error {
	logger := logging.FromContext(ctx).With(
		"org", org,
		"team", team)

	logger.DebugContext(ctx, "starting sync",
		"target_memberships", targetMemberships)
	defer logger.DebugContext(ctx, "finished sync")

	orgAdmins, err := s.OrgAdmins(ctx, org)
	if err != nil {
		return err
	}
	logger.DebugContext(ctx, "found org admins",
		"admins", MapKeysLogAttr(orgAdmins))

	existingMembers, _, err := s.githubClient.Teams.ListTeamMembersBySlug(ctx, org, team, &github.TeamListTeamMembersOptions{
		Role: "member",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get members for github team %s/%s: %w", org, team, err)
	}

	existingMaintainers, _, err := s.githubClient.Teams.ListTeamMembersBySlug(ctx, org, team, &github.TeamListTeamMembersOptions{
		Role: "maintainer",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get maintainers for github team %s/%s: %w", org, team, err)
	}

	// Put things in a map so things go fast.
	existingMemberships := make(map[string]string, len(existingMaintainers)+len(existingMembers))
	for _, v := range existingMembers {
		existingMemberships[*v.Login] = "member"
	}
	for _, v := range existingMaintainers {
		existingMemberships[*v.Login] = "maintainer"
	}
	logger.DebugContext(ctx, "upstream team state",
		"existing_memberships", existingMemberships)

	var merr error

	// Add anyone who isn't in the list
	for user, targetRole := range targetMemberships {
		existingRole, ok := existingMemberships[user]

		// Org Admins can ONLY be added as "maintainer" types to teams. This
		// prevents a perpetual diff for org admins.
		if _, ok := orgAdmins[user]; ok && targetRole != "maintainer" {
			logger.WarnContext(ctx, "upgrading target role to maintainer for org admin",
				"user", user,
				"target_role", targetRole)
			targetRole = "maintainer"
		}

		// User is not in the list, add them.
		if !ok {
			logger.DebugContext(ctx, "adding missing user",
				"user", user,
				"target_role", targetRole)

			fmt.Fprintf(os.Stdout, "✅ adding %s to %s/%s as %s\n", user, org, team, targetRole)

			if !s.dryRun {
				if _, _, err := s.githubClient.Teams.AddTeamMembershipBySlug(ctx, org, team, user, &github.TeamAddTeamMembershipOptions{
					Role: targetRole,
				}); err != nil {
					merr = errors.Join(merr, fmt.Errorf("failed to add %s to %s/%s: %w", user, org, team, err))
				}
			}
		}

		// The user is in the team, but with the wrong permissions.
		if existingRole != "" && existingRole != targetRole {
			logger.DebugContext(ctx, "user exists with wrong role",
				"user", user,
				"existing_role", existingRole,
				"target_role", targetRole)

			fmt.Fprintf(os.Stdout, "♻️ updating %s role in %s/%s from %s to %s\n",
				user, org, team, existingRole, targetRole)

			if !s.dryRun {
				if _, _, err := s.githubClient.Teams.AddTeamMembershipBySlug(ctx, org, team, user, &github.TeamAddTeamMembershipOptions{
					Role: targetRole,
				}); err != nil {
					merr = errors.Join(merr, fmt.Errorf("failed to add %s to %s/%s: %w", user, org, team, err))
				}
			}
		}
	}

	// Remove anyone who should no longer be in the list
	for user := range existingMemberships {
		if _, ok := targetMemberships[user]; !ok {
			logger.DebugContext(ctx, "remove existing user",
				"user", user)

			fmt.Fprintf(os.Stdout, "❌ removing %s from %s/%s\n", user, org, team)

			if !s.dryRun {
				if _, err := s.githubClient.Teams.RemoveTeamMembershipBySlug(ctx, org, team, user); err != nil {
					merr = errors.Join(merr, fmt.Errorf("failed to remove %s from %s/%s: %w", user, org, team, err))
				}
			}
		}
	}

	return merr
}

// OrgAdmins gets the list of org admins. This is a helper that wraps the cache.
func (s *Syncer) OrgAdmins(ctx context.Context, org string) (map[string]struct{}, error) {
	logger := logging.FromContext(ctx).With("org", org)

	// Try the fast path cache first
	s.orgAdminsCacheLock.RLock()
	if v, ok := s.orgAdminsCache[org]; ok {
		s.orgAdminsCacheLock.RUnlock()
		logger.DebugContext(ctx, "using cached org admins")
		return v, nil
	}
	s.orgAdminsCacheLock.RUnlock()

	// We missed, do a full lookup
	s.orgAdminsCacheLock.Lock()
	defer s.orgAdminsCacheLock.Unlock()

	logger.DebugContext(ctx, "looking up org admins")
	orgAdmins, _, err := s.githubClient.Organizations.ListMembers(ctx, org, &github.ListMembersOptions{
		Role: "admin",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get org admins for %s: %w", org, err)
	}

	m := make(map[string]struct{}, len(orgAdmins))
	for _, admin := range orgAdmins {
		login := *admin.Login
		if _, ok := ignoredOrgAdmins[login]; !ok {
			m[login] = struct{}{}
		}
	}

	// Save the result
	if s.orgAdminsCache == nil {
		s.orgAdminsCache = make(map[string]map[string]struct{}, 8)
	}
	s.orgAdminsCache[org] = m

	// Return
	return m, nil
}

// MapKeysLogAttr is a helper that creates a log entry for a map's keys.
func MapKeysLogAttr[K cmp.Ordered, V any](m map[K]V) *mapKeysAttr[K, V] {
	return &mapKeysAttr[K, V]{
		m: m,
	}
}

type mapKeysAttr[K cmp.Ordered, V any] struct {
	m map[K]V
}

func (a mapKeysAttr[K, V]) LogValue() slog.Value {
	return slog.AnyValue(slices.Sorted(maps.Keys(a.m)))
}
