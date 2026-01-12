# Kubernetes Operator - Hello World Example

A minimal Kubernetes Operator that demonstrates the operator pattern. When you create a `Greeting` custom resource, the operator automatically creates a Pod that prints your greeting message.

## What is a Kubernetes Operator?

A Kubernetes Operator extends Kubernetes to manage custom applications. It consists of:

1. **Custom Resource Definition (CRD)** - Defines a new resource type (like `Greeting`)
2. **Controller** - Watches for changes and reconciles the desired state

```
┌─────────────────────────────────────────────────────────────────┐
│                    OPERATOR PATTERN                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   User creates:                    Operator creates:             │
│   ┌──────────────┐                ┌──────────────┐              │
│   │   Greeting   │  ──watches──►  │     Pod      │              │
│   │              │                │              │              │
│   │ message:     │  ──creates──►  │ echo "Hello" │              │
│   │ "Hello!"     │                │              │              │
│   └──────────────┘                └──────────────┘              │
│                                                                  │
│   The operator continuously reconciles:                          │
│   - If Greeting exists but Pod doesn't → Create Pod              │
│   - If Greeting is deleted → Pod is garbage collected            │
│   - If Pod dies → Operator recreates it                          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
k8s-operator/
├── app/                           # Operator source code
│   ├── main.go                    # Entry point - sets up the manager
│   ├── api/v1/                    # CRD type definitions
│   │   ├── greeting_types.go      # Greeting struct (Spec & Status)
│   │   ├── groupversion_info.go   # API group registration
│   │   └── zz_generated_deepcopy.go
│   ├── controllers/
│   │   └── greeting_controller.go # Reconciliation logic
│   ├── go.mod / go.sum
│   └── Dockerfile
│
├── k8s/operator/                  # Kubernetes manifests for operator
│   ├── crd.yaml                   # CustomResourceDefinition
│   ├── rbac.yaml                  # Permissions (ServiceAccount, ClusterRole)
│   ├── deployment.yaml            # Operator deployment
│   └── example-greeting.yaml      # Example Greeting resources
│
├── terraform/                     # Infrastructure as Code
│   └── *.tf                       # Kind cluster configuration
│
└── k8s/                           # (Legacy) Simple app manifests
```

## How the Code Works

### 1. CRD Types (`api/v1/greeting_types.go`)

Defines what a `Greeting` resource looks like:

```go
type GreetingSpec struct {
    Message string `json:"message,omitempty"`  // User provides this
}

type GreetingStatus struct {
    PodName string `json:"podName,omitempty"`  // Operator fills this
    Ready   bool   `json:"ready,omitempty"`    // Operator updates this
}
```

### 2. Controller (`controllers/greeting_controller.go`)

The reconciliation loop - called whenever a Greeting changes:

```go
func (r *GreetingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch the Greeting resource
    // 2. Check if Pod already exists
    // 3. If not, create a Pod that echoes the message
    // 4. Update Greeting status with Pod name and ready state
    // 5. Return (Kubernetes will call us again if anything changes)
}
```

### 3. Main (`main.go`)

Sets up the controller-runtime manager:

```go
func main() {
    // 1. Create a manager (handles leader election, metrics, etc.)
    // 2. Register our Greeting controller
    // 3. Start the manager (blocks forever, processing events)
}
```

---

## Quick Start (How to Run)

### Prerequisites

- Docker
- Terraform
- kubectl
- kind

### Step 1: Create the Cluster

```bash
cd terraform
terraform init
terraform apply -auto-approve
```

### Step 2: Build and Deploy the Operator

```bash
# Build the operator image
cd ../app
docker build -t greeting-operator:latest .

# Load into kind
kind load docker-image greeting-operator:latest --name local-k8s-cluster

# Deploy CRD, RBAC, and Operator
kubectl apply -f ../k8s/operator/crd.yaml
kubectl apply -f ../k8s/operator/rbac.yaml
kubectl apply -f ../k8s/operator/deployment.yaml

# Wait for operator to be ready
kubectl rollout status deployment/greeting-operator -n hello-app
```

### Step 3: Create a Greeting

```bash
# Create example greetings
kubectl apply -f ../k8s/operator/example-greeting.yaml

# Or create your own
kubectl apply -f - <<EOF
apiVersion: hello.example.com/v1
kind: Greeting
metadata:
  name: my-greeting
  namespace: hello-app
spec:
  message: "Hello from my first operator!"
EOF
```

### Step 4: Watch the Magic

```bash
# See your Greetings
kubectl get greetings -n hello-app

# Output:
# NAME                MESSAGE                                   POD                          READY
# my-greeting         Hello from my first operator!             greeting-my-greeting         true

# See the pods the operator created
kubectl get pods -n hello-app -l app=greeting

# Check the pod's output
kubectl logs greeting-my-greeting -n hello-app
# Output: Hello from my first operator!
```

### Step 5: Watch Operator Logs

```bash
# See the operator reconciling
kubectl logs -l app=greeting-operator -n hello-app -f
```

---

## Understanding the Flow

```
1. You apply a Greeting YAML
   │
   ▼
2. Kubernetes API stores the Greeting resource
   │
   ▼
3. Operator's controller sees the new Greeting (via Watch)
   │
   ▼
4. Reconcile() is called
   │
   ├─► Checks if Pod exists → NO
   │
   ▼
5. Creates a Pod with: echo "<your message>" && sleep 3600
   │
   ▼
6. Updates Greeting.status.podName and .status.ready
   │
   ▼
7. Done! (Until something changes)
```

**If you delete the Greeting:**
- Kubernetes garbage collection deletes the Pod (owner reference)

**If the Pod crashes:**
- Operator sees the change and recreates it

---

## Useful Commands

```bash
# List all greetings
kubectl get greetings -n hello-app

# Describe a greeting (shows events)
kubectl describe greeting my-greeting -n hello-app

# Delete a greeting (Pod will be auto-deleted)
kubectl delete greeting my-greeting -n hello-app

# Watch operator logs
kubectl logs -l app=greeting-operator -n hello-app -f

# Check CRD is installed
kubectl get crd greetings.hello.example.com
```

## Cleanup

```bash
# Delete all greetings
kubectl delete greetings --all -n hello-app

# Remove the operator
kubectl delete -f k8s/operator/deployment.yaml
kubectl delete -f k8s/operator/rbac.yaml
kubectl delete -f k8s/operator/crd.yaml

# Destroy the cluster
cd terraform && terraform destroy -auto-approve
```

## Next Steps

To build a real operator, consider:

1. **kubebuilder** - Scaffolding tool that generates boilerplate
2. **operator-sdk** - Red Hat's framework with OLM integration
3. Add more complex reconciliation (Deployments, Services, ConfigMaps)
4. Add validation webhooks
5. Add status conditions for better observability
