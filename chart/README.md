<!-- Everything below this point is automatically generated using https://github.com/bitnami-labs/readme-generator-for-helm. Any manual changes will be overwritten on the next release -->
<!-- To make changes to this file, modify the comments in the values.yaml file and re-run readme-generator-for-helm -->

## Parameters

### Image Parameters

| Name               | Description                                                                  | Value                                          |
| ------------------ | ---------------------------------------------------------------------------- | ---------------------------------------------- |
| `image.repository` | Image repository of cert-manager-webhook-gandi                               | `ghcr.io/krancovia/cert-manager-webhook-gandi` |
| `image.tag`        | Overrides the image tag. The default tag is the value of `.Chart.AppVersion` | `""`                                           |
| `image.pullPolicy` | Image pull policy                                                            | `IfNotPresent`                                 |

### Deployment Parameters

| Name                          | Description                                 | Value |
| ----------------------------- | ------------------------------------------- | ----- |
| `deployment.additionalLabels` | Additional labels to add to the Deployment. | `{}`  |
| `deployment.annotations`      | Annotations to add to the Deployment.       | `{}`  |

### Pod Parameters

| Name                   | Description                                   | Value |
| ---------------------- | --------------------------------------------- | ----- |
| `pod.additionalLabels` | Additional labels to add to Pods.             | `{}`  |
| `pod.annotations`      | Annotations to add to Pods.                   | `{}`  |
| `pod.resources`        | Resources limits and requests for containers. | `{}`  |
| `pod.nodeSelector`     | Node selector for pods.                       | `{}`  |
| `pod.tolerations`      | Tolerations for pods.                         | `[]`  |
| `pod.affinity`         | Specifies pod affinity.                       | `{}`  |

