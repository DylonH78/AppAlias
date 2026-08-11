package model

import "time"

const CurrentSchemaVersion = 1

type Source string

const (
	SourceStartMenu Source = "start-menu"
	SourceAppPaths  Source = "app-paths"
	SourceUWP       Source = "uwp"
	SourceManual    Source = "manual"
)

type LaunchKind string

const (
	LaunchExecutable LaunchKind = "executable"
	LaunchAppsFolder LaunchKind = "apps-folder"
)

type LaunchSpec struct {
	Kind             LaunchKind `json:"kind"`
	Target           string     `json:"target,omitempty"`
	Arguments        []string   `json:"arguments,omitempty"`
	WorkingDirectory string     `json:"workingDirectory,omitempty"`
	AppUserModelID   string     `json:"appUserModelId,omitempty"`
}

type Alias struct {
	Name        string     `json:"name"`
	CandidateID string     `json:"candidateId,omitempty"`
	Source      Source     `json:"source"`
	DisplayName string     `json:"displayName"`
	Launch      LaunchSpec `json:"launch"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type Config struct {
	SchemaVersion int              `json:"schemaVersion"`
	Aliases       map[string]Alias `json:"aliases"`
}

type Candidate struct {
	ID                   string     `json:"id"`
	DisplayName          string     `json:"displayName"`
	Source               Source     `json:"source"`
	Launch               LaunchSpec `json:"launch"`
	Suggestions          []string   `json:"suggestions"`
	Recommended          string     `json:"recommended,omitempty"`
	RequiresConfirmation bool       `json:"requiresConfirmation"`
	ConfirmationReason   string     `json:"confirmationReason,omitempty"`
}

type ScanResult struct {
	Candidates  []Candidate `json:"candidates"`
	Diagnostics []string    `json:"diagnostics,omitempty"`
}
