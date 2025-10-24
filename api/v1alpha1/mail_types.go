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

// Address defines the fields required to send Mail
type Address struct {
	Name         string `json:"name"`
	Organization string `json:"organization,omitempty"`
	Address1     string `json:"address1"`
	Address2     string `json:"address2,omitempty"`
	City         string `json:"city"`
	State        string `json:"state"`
	Postcode     string `json:"postcode"`
	Country      string `json:"country"`
}

// MailSpec defines the desired state of Mail
type MailSpec struct {
	FilePath          string  `json:"filePath,omitempty"`
	URL               string  `json:"url,omitempty"`
	CustomerReference string  `json:"customerReference,omitempty"`
	Service           string  `json:"service"`
	Webhook           string  `json:"webhook,omitempty"`
	Company           string  `json:"company,omitempty"`
	Simplex           bool    `json:"simplex,omitempty"`
	Color             bool    `json:"color,omitempty"`
	Flat              bool    `json:"flat,omitempty"`
	Stamp             bool    `json:"stamp,omitempty"`
	Message           string  `json:"message,omitempty"`
	To                Address `json:"to"`
	From              Address `json:"from"`
}

// MailStatus defines the observed state of Mail.
type MailStatus struct {
	ID                 string       `json:"id,omitempty"`
	State              string       `json:"state,omitempty"`
	Total              int          `json:"total,omitempty"`
	Created            *metav1.Time `json:"created,omitempty"`
	Modified           *metav1.Time `json:"modified,omitempty"`
	LastAttemptMessage string       `json:"lastAttemptMessage,omitempty"`
	Sent               bool         `json:"sent"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Mail is the Schema for the mails API
type Mail struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of Mail
	// +required
	Spec MailSpec `json:"spec"`

	// status defines the observed state of Mail
	// +optional
	Status MailStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// MailList contains a list of Mail
type MailList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Mail `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Mail{}, &MailList{})
}
