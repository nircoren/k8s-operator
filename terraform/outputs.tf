output "cluster_name" {
  description = "Name of the created kind cluster"
  value       = kind_cluster.default.name
}

output "kubeconfig" {
  description = "Path to the kubeconfig file"
  value       = kind_cluster.default.kubeconfig_path
  sensitive   = true
}

output "endpoint" {
  description = "Kubernetes API server endpoint"
  value       = kind_cluster.default.endpoint
}
