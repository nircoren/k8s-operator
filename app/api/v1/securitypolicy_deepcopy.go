package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto copies the receiver into out
func (in *SecurityPolicy) DeepCopyInto(out *SecurityPolicy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy creates a deep copy of SecurityPolicy
func (in *SecurityPolicy) DeepCopy() *SecurityPolicy {
	if in == nil {
		return nil
	}
	out := new(SecurityPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object
func (in *SecurityPolicy) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies the receiver into out
func (in *SecurityPolicyList) DeepCopyInto(out *SecurityPolicyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]SecurityPolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates a deep copy of SecurityPolicyList
func (in *SecurityPolicyList) DeepCopy() *SecurityPolicyList {
	if in == nil {
		return nil
	}
	out := new(SecurityPolicyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object
func (in *SecurityPolicyList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies SecurityPolicySpec
func (in *SecurityPolicySpec) DeepCopyInto(out *SecurityPolicySpec) {
	*out = *in
	out.ImageScanning = in.ImageScanning
	in.NetworkPolicy.DeepCopyInto(&out.NetworkPolicy)
	out.PodSecurity = in.PodSecurity
	if in.TargetNamespaces != nil {
		in, out := &in.TargetNamespaces, &out.TargetNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopyInto copies SecurityPolicyStatus
func (in *SecurityPolicyStatus) DeepCopyInto(out *SecurityPolicyStatus) {
	*out = *in
	if in.LastScanTime != nil {
		in, out := &in.LastScanTime, &out.LastScanTime
		*out = (*in).DeepCopy()
	}
	in.VulnerabilitySummary.DeepCopyInto(&out.VulnerabilitySummary)
	if in.PodSecurityViolations != nil {
		in, out := &in.PodSecurityViolations, &out.PodSecurityViolations
		*out = make([]PodSecurityViolation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.ComplianceSummary = in.ComplianceSummary
}

// DeepCopyInto copies ImageScanningConfig
func (in *ImageScanningConfig) DeepCopyInto(out *ImageScanningConfig) {
	*out = *in
}

// DeepCopyInto copies NetworkPolicyConfig
func (in *NetworkPolicyConfig) DeepCopyInto(out *NetworkPolicyConfig) {
	*out = *in
	if in.AllowedDependencies != nil {
		in, out := &in.AllowedDependencies, &out.AllowedDependencies
		*out = make([]ServiceDependency, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto copies ServiceDependency
func (in *ServiceDependency) DeepCopyInto(out *ServiceDependency) {
	*out = *in
	in.From.DeepCopyInto(&out.From)
	in.To.DeepCopyInto(&out.To)
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]int32, len(*in))
		copy(*out, *in)
	}
}

// DeepCopyInto copies ServiceSelector
func (in *ServiceSelector) DeepCopyInto(out *ServiceSelector) {
	*out = *in
	if in.MatchLabels != nil {
		in, out := &in.MatchLabels, &out.MatchLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopyInto copies VulnerabilitySummary
func (in *VulnerabilitySummary) DeepCopyInto(out *VulnerabilitySummary) {
	*out = *in
	if in.ImageResults != nil {
		in, out := &in.ImageResults, &out.ImageResults
		*out = make([]ImageScanResult, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto copies ImageScanResult
func (in *ImageScanResult) DeepCopyInto(out *ImageScanResult) {
	*out = *in
	if in.LastScanned != nil {
		in, out := &in.LastScanned, &out.LastScanned
		*out = (*in).DeepCopy()
	}
}

// DeepCopyInto copies PodSecurityViolation
func (in *PodSecurityViolation) DeepCopyInto(out *PodSecurityViolation) {
	*out = *in
	if in.DetectedAt != nil {
		in, out := &in.DetectedAt, &out.DetectedAt
		*out = (*in).DeepCopy()
	}
}

// DeepCopyInto copies ComplianceSummary
func (in *ComplianceSummary) DeepCopyInto(out *ComplianceSummary) {
	*out = *in
}
