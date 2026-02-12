package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	hellov1 "hello-operator/api/v1"
)

const (
	networkPolicyLabelKey = "security.example.com/managed-by"
	networkPolicyOwner    = "security.example.com/policy-name"
)

// NetworkEnforcer manages NetworkPolicy resources based on SecurityPolicy declarations
type NetworkEnforcer struct {
	client client.Client
	scheme *runtime.Scheme
}

// NewNetworkEnforcer creates a new NetworkEnforcer
func NewNetworkEnforcer(c client.Client, scheme *runtime.Scheme) *NetworkEnforcer {
	return &NetworkEnforcer{client: c, scheme: scheme}
}

// EnforceNetworkPolicies creates or updates NetworkPolicies based on the SecurityPolicy spec
func (n *NetworkEnforcer) EnforceNetworkPolicies(ctx context.Context, policy *hellov1.SecurityPolicy) (int, error) {
	logger := log.FromContext(ctx)

	namespaces := policy.Spec.TargetNamespaces
	if len(namespaces) == 0 {
		namespaces = []string{policy.Namespace}
	}

	enforced := 0
	desiredNames := make(map[string]bool)

	// Create/update a NetworkPolicy for each allowed dependency
	for i, dep := range policy.Spec.NetworkPolicy.AllowedDependencies {
		for _, ns := range namespaces {
			npName := fmt.Sprintf("secpol-%s-%d", policy.Name, i)
			desiredNames[npName] = true

			np := n.buildNetworkPolicy(npName, ns, policy, dep)

			// Set owner reference for garbage collection
			if ns == policy.Namespace {
				if err := ctrl.SetControllerReference(policy, np, n.scheme); err != nil {
					logger.Error(err, "Failed to set owner reference on NetworkPolicy", "name", npName)
				}
			}

			// Create or update the NetworkPolicy
			existing := &networkingv1.NetworkPolicy{}
			err := n.client.Get(ctx, client.ObjectKey{Name: npName, Namespace: ns}, existing)
			if err != nil && errors.IsNotFound(err) {
				logger.Info("Creating NetworkPolicy", "name", npName, "namespace", ns)
				if err := n.client.Create(ctx, np); err != nil {
					logger.Error(err, "Failed to create NetworkPolicy", "name", npName)
					continue
				}
				enforced++
			} else if err == nil {
				// Update existing
				existing.Spec = np.Spec
				existing.Labels = np.Labels
				if err := n.client.Update(ctx, existing); err != nil {
					logger.Error(err, "Failed to update NetworkPolicy", "name", npName)
					continue
				}
				enforced++
			} else {
				logger.Error(err, "Failed to get NetworkPolicy", "name", npName)
			}
		}
	}

	// Clean up stale NetworkPolicies that are no longer in the spec
	for _, ns := range namespaces {
		if err := n.cleanupStaleNetworkPolicies(ctx, policy.Name, ns, desiredNames); err != nil {
			logger.Error(err, "Failed to cleanup stale NetworkPolicies", "namespace", ns)
		}
	}

	return enforced, nil
}

// buildNetworkPolicy creates a NetworkPolicy from a ServiceDependency
func (n *NetworkEnforcer) buildNetworkPolicy(name, namespace string, policy *hellov1.SecurityPolicy, dep hellov1.ServiceDependency) *networkingv1.NetworkPolicy {
	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				networkPolicyLabelKey: "securitypolicy",
				networkPolicyOwner:   policy.Name,
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			// Apply to destination pods
			PodSelector: metav1.LabelSelector{
				MatchLabels: dep.To.MatchLabels,
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: dep.From.MatchLabels,
							},
						},
					},
				},
			},
		},
	}

	// Add port rules if specified
	if len(dep.Ports) > 0 {
		ports := make([]networkingv1.NetworkPolicyPort, len(dep.Ports))
		protocol := corev1.ProtocolTCP
		for i, port := range dep.Ports {
			p := intstr.FromInt32(port)
			ports[i] = networkingv1.NetworkPolicyPort{
				Port:     &p,
				Protocol: &protocol,
			}
		}
		np.Spec.Ingress[0].Ports = ports
	}

	return np
}

// cleanupStaleNetworkPolicies removes NetworkPolicies managed by a SecurityPolicy
// that are no longer in the desired set
func (n *NetworkEnforcer) cleanupStaleNetworkPolicies(ctx context.Context, policyName, namespace string, desiredNames map[string]bool) error {
	logger := log.FromContext(ctx)

	npList := &networkingv1.NetworkPolicyList{}
	if err := n.client.List(ctx, npList, client.InNamespace(namespace), client.MatchingLabels{
		networkPolicyLabelKey: "securitypolicy",
		networkPolicyOwner:   policyName,
	}); err != nil {
		return fmt.Errorf("listing managed NetworkPolicies: %w", err)
	}

	for i := range npList.Items {
		np := &npList.Items[i]
		if !desiredNames[np.Name] {
			logger.Info("Deleting stale NetworkPolicy", "name", np.Name, "namespace", namespace)
			if err := n.client.Delete(ctx, np); err != nil && !errors.IsNotFound(err) {
				logger.Error(err, "Failed to delete stale NetworkPolicy", "name", np.Name)
			}
		}
	}

	return nil
}

// CreateQuarantinePolicy creates a deny-all NetworkPolicy for an isolated pod
func (n *NetworkEnforcer) CreateQuarantinePolicy(ctx context.Context, pod *corev1.Pod) error {
	logger := log.FromContext(ctx)
	npName := fmt.Sprintf("quarantine-%s", pod.Name)
	if len(npName) > 63 {
		npName = npName[:63]
	}

	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      npName,
			Namespace: pod.Namespace,
			Labels: map[string]string{
				networkPolicyLabelKey: "quarantine",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"security.example.com/quarantined": "true",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			// Empty ingress and egress = deny all
			Ingress: []networkingv1.NetworkPolicyIngressRule{},
			Egress:  []networkingv1.NetworkPolicyEgressRule{},
		},
	}

	existing := &networkingv1.NetworkPolicy{}
	err := n.client.Get(ctx, client.ObjectKey{Name: npName, Namespace: pod.Namespace}, existing)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating quarantine NetworkPolicy", "pod", pod.Name)
		return n.client.Create(ctx, np)
	}
	return err
}
