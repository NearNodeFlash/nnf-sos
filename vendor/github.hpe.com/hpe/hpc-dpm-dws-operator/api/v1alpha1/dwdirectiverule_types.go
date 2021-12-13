package v1alpha1

import (
	"github.hpe.com/hpe/hpc-dpm-dws-operator/utils/dwdparse"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true

// DWDirectiveRule is the Schema for the DWDirective API
type DWDirectiveRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec []dwdparse.DWDirectiveRuleSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// DWDirectiveRuleList contains a list of DWDirective
type DWDirectiveRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DWDirectiveRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DWDirectiveRule{}, &DWDirectiveRuleList{})
}
