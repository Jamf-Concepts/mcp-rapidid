// Copyright 2026, Jamf Software LLC

package ri

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/helper"
	"github.com/Jamf-Concepts/mcp-rapidid/internal/pkg/telemetry"
	"github.com/hatch-ed-com/ri-sdk-go/pkg/rapididentity"
)

// POST /admin/connect/jobs creates a job when the body has no id and updates
// the job with the body's id otherwise. Observed against a live tenant:
//
//   - A body without an action fails (500, NullPointerException).
//   - A new job whose name is already taken, compared exactly, fails with
//     "Duplicate job name". A name differing only in case is accepted, so
//     create-connect-job refuses those itself.
//   - An update needs the job's current version; a stale one fails with 409
//     "The data you provided is out of date".
//   - An update REPLACES the whole job: a field missing from the body is
//     reset (cronSpec to null, timeZone to UTC, timeoutSeconds to 0, the
//     action's args to empty). So update-connect-job sends back the whole
//     current job with only the requested fields changed.
//   - Every job runs an action set from its own project.

// ConnectJobArgInput is an argument the job passes to its action set.
type ConnectJobArgInput struct {
	Name  string `json:"name" jsonschema:"The name of one of the action set's arguments"`
	Value string `json:"value" jsonschema:"The value as a Connect expression, for example true, 5 or \"normal\" including the quotes. An empty string leaves the argument without a value"`
}

// connectJobAction is a job's action as the server sends it. The id is the
// job's own action step, not the action set's id; the server generates one
// when it is missing.
type connectJobAction struct {
	Id      string          `json:"id,omitempty"`
	Name    string          `json:"name"`
	Project string          `json:"project"`
	Args    []connectJobArg `json:"args"`
}

// connectJobArg is one argument of a job's action. A nil Value is an
// argument without a value, which is how Connect stores one left unset.
type connectJobArg struct {
	Name  string  `json:"name"`
	Value *string `json:"value,omitempty"`
}

// connectJobChanges holds the job fields that create and update set, other
// than the action. A nil field is left as it is.
type connectJobChanges struct {
	Description *string `json:"description,omitempty"`
	CronSpec    *string `json:"cronSpec,omitempty"`
	TimeZone    *string `json:"timeZone,omitempty"`
	Disabled    *bool   `json:"disabled,omitempty"`
}

// rawFields returns the fields that are set, keyed by their JSON names and
// encoded as raw JSON, ready to be merged into a job body.
func (c connectJobChanges) rawFields() (map[string]json.RawMessage, error) {
	encoded, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	return fields, nil
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
// longer has are dropped.
func connectJobArgsFor(actionSet *rapididentity.ActionDef, current []connectJobArg) []connectJobArg {
	values := make(map[string]*string, len(current))
	for _, arg := range current {
		values[arg.Name] = arg.Value
	}
	args := make([]connectJobArg, 0, len(actionSet.ArgDefs))
	for _, def := range actionSet.ArgDefs {
		args = append(args, connectJobArg{Name: def.Name, Value: values[def.Name]})
	}
	return args
}

// applyConnectJobArgs sets the given argument values by name; an empty value
// leaves the argument without one. Every name must be one of the action
// set's arguments.
func applyConnectJobArgs(actionSet *rapididentity.ActionDef, args []connectJobArg, changes []ConnectJobArgInput) ([]connectJobArg, error) {
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
		if change.Value == "" {
			args[i].Value = nil
		} else {
			value := change.Value
			args[i].Value = &value
		}
	}
	return args, nil
}

// postConnectJob sends a job body to POST /admin/connect/jobs and returns the
// saved job.
func postConnectJob(ctx context.Context, client *rapididentity.Client, sh *helper.ServerHelper, toolName string, body []byte) (rapididentity.ConnectJob, error) {
	sh.Logger().Info("Calling admin/connect/jobs endpoint")
	res, err := client.DoCustomRequest(ctx, "POST", "admin/connect/jobs", bytes.NewBuffer(body))
	if err != nil {
		LogRIError(ctx, sh, "unable to save Connect job", err)
		return rapididentity.ConnectJob{}, err
	}

	resBody, err := readConnectJobResponse(ctx, sh, toolName, res)
	if err != nil {
		return rapididentity.ConnectJob{}, err
	}

	sh.Logger().Debug("POST admin/connect/jobs response body", "body", string(resBody))

	var job rapididentity.ConnectJob
	if err = json.Unmarshal(resBody, &job); err != nil {
		sh.Logger().Error("unable to unmarshal json for POST admin/connect/jobs response body", "error", err)
		telemetry.RecordError(ctx, fmt.Sprintf(telemetry.EventPrefix+telemetry.ToolErrorPrefix+"%s", toolName), "see server logs", telemetry.ErrorCategoryThrownException)
		return rapididentity.ConnectJob{}, err
	}
	return job, nil
}
