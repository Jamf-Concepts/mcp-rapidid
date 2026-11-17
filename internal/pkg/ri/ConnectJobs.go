// Copyright 2026, Jamf Software LLC

package ri

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/helper"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

// The Connect job endpoints behave like this (observed against a live tenant):
//
//   - GET/DELETE /admin/connect/jobs/{nameOrId} and GET runJob/{nameOrId}
//     do not find a job by its bare name (404 "Couldn't find job"); only the
//     id or "<project>.<name>" work. So the tools look the job up by name in
//     the job list and address it by its id.
//   - POST /admin/connect/jobs creates a job when the body has no id and
//     updates the job with the body's id otherwise. A body without an action
//     fails (500). A new job whose name is already taken, compared exactly,
//     fails with "Duplicate job name", but a name differing only in case is
//     accepted, so create-connect-job refuses those itself.
//   - An update needs the job's current version; a stale one fails with 409.
//   - An update REPLACES the whole job: a field missing from the body is
//     reset (cronSpec to null, timeZone to UTC, timeoutSeconds to 0, the
//     action's args to empty). So update-connect-job reads the whole job into
//     rapididentity.ConnectJob, which models every field the API returns, and
//     posts it back with only the requested fields changed.
//   - A disabled job can still be run with runJob.
//   - Every job runs an action set from its own project.

// ConnectJobArgInput is an argument the job passes to its action set.
type ConnectJobArgInput struct {
	Name  string `json:"name" jsonschema:"The name of one of the action set's arguments"`
	Value string `json:"value" jsonschema:"The value as a Connect expression, for example true, 5 or \"normal\" including the quotes. An empty string leaves the argument without a value"`
}

// findConnectJob returns the job named name, or nil when there is none. Job
// names are unique across all projects, but only when compared exactly: the
// server refuses a second "xyz1JobTest" ("Duplicate job name") yet accepts
// "XYZ1JOBTEST" next to it. So an exact match wins; otherwise a single match
// ignoring case is used, and several matches ignoring case are an error
// rather than a guess.
func findConnectJob(ctx context.Context, client *rapididentity.Client, sh *helper.ServerHelper, name string) (*rapididentity.ConnectJob, error) {
	sh.Logger().Info("Looking up the Connect job by name")
	jobs, err := client.GetConnectJobs(ctx, rapididentity.GetConnectJobsInput{})
	if err != nil {
		LogRIError(ctx, sh, "unable to retrieve rapididentity connect jobs", err)
		return nil, err
	}
	var match *rapididentity.ConnectJob
	var matchNames []string
	for i := range jobs.Jobs {
		if jobs.Jobs[i].Name == name {
			return &jobs.Jobs[i], nil
		}
		if strings.EqualFold(jobs.Jobs[i].Name, name) {
			match = &jobs.Jobs[i]
			matchNames = append(matchNames, jobs.Jobs[i].Name)
		}
	}
	if len(matchNames) > 1 {
		err = fmt.Errorf("more than one Connect job matches %q ignoring case (%s); give the exact name", name, strings.Join(matchNames, ", "))
		LogRIError(ctx, sh, "Connect job request refused", err)
		return nil, err
	}
	return match, nil
}

// requireConnectJob returns the existing job from the job list, or an error
// naming the missing job. Callers address the job by the returned job's id.
func requireConnectJob(ctx context.Context, client *rapididentity.Client, sh *helper.ServerHelper, name string) (*rapididentity.ConnectJob, error) {
	job, err := findConnectJob(ctx, client, sh, name)
	if err != nil {
		return nil, err
	}
	if job == nil {
		err = fmt.Errorf("no Connect job named %q exists", name)
		LogRIError(ctx, sh, "Connect job request refused", err)
		return nil, err
	}
	return job, nil
}

// lookupConnectActionSet returns the action set's metadata, including its
// argument definitions, or a clear error when the project has no action set
// with that name.
func lookupConnectActionSet(ctx context.Context, client *rapididentity.Client, sh *helper.ServerHelper, project string, name string) (*rapididentity.ActionDef, error) {
	sh.Logger().Info("Looking up the Connect action set", "project", project, "name", name)
	result, err := client.GetConnectActionById(ctx, rapididentity.GetConnectActionByIdInput{Id: project + "." + name, MetaDataOnly: true})
	if err != nil {
		var riErr rapididentity.RapidIdentityError
		if errors.As(err, &riErr) && riErr.Code == http.StatusNotFound {
			err = fmt.Errorf("project %q has no action set named %q", project, name)
		}
		LogRIError(ctx, sh, "unable to retrieve Connect action set", err)
		return nil, err
	}
	return &result.Action, nil
}

// connectJobArgsFor lists the action set's arguments in order, keeping the
// value of each one that current already sets. Arguments the action set no
// longer has are dropped. An argument without a value has an empty Value,
// which is left out of the JSON, as Connect stores one left unset.
func connectJobArgsFor(actionSet *rapididentity.ActionDef, current rapididentity.ArgDefList) rapididentity.ArgDefList {
	values := make(map[string]string, len(current))
	for _, arg := range current {
		values[arg.Name] = arg.Value
	}
	args := make(rapididentity.ArgDefList, 0, len(actionSet.ArgDefs))
	for _, def := range actionSet.ArgDefs {
		args = append(args, rapididentity.ArgDef{Name: def.Name, Value: values[def.Name]})
	}
	return args
}

// applyConnectJobArgs sets the given argument values by name; an empty value
// leaves the argument without one. Every name must be one of the action
// set's arguments.
func applyConnectJobArgs(actionSet *rapididentity.ActionDef, args rapididentity.ArgDefList, changes []ConnectJobArgInput) (rapididentity.ArgDefList, error) {
	index := make(map[string]int, len(args))
	for i, arg := range args {
		index[arg.Name] = i
	}
	for _, change := range changes {
		i, ok := index[change.Name]
		if !ok {
			known := make([]string, 0, len(args))
			for _, arg := range args {
				known = append(known, arg.Name)
			}
			sort.Strings(known)
			return nil, fmt.Errorf("action set %q has no argument named %q (its arguments: %s)", actionSet.Name, change.Name, strings.Join(known, ", "))
		}
		args[i].Value = change.Value
	}
	return args, nil
}
