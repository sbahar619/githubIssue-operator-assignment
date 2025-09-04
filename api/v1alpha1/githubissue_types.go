/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GithubIssueSpec defines the desired state of GithubIssue
type GithubIssueSpec struct {
	// Repo is the GitHub repository URL
	// +kubebuilder:validation:Pattern=`^https://github\.com/[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`
	// +kubebuilder:validation:MinLength=19
	// +kubebuilder:validation:MaxLength=150
	Repo string `json:"repo"`

	// Title is the GitHub issue title that will be created or updated
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	Title string `json:"title"`

	// Description is the GitHub issue body content
	// +kubebuilder:validation:MaxLength=65536
	// +optional
	Description *string `json:"description,omitempty"`
}

// GithubIssueStatus defines the observed state of GithubIssue.
type GithubIssueStatus struct {
	// Conditions represent the latest available observations of the GithubIssue's current state
	// Follows standard Kubernetes condition patterns for status reporting
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// IssueID is the GitHub issue number/ID returned from GitHub API
	// This field is populated once the issue is successfully created or found
	// +optional
	IssueID *int `json:"issueID,omitempty"`

	// URL is the direct link to the GitHub issue
	// Contains the full GitHub issue URL when available
	// +optional
	URL *string `json:"url,omitempty"`

	// State represents the current GitHub issue state
	// Reflects the current state as returned by GitHub API
	// +optional
	State *string `json:"state,omitempty"`

	// HasPullRequest indicates if the issue has an associated pull request
	// Populated based on GitHub issue analysis
	// +optional
	HasPullRequest *bool `json:"hasPullRequest,omitempty"`

	// LastSyncTime is when the issue was last synchronized with GitHub
	// Updated during successful synchronization operations
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// GithubIssue is the Schema for the githubissues API
type GithubIssue struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of GithubIssue
	// +required
	Spec GithubIssueSpec `json:"spec"`

	// status defines the observed state of GithubIssue
	// +optional
	Status GithubIssueStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// GithubIssueList contains a list of GithubIssue
type GithubIssueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GithubIssue `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GithubIssue{}, &GithubIssueList{})
}
