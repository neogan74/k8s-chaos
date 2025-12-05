{{/*
Expand the name of the chart.
*/}}
{{- define "k8s-chaos.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "k8s-chaos.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "k8s-chaos.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "k8s-chaos.labels" -}}
helm.sh/chart: {{ include "k8s-chaos.chart" . }}
{{ include "k8s-chaos.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "k8s-chaos.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k8s-chaos.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
control-plane: controller-manager
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "k8s-chaos.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "k8s-chaos.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the image name
*/}}
{{- define "k8s-chaos.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
Create the webhook service name
*/}}
{{- define "k8s-chaos.webhookServiceName" -}}
{{- printf "%s-webhook-service" (include "k8s-chaos.fullname" .) }}
{{- end }}

{{/*
Create the webhook certificate secret name
*/}}
{{- define "k8s-chaos.webhookCertSecretName" -}}
{{- printf "%s-webhook-cert" (include "k8s-chaos.fullname" .) }}
{{- end }}

{{/*
Return the appropriate apiVersion for RBAC
*/}}
{{- define "k8s-chaos.rbacApiVersion" -}}
{{- if .Capabilities.APIVersions.Has "rbac.authorization.k8s.io/v1" -}}
rbac.authorization.k8s.io/v1
{{- else -}}
rbac.authorization.k8s.io/v1beta1
{{- end -}}
{{- end -}}

{{/*
Return the history namespace
*/}}
{{- define "k8s-chaos.historyNamespace" -}}
{{- if .Values.history.namespace }}
{{- tpl .Values.history.namespace . }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}