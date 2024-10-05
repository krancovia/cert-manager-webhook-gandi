{{/* vim: set filetype=mustache: */}}

{{/*
Common labels
*/}}
{{- define "labels" -}}
helm.sh/chart: cert-manager-webhook-gandi-{{ .Chart.Version }}
{{ include "selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "selectorLabels" -}}
app.kubernetes.io/name: cert-manager-webhook-gandi
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
