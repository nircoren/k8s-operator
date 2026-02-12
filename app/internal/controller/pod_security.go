package controller

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	hellov1 "hello-operator/api/v1"
)

const (
	quarantineLabel      = "security.example.com/quarantined"
	violationAnnotation  = "security.example.com/violations"
)

// PodSecurityChecker detects and quarantines pods with security violations
type PodSecurityChecker struct {
	client   client.Client
	enforcer *NetworkEnforcer
}

// NewPodSecurityChecker creates a new PodSecurityChecker
func NewPodSecurityChecker(c client.Client, enforcer *NetworkEnforcer) *PodSecurityChecker {
	return &PodSecurityChecker{client: c, enforcer: enforcer}
}

// CheckPodSecurity scans all pods in target namespaces for security violations
func (p *PodSecurityChecker) CheckPodSecurity(ctx context.Context, policy *hellov1.SecurityPolicy) ([]hellov1.PodSecurityViolation, error) {
	logger := log.FromContext(ctx)

	namespaces := policy.Spec.TargetNamespaces
	if len(namespaces) == 0 {
		namespaces = []string{policy.Namespace}
	}

	var violations []hellov1.PodSecurityViolation
	now := metav1.Now()

	for _, ns := range namespaces {
		podList := &corev1.PodList{}
		if err := p.client.List(ctx, podList, client.InNamespace(ns)); err != nil {
			logger.Error(err, "Failed to list pods", "namespace", ns)
			continue
		}

		for i := range podList.Items {
			pod := &podList.Items[i]
			podViolations := p.checkPod(policy, pod)

			if len(podViolations) == 0 {
				continue
			}

			action := policy.Spec.PodSecurity.QuarantineAction
			if action == "" {
				action = "label"
			}

			// Quarantine the pod
			if err := p.quarantinePod(ctx, pod, podViolations, action); err != nil {
				logger.Error(err, "Failed to quarantine pod", "pod", pod.Name, "namespace", ns)
			}

			for _, v := range podViolations {
				violations = append(violations, hellov1.PodSecurityViolation{
					PodName:    pod.Name,
					Namespace:  ns,
					Violation:  v,
					Action:     action,
					DetectedAt: &now,
				})
			}
		}
	}

	return violations, nil
}

// checkPod inspects a single pod for security violations
func (p *PodSecurityChecker) checkPod(policy *hellov1.SecurityPolicy, pod *corev1.Pod) []string {
	var violations []string

	// Check pod-level security context
	if pod.Spec.SecurityContext != nil {
		if policy.Spec.PodSecurity.EnforceNonRoot {
			if pod.Spec.SecurityContext.RunAsUser != nil && *pod.Spec.SecurityContext.RunAsUser == 0 {
				violations = append(violations, "pod runs as root (runAsUser=0)")
			}
			if pod.Spec.SecurityContext.RunAsNonRoot != nil && !*pod.Spec.SecurityContext.RunAsNonRoot {
				violations = append(violations, "pod does not enforce non-root (runAsNonRoot=false)")
			}
		}
	}

	// Check each container's security context
	allContainers := append(pod.Spec.Containers, pod.Spec.InitContainers...)
	for _, container := range allContainers {
		if container.SecurityContext == nil {
			continue
		}
		sc := container.SecurityContext

		if policy.Spec.PodSecurity.EnforceNonRoot {
			if sc.RunAsUser != nil && *sc.RunAsUser == 0 {
				violations = append(violations, fmt.Sprintf("container %q runs as root (runAsUser=0)", container.Name))
			}
			if sc.Privileged != nil && *sc.Privileged {
				violations = append(violations, fmt.Sprintf("container %q runs in privileged mode", container.Name))
			}
		}

		if policy.Spec.PodSecurity.EnforceNoPrivilegeEscalation {
			if sc.AllowPrivilegeEscalation != nil && *sc.AllowPrivilegeEscalation {
				violations = append(violations, fmt.Sprintf("container %q allows privilege escalation", container.Name))
			}
		}
	}

	return violations
}

// quarantinePod applies quarantine actions to a violating pod
func (p *PodSecurityChecker) quarantinePod(ctx context.Context, pod *corev1.Pod, violations []string, action string) error {
	logger := log.FromContext(ctx)

	// Check if already quarantined
	if pod.Labels != nil && pod.Labels[quarantineLabel] == "true" {
		return nil
	}

	// Add quarantine label and violation annotation
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[quarantineLabel] = "true"

	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations[violationAnnotation] = strings.Join(violations, "; ")

	if err := p.client.Update(ctx, pod); err != nil {
		return fmt.Errorf("updating pod labels: %w", err)
	}
	logger.Info("Quarantined pod", "pod", pod.Name, "namespace", pod.Namespace, "action", action, "violations", violations)

	// If action is "isolate", also create a deny-all NetworkPolicy
	if action == "isolate" {
		if err := p.enforcer.CreateQuarantinePolicy(ctx, pod); err != nil {
			return fmt.Errorf("creating quarantine network policy: %w", err)
		}
	}

	return nil
}
