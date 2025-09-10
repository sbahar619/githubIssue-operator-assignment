{{/*
Expand the name of the chart.
*/}}
{{- define "github-issue-operator.name" -}}
{{- .Chart.Name }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "github-issue-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "github-issue-operator.labels" -}}
helm.sh/chart: {{ include "github-issue-operator.chart" . }}
{{ include "github-issue-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "github-issue-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "github-issue-operator.name" . }}
control-plane: controller-manager
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "github-issue-operator.serviceAccountName" -}}
{{- printf "%s-controller-manager" (include "github-issue-operator.name" .) }}
{{- end }}

{{/*
Create the namespace
*/}}
{{- define "github-issue-operator.namespace" -}}
{{- printf "%s-system" (include "github-issue-operator.name" .) }}
{{- end }}

{{/*
Create the deployment name
*/}}
{{- define "github-issue-operator.deploymentName" -}}
{{- printf "%s-controller-manager" (include "github-issue-operator.name" .) }}
{{- end }}

{{/*
Create the secret name
*/}}
{{- define "github-issue-operator.secretName" -}}
{{- printf "%s-token-secret" (include "github-issue-operator.name" .) }}
{{- end }}

{{/*
Create the metrics service name
*/}}
{{- define "github-issue-operator.metricsServiceName" -}}
{{- printf "%s-controller-manager-metrics-service" (include "github-issue-operator.name" .) }}
{{- end }}
