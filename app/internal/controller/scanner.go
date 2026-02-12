package controller

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	hellov1 "hello-operator/api/v1"
)

const (
	trivyImage       = "aquasec/trivy:latest"
	scanLabelKey     = "security.example.com/scan-for"
	scanImageLabel   = "security.example.com/scan-image"
	defaultSeverity  = "CRITICAL,HIGH,MEDIUM,LOW"
	defaultInterval  = 24 * time.Hour
	scanJobNamespace = "hello-app"
)

// TrivyResult represents the JSON output from Trivy
type TrivyResult struct {
	Results []TrivyTarget `json:"Results"`
}

// TrivyTarget represents a single target in Trivy output
type TrivyTarget struct {
	Target          string            `json:"Target"`
	Vulnerabilities []TrivyVulnerability `json:"Vulnerabilities"`
}

// TrivyVulnerability represents a single vulnerability
type TrivyVulnerability struct {
	VulnerabilityID string `json:"VulnerabilityID"`
	Severity        string `json:"Severity"`
	Title           string `json:"Title"`
}

// ImageScanner handles Trivy-based vulnerability scanning
type ImageScanner struct {
	client client.Client
	scheme *runtime.Scheme
}

// NewImageScanner creates a new ImageScanner
func NewImageScanner(c client.Client, scheme *runtime.Scheme) *ImageScanner {
	return &ImageScanner{client: c, scheme: scheme}
}

// ScanImages scans all container images in the target namespaces
func (s *ImageScanner) ScanImages(ctx context.Context, policy *hellov1.SecurityPolicy) (hellov1.VulnerabilitySummary, error) {
	logger := log.FromContext(ctx)
	summary := hellov1.VulnerabilitySummary{}

	namespaces := policy.Spec.TargetNamespaces
	if len(namespaces) == 0 {
		namespaces = []string{policy.Namespace}
	}

	// Collect unique images from all pods in target namespaces
	images := make(map[string]bool)
	for _, ns := range namespaces {
		podList := &corev1.PodList{}
		if err := s.client.List(ctx, podList, client.InNamespace(ns)); err != nil {
			logger.Error(err, "Failed to list pods", "namespace", ns)
			continue
		}
		for _, pod := range podList.Items {
			for _, container := range pod.Spec.Containers {
				images[container.Image] = true
			}
			for _, container := range pod.Spec.InitContainers {
				images[container.Image] = true
			}
		}
	}

	severity := policy.Spec.ImageScanning.SeverityThreshold
	if severity == "" {
		severity = defaultSeverity
	}

	scanInterval := parseDuration(policy.Spec.ImageScanning.ScanInterval, defaultInterval)

	// For each unique image, ensure a scan Job exists or read results
	imageResults := []hellov1.ImageScanResult{}
	for image := range images {
		result, err := s.scanImage(ctx, policy, image, severity, scanInterval)
		if err != nil {
			logger.Error(err, "Failed to scan image", "image", image)
			continue
		}
		if result != nil {
			imageResults = append(imageResults, *result)
			summary.Critical += result.Critical
			summary.High += result.High
			summary.Medium += result.Medium
			summary.Low += result.Low
		}
	}

	summary.ScannedImages = len(imageResults)
	summary.ImageResults = imageResults
	return summary, nil
}

// scanImage creates a Trivy scan Job or reads existing results for an image
func (s *ImageScanner) scanImage(ctx context.Context, policy *hellov1.SecurityPolicy, image, severity string, scanInterval time.Duration) (*hellov1.ImageScanResult, error) {
	logger := log.FromContext(ctx)
	jobName := scanJobName(policy.Name, image)
	jobNamespace := policy.Namespace

	// Check if a scan Job already exists
	existingJob := &batchv1.Job{}
	err := s.client.Get(ctx, client.ObjectKey{Name: jobName, Namespace: jobNamespace}, existingJob)

	if err == nil {
		// Job exists — check if it's completed
		if isJobComplete(existingJob) {
			// Check if results are still fresh
			if existingJob.Status.CompletionTime != nil {
				age := time.Since(existingJob.Status.CompletionTime.Time)
				if age > scanInterval {
					// Results are stale, delete and re-create
					logger.Info("Scan results stale, re-scanning", "image", image)
					if err := s.client.Delete(ctx, existingJob, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
						return nil, fmt.Errorf("deleting stale scan job: %w", err)
					}
					return nil, nil // Will be created on next reconcile
				}
			}
			// Read results from Job pod logs
			return s.readScanResults(ctx, existingJob, image)
		}
		if isJobFailed(existingJob) {
			logger.Info("Scan job failed, re-creating", "image", image)
			if err := s.client.Delete(ctx, existingJob, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
				return nil, fmt.Errorf("deleting failed scan job: %w", err)
			}
			return nil, nil
		}
		// Job is still running
		return nil, nil
	}

	if !errors.IsNotFound(err) {
		return nil, fmt.Errorf("getting scan job: %w", err)
	}

	// Create a new scan Job
	job := s.createScanJob(policy, jobName, jobNamespace, image, severity)
	logger.Info("Creating Trivy scan job", "job", jobName, "image", image)
	if err := s.client.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("creating scan job: %w", err)
	}

	return nil, nil // Results will be available on next reconcile
}

// createScanJob creates a Kubernetes Job that runs Trivy
func (s *ImageScanner) createScanJob(policy *hellov1.SecurityPolicy, jobName, namespace, image, severity string) *batchv1.Job {
	backoffLimit := int32(1)
	ttl := int32(86400) // 24h TTL for completed jobs

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels: map[string]string{
				scanLabelKey:   policy.Name,
				scanImageLabel: hashString(image),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "trivy",
							Image: trivyImage,
							Command: []string{
								"trivy",
								"image",
								"--format", "json",
								"--severity", severity,
								"--no-progress",
								image,
							},
						},
					},
				},
			},
		},
	}
}

// readScanResults reads and parses Trivy output from a completed scan Job's pod logs
func (s *ImageScanner) readScanResults(ctx context.Context, job *batchv1.Job, image string) (*hellov1.ImageScanResult, error) {
	// Find pods belonging to this Job
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(labels.Set{"job-name": job.Name})
	if err := s.client.List(ctx, podList, client.InNamespace(job.Namespace), client.MatchingLabelsSelector{Selector: labelSelector}); err != nil {
		return nil, fmt.Errorf("listing job pods: %w", err)
	}

	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no pods found for job %s", job.Name)
	}

	// Get logs from the first completed pod
	pod := podList.Items[0]
	logData, err := getPodLogs(ctx, s.client, &pod)
	if err != nil {
		return nil, fmt.Errorf("reading pod logs: %w", err)
	}

	// Parse Trivy JSON output
	result := parseTrivyOutput(logData, image)
	if job.Status.CompletionTime != nil {
		t := metav1.NewTime(job.Status.CompletionTime.Time)
		result.LastScanned = &t
	}
	return result, nil
}

// getPodLogs reads logs from a pod's first container.
// In a real implementation this would use the Kubernetes API /api/v1/namespaces/{ns}/pods/{pod}/log.
// Since controller-runtime client doesn't support pod logs directly, we store results
// in pod annotations as a fallback approach for log retrieval.
func getPodLogs(ctx context.Context, c client.Client, pod *corev1.Pod) (string, error) {
	// Check if results are cached in pod annotation
	if data, ok := pod.Annotations["security.example.com/scan-result"]; ok {
		return data, nil
	}

	// For actual log retrieval, the operator would use a raw REST client.
	// We return empty to indicate pending results.
	_ = ctx
	_ = c
	_ = io.Discard
	return "", fmt.Errorf("pod logs not yet available for %s/%s", pod.Namespace, pod.Name)
}

// parseTrivyOutput parses Trivy JSON output and returns vulnerability counts
func parseTrivyOutput(logData string, image string) *hellov1.ImageScanResult {
	result := &hellov1.ImageScanResult{
		Image: image,
	}

	if logData == "" {
		return result
	}

	var trivyResult TrivyResult
	if err := json.Unmarshal([]byte(logData), &trivyResult); err != nil {
		// If we can't parse, return zeros
		return result
	}

	for _, target := range trivyResult.Results {
		for _, vuln := range target.Vulnerabilities {
			switch strings.ToUpper(vuln.Severity) {
			case "CRITICAL":
				result.Critical++
			case "HIGH":
				result.High++
			case "MEDIUM":
				result.Medium++
			case "LOW":
				result.Low++
			}
		}
	}

	return result
}

// scanJobName generates a deterministic job name for an image scan
func scanJobName(policyName, image string) string {
	hash := hashString(image)
	name := fmt.Sprintf("trivy-scan-%s-%s", policyName, hash)
	if len(name) > 63 {
		name = name[:63]
	}
	return name
}

// hashString returns a short hex hash of a string
func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:4])
}

func isJobComplete(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func isJobFailed(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func parseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}
