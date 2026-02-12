package controller

import (
	"context"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	hellov1 "hello-operator/api/v1"
)

const (
	defaultRequeueInterval = 5 * time.Minute
)

// SecurityPolicyReconciler reconciles a SecurityPolicy object
type SecurityPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=hello.example.com,resources=securitypolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hello.example.com,resources=securitypolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=pods/log,verbs=get
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

// Reconcile handles the reconciliation loop for SecurityPolicy resources
func (r *SecurityPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the SecurityPolicy instance
	policy := &hellov1.SecurityPolicy{}
	if err := r.Get(ctx, req.NamespacedName, policy); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("SecurityPolicy resource not found, ignoring")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get SecurityPolicy")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling SecurityPolicy", "name", policy.Name)

	now := metav1.Now()
	status := hellov1.SecurityPolicyStatus{
		LastScanTime:     &now,
		ComplianceStatus: "Scanning",
	}

	totalChecks := 0
	failedChecks := 0

	// 1. Image vulnerability scanning
	if policy.Spec.ImageScanning.Enabled {
		scanner := NewImageScanner(r.Client, r.Scheme)
		vulnSummary, err := scanner.ScanImages(ctx, policy)
		if err != nil {
			logger.Error(err, "Failed to scan images")
		} else {
			status.VulnerabilitySummary = vulnSummary
			totalChecks += vulnSummary.ScannedImages
			if vulnSummary.Critical > 0 || vulnSummary.High > 0 {
				failedChecks += vulnSummary.Critical + vulnSummary.High
			}
		}
	}

	// 2. Network policy enforcement
	if policy.Spec.NetworkPolicy.Enabled {
		enforcer := NewNetworkEnforcer(r.Client, r.Scheme)
		enforced, err := enforcer.EnforceNetworkPolicies(ctx, policy)
		if err != nil {
			logger.Error(err, "Failed to enforce network policies")
		} else {
			status.NetworkPoliciesEnforced = enforced
			totalChecks += enforced
		}
	}

	// 3. Pod security enforcement
	if policy.Spec.PodSecurity.Enabled {
		enforcer := NewNetworkEnforcer(r.Client, r.Scheme)
		checker := NewPodSecurityChecker(r.Client, enforcer)
		violations, err := checker.CheckPodSecurity(ctx, policy)
		if err != nil {
			logger.Error(err, "Failed to check pod security")
		} else {
			status.PodSecurityViolations = violations
			totalChecks += len(violations)
			failedChecks += len(violations)
		}
	}

	// Compute compliance summary
	passedChecks := totalChecks - failedChecks
	if passedChecks < 0 {
		passedChecks = 0
	}
	status.ComplianceSummary = hellov1.ComplianceSummary{
		TotalChecks:  totalChecks,
		PassedChecks: passedChecks,
		FailedChecks: failedChecks,
	}

	if failedChecks > 0 {
		status.ComplianceStatus = "NonCompliant"
	} else {
		status.ComplianceStatus = "Compliant"
	}

	// Update status
	policy.Status = status
	if err := r.Status().Update(ctx, policy); err != nil {
		logger.Error(err, "Failed to update SecurityPolicy status")
		return ctrl.Result{}, err
	}

	logger.Info("SecurityPolicy reconciled",
		"compliance", status.ComplianceStatus,
		"totalChecks", totalChecks,
		"failedChecks", failedChecks,
	)

	return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
// The controller watches SecurityPolicy resources and owns Jobs and NetworkPolicies.
// Pod changes are picked up via the periodic requeue interval.
func (r *SecurityPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hellov1.SecurityPolicy{}).
		Owns(&batchv1.Job{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Complete(r)
}
