package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImageScanningConfig configures vulnerability scanning via Trivy
type ImageScanningConfig struct {
	// Enabled enables container image vulnerability scanning
	Enabled bool `json:"enabled"`
	// SeverityThreshold is the minimum severity to report (CRITICAL, HIGH, MEDIUM, LOW)
	SeverityThreshold string `json:"severityThreshold,omitempty"`
	// ScanInterval is how often to re-scan images (e.g. "24h", "1h")
	ScanInterval string `json:"scanInterval,omitempty"`
}

// ServiceSelector selects pods by labels
type ServiceSelector struct {
	// MatchLabels selects pods with these labels
	MatchLabels map[string]string `json:"matchLabels"`
}

// ServiceDependency declares an allowed network path between services
type ServiceDependency struct {
	// From selects the source pods
	From ServiceSelector `json:"from"`
	// To selects the destination pods
	To ServiceSelector `json:"to"`
	// Ports are the allowed destination ports
	Ports []int32 `json:"ports,omitempty"`
}

// NetworkPolicyConfig configures automatic NetworkPolicy enforcement
type NetworkPolicyConfig struct {
	// Enabled enables automatic NetworkPolicy creation
	Enabled bool `json:"enabled"`
	// AllowedDependencies declares permitted service-to-service communication
	AllowedDependencies []ServiceDependency `json:"allowedDependencies,omitempty"`
}

// PodSecurityConfig configures pod security enforcement
type PodSecurityConfig struct {
	// Enabled enables pod security scanning
	Enabled bool `json:"enabled"`
	// EnforceNonRoot detects and quarantines pods running as root
	EnforceNonRoot bool `json:"enforceNonRoot,omitempty"`
	// EnforceNoPrivilegeEscalation detects pods with privilege escalation
	EnforceNoPrivilegeEscalation bool `json:"enforceNoPrivilegeEscalation,omitempty"`
	// QuarantineAction is the action to take on violating pods: "label" or "isolate"
	QuarantineAction string `json:"quarantineAction,omitempty"`
}

// SecurityPolicySpec defines the desired state of SecurityPolicy
type SecurityPolicySpec struct {
	// ImageScanning configures vulnerability scanning
	ImageScanning ImageScanningConfig `json:"imageScanning,omitempty"`
	// NetworkPolicy configures automatic network policy enforcement
	NetworkPolicy NetworkPolicyConfig `json:"networkPolicy,omitempty"`
	// PodSecurity configures pod security enforcement
	PodSecurity PodSecurityConfig `json:"podSecurity,omitempty"`
	// TargetNamespaces selects which namespaces to monitor
	TargetNamespaces []string `json:"targetNamespaces,omitempty"`
}

// ImageScanResult holds scan results for a single container image
type ImageScanResult struct {
	// Image is the container image reference
	Image string `json:"image"`
	// Critical is the count of critical vulnerabilities
	Critical int `json:"critical"`
	// High is the count of high vulnerabilities
	High int `json:"high"`
	// Medium is the count of medium vulnerabilities
	Medium int `json:"medium"`
	// Low is the count of low vulnerabilities
	Low int `json:"low"`
	// LastScanned is when this image was last scanned
	LastScanned *metav1.Time `json:"lastScanned,omitempty"`
}

// VulnerabilitySummary summarizes vulnerability scan results
type VulnerabilitySummary struct {
	// ScannedImages is the total number of images scanned
	ScannedImages int `json:"scannedImages"`
	// Critical is the total count of critical vulnerabilities
	Critical int `json:"critical"`
	// High is the total count of high vulnerabilities
	High int `json:"high"`
	// Medium is the total count of medium vulnerabilities
	Medium int `json:"medium"`
	// Low is the total count of low vulnerabilities
	Low int `json:"low"`
	// ImageResults holds per-image scan results
	ImageResults []ImageScanResult `json:"imageResults,omitempty"`
}

// PodSecurityViolation records a security violation found on a pod
type PodSecurityViolation struct {
	// PodName is the name of the violating pod
	PodName string `json:"podName"`
	// Namespace is the namespace of the violating pod
	Namespace string `json:"namespace"`
	// Violation describes the security violation
	Violation string `json:"violation"`
	// Action is the quarantine action taken
	Action string `json:"action"`
	// DetectedAt is when the violation was detected
	DetectedAt *metav1.Time `json:"detectedAt,omitempty"`
}

// ComplianceSummary provides an overview of all compliance checks
type ComplianceSummary struct {
	// TotalChecks is the total number of checks performed
	TotalChecks int `json:"totalChecks"`
	// PassedChecks is the number of checks that passed
	PassedChecks int `json:"passedChecks"`
	// FailedChecks is the number of checks that failed
	FailedChecks int `json:"failedChecks"`
}

// SecurityPolicyStatus defines the observed state of SecurityPolicy (compliance report)
type SecurityPolicyStatus struct {
	// LastScanTime is when the last full scan was performed
	LastScanTime *metav1.Time `json:"lastScanTime,omitempty"`
	// ComplianceStatus is the overall compliance status: Compliant, NonCompliant, or Scanning
	ComplianceStatus string `json:"complianceStatus,omitempty"`
	// VulnerabilitySummary holds vulnerability scan results
	VulnerabilitySummary VulnerabilitySummary `json:"vulnerabilitySummary,omitempty"`
	// NetworkPoliciesEnforced is the number of NetworkPolicies managed
	NetworkPoliciesEnforced int `json:"networkPoliciesEnforced,omitempty"`
	// PodSecurityViolations lists detected pod security violations
	PodSecurityViolations []PodSecurityViolation `json:"podSecurityViolations,omitempty"`
	// ComplianceSummary provides a high-level compliance overview
	ComplianceSummary ComplianceSummary `json:"complianceSummary,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SecurityPolicy is the Schema for the securitypolicies API
type SecurityPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecurityPolicySpec   `json:"spec,omitempty"`
	Status SecurityPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecurityPolicyList contains a list of SecurityPolicy
type SecurityPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecurityPolicy `json:"items"`
}
