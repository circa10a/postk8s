package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Address defines the fields required to send Mail
type Address struct {
	// +kubebuilder:validation:Required
	Name         string `json:"name"`
	Organization string `json:"organization,omitempty"`
	// +kubebuilder:validation:Required
	Address1 string `json:"address1"`
	Address2 string `json:"address2,omitempty"`
	// +kubebuilder:validation:Required
	City string `json:"city"`
	// +kubebuilder:validation:Required
	State string `json:"state"`
	// +kubebuilder:validation:Required
	Postcode string `json:"postcode"`
	// +kubebuilder:validation:Required
	Country string `json:"country"`
}

// MailSpec defines the desired state of Mail
type MailSpec struct {
	FilePath          string `json:"filePath,omitempty"`
	URL               string `json:"url,omitempty"`
	CustomerReference string `json:"customerReference,omitempty"`
	// +kubebuilder:validation:Required
	Service string `json:"service"`
	Webhook string `json:"webhook,omitempty"`
	Company string `json:"company,omitempty"`
	Simplex bool   `json:"simplex,omitempty"`
	Color   bool   `json:"color,omitempty"`
	Flat    bool   `json:"flat,omitempty"`
	Stamp   bool   `json:"stamp,omitempty"`
	Message string `json:"message,omitempty"`
	// +kubebuilder:validation:Required
	To *Address `json:"to,omitempty"`
	// +kubebuilder:validation:Required
	From *Address `json:"from,omitempty"`
}

// MailStatus defines the observed state of Mail.
type MailStatus struct {
	ID                 string      `json:"id,omitempty"`
	State              string      `json:"state"`
	Sent               bool        `json:"sent"`
	Valid              bool        `json:"valid"`
	Total              int         `json:"total,omitempty"`
	Created            metav1.Time `json:"created,omitempty"`
	Modified           metav1.Time `json:"modified,omitempty"`
	Cancelled          metav1.Time `json:"cancelled,omitempty"`
	CancellationReason string      `json:"cancellationReason,omitempty"`
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
