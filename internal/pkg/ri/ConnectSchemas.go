// Copyright 2026, Jamf Software LLC

package ri

import "github.com/google/jsonschema-go/jsonschema"

// connectSchemaDefs defines $defs shared across Connect tool schemas to break
// the cycle: ConnectAction -> ArgDefList -> ArgDef.Actions -> ConnectActionList -> ConnectAction.
var connectSchemaDefs = map[string]*jsonschema.Schema{
	"argDef": {
		Type:        "object",
		Description: "An input parameter of the action set.",
		Properties: map[string]*jsonschema.Schema{
			"optional":    {Type: "boolean", Description: "Whether the action set input parameter is optional."},
			"type":        {Type: "string", Description: "The type of the input parameter"},
			"name":        {Type: "string", Description: "The name of the input parameter."},
			"description": {Type: "string", Description: "The description for the input parameter."},
			"value":       {Type: "string", Description: "The value of the input parameter"},
			"actions": {
				Type:        "array",
				Description: "The nested actions for container args such as while, if, section, else",
				Items:       &jsonschema.Schema{Ref: "#/$defs/connectAction"},
			},
		},
	},
	"connectAction": {
		Type:        "object",
		Description: "An individual action step within a Connect action set.",
		Properties: map[string]*jsonschema.Schema{
			"id":        {Type: "string", Description: "The unique ID of the Connect action."},
			"name":      {Type: "string", Description: "The name of the Connect action."},
			"outputVar": {Type: "string", Description: "Whether the action returns a value."},
			"disabled":  {Type: "boolean", Description: "Whether the action is disabled."},
			"project":   {Type: "string", Description: "The project where the action resides."},
			"args": {
				Type:        "array",
				Description: "The input parameters for the action.",
				Items:       &jsonschema.Schema{Ref: "#/$defs/argDef"},
			},
		},
	},
	"actionDef": {
		Type:        "object",
		Description: "A Connect action set definition.",
		Properties: map[string]*jsonschema.Schema{
			"id":             {Type: "string", Description: "The action set ID. On creation of an action this must be populated with a UUID of the format xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"},
			"version":        {Type: "integer", Description: "The action set version."},
			"project":        {Type: "string", Description: "The project where the action set resides."},
			"name":           {Type: "string", Description: "The name of the action set."},
			"category":       {Type: "string", Description: "The category the action set is a part of."},
			"builtIn":        {Type: "boolean", Description: "Whether the action set is built in or custom."},
			"community":      {Type: "boolean", Description: "Whether the action set is a part of the community depot."},
			"returnsValue":   {Type: "boolean", Description: "Whether the action set returns a value."},
			"description":    {Type: "string", Description: "The description of the action set."},
			"unlicensed":     {Type: "boolean", Description: "Whether the action set is licensed or not."},
			"sensitive":      {Type: "boolean", Description: "Whether the action set contains sensitive information."},
			"deprecated":     {Type: "string", Description: "Whether the action set is deprecated."},
			"httpStatus":     {Type: "integer", Description: "The http status code of the return."},
			"changeCount":    {Type: "integer", Description: "The number of times the action set has been modified."},
			"modifiedMs":     {Type: "integer", Description: "When the action set was last modified in milliseconds."},
			"modifiedBy":     {Type: "string", Description: "The idautoID of the user who last modified the action set."},
			"modifiedByName": {Type: "string", Description: "The display name of the user who last modified the action set."},
			"argDefs": {
				Type:        "array",
				Description: "The input parameters of the action set.",
				Items:       &jsonschema.Schema{Ref: "#/$defs/argDef"},
			},
			"actions": {
				Type:        "array",
				Description: "The actions within the action set.",
				Items:       &jsonschema.Schema{Ref: "#/$defs/connectAction"},
			},
		},
	},
}

// ConnectActionDefSchema is the schema for a single ActionDef, used as output
// for get-connect-action and save-connect-action.
var ConnectActionDefSchema = &jsonschema.Schema{
	Type: "object",
	Defs: connectSchemaDefs,
	Ref:  "#/$defs/actionDef",
}

// ConnectActionsOutputSchema is the output schema for get-connect-actions.
var ConnectActionsOutputSchema = &jsonschema.Schema{
	Type: "object",
	Defs: connectSchemaDefs,
	Properties: map[string]*jsonschema.Schema{
		"name": {Type: "string", Description: "Query type name. For example \"all\"."},
		"actionDefs": {
			Type:        "array",
			Description: "The list of actions.",
			Items:       &jsonschema.Schema{Ref: "#/$defs/actionDef"},
		},
	},
}

// SaveConnectActionInputSchema is the input schema for save-connect-action.
var SaveConnectActionInputSchema = &jsonschema.Schema{
	Type: "object",
	Defs: connectSchemaDefs,
	Properties: map[string]*jsonschema.Schema{
		"action": {
			Description: "The action to save or update. When updating an existing action set the version number must be the one provided to you in a Connect action query.",
			Ref:         "#/$defs/actionDef",
		},
	},
}

// RunConnectActionInputSchema is the input schema for run-connect-action.
var RunConnectActionInputSchema = &jsonschema.Schema{
	Type: "object",
	Defs: connectSchemaDefs,
	Properties: map[string]*jsonschema.Schema{
		"action": {
			Description: "The Connect action to run",
			Ref:         "#/$defs/connectAction",
		},
	},
}
