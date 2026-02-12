package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	hellov1 "hello-operator/api/v1"
)

// GreetingReconciler reconciles a Greeting object
type GreetingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=hello.example.com,resources=greetings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hello.example.com,resources=greetings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile handles the reconciliation loop for Greeting resources
func (r *GreetingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Greeting instance
	greeting := &hellov1.Greeting{}
	if err := r.Get(ctx, req.NamespacedName, greeting); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Greeting resource not found, ignoring")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Greeting")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling Greeting", "name", greeting.Name, "message", greeting.Spec.Message)

	// Define the Pod name
	podName := fmt.Sprintf("greeting-%s", greeting.Name)

	// Check if Pod already exists
	existingPod := &corev1.Pod{}
	err := r.Get(ctx, client.ObjectKey{Name: podName, Namespace: greeting.Namespace}, existingPod)

	if err != nil && errors.IsNotFound(err) {
		// Create the Pod
		pod := r.createPodForGreeting(greeting, podName)

		// Set Greeting as the owner of the Pod
		if err := ctrl.SetControllerReference(greeting, pod, r.Scheme); err != nil {
			logger.Error(err, "Failed to set owner reference")
			return ctrl.Result{}, err
		}

		logger.Info("Creating Pod", "pod", podName)
		if err := r.Create(ctx, pod); err != nil {
			logger.Error(err, "Failed to create Pod")
			return ctrl.Result{}, err
		}

		// Update status
		greeting.Status.PodName = podName
		greeting.Status.Ready = false
		if err := r.Status().Update(ctx, greeting); err != nil {
			logger.Error(err, "Failed to update Greeting status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Pod")
		return ctrl.Result{}, err
	}

	// Pod exists, update status based on pod phase
	isReady := existingPod.Status.Phase == corev1.PodSucceeded ||
		existingPod.Status.Phase == corev1.PodRunning

	if greeting.Status.Ready != isReady || greeting.Status.PodName != podName {
		greeting.Status.PodName = podName
		greeting.Status.Ready = isReady
		if err := r.Status().Update(ctx, greeting); err != nil {
			logger.Error(err, "Failed to update Greeting status")
			return ctrl.Result{}, err
		}
	}

	logger.Info("Greeting reconciled", "pod", podName, "ready", isReady)
	return ctrl.Result{}, nil
}

// createPodForGreeting creates a Pod that prints the greeting message
func (r *GreetingReconciler) createPodForGreeting(greeting *hellov1.Greeting, podName string) *corev1.Pod {
	message := greeting.Spec.Message
	if message == "" {
		message = "Hello from Kubernetes Operator!"
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: greeting.Namespace,
			Labels: map[string]string{
				"app":      "greeting",
				"greeting": greeting.Name,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyOnFailure,
			Containers: []corev1.Container{
				{
					Name:    "greeter",
					Image:   "busybox",
					Command: []string{"sh", "-c"},
					Args:    []string{fmt.Sprintf("echo '%s' && sleep 3600", message)},
				},
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager
func (r *GreetingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hellov1.Greeting{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
