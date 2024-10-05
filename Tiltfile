trigger_mode(TRIGGER_MODE_MANUAL)
allow_k8s_contexts('orbstack')

local_resource(
  'compile',
  'CGO_ENABLED=0 GOOS=linux GOARCH=$(go env GOARCH) go build -o bin/cert-manager-webhook-gandi ./cmd',
  deps=[
    'cmd/',
    'internal/',
    'go.mod',
    'go.sum'
  ],
  labels = ['native-processes'],
  trigger_mode = TRIGGER_MODE_AUTO
)
docker_build(
  'ghcr.io/krancovia/cert-manager-webhook-gandi',
  '.',
  only = ['bin/cert-manager-webhook-gandi'],
  target = 'dev', # Binary built natively, copied to the image
)

k8s_yaml(
  helm(
    './chart',
    name = 'cert-manager-webhook-gandi',
    namespace = 'cert-manager'
  )
)

k8s_resource(
  new_name = 'kube-system',
  labels = ['cert-manager-webhook-gandi'],
  objects = [
    # Registers our API extension with the Kubernetes API server
    'v1alpha1.acme.krancovia.io:apiservice'
  ]
)

k8s_resource(
  new_name = 'cert-manager',
  labels = ['cert-manager-webhook-gandi'],
  objects = [
    # Allows cert-manager to use our API extension
    'cert-manager\\:cert-manager-webhook-gandi:clusterrolebinding'
  ]
)

k8s_resource(
  workload = 'cert-manager-webhook-gandi',
  new_name = 'webhook-server',
  labels = ['cert-manager-webhook-gandi'],
  objects = [
    'cert-manager-webhook-gandi:serviceaccount',
    'cert-manager-webhook-gandi:clusterrole',
    'cert-manager-webhook-gandi:clusterrolebinding',
    # Allows the webhook server to read the extension-apiserver-authentication ConfigMap:
    'cert-manager-webhook-gandi\\:extension-apiserver-authentication-reader:rolebinding',  
    # Allows the webhook server to delegate auth decisions to the Kubernetes API server:
    'cert-manager-webhook-gandi\\:auth-delegator:clusterrolebinding',
    # To be bound to authorized users of our API extension, i.e. cert-manager:
    'cert-manager-webhook-gandi\\:domain-solver:clusterrole',
    # Used to create a self-signed CA cert:
    'cert-manager-webhook-gandi-selfsign:issuer',
    # Used to sign the webhook server's own cert:
    'cert-manager-webhook-gandi-ca-cert:certificate',
    'cert-manager-webhook-gandi-ca:issuer',
    # The webhook server's cert:
    'cert-manager-webhook-gandi:certificate'
  ]
)
