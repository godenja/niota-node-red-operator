{{/*
Expand the name of the chart.
*/}}
{{- define "niota-node-red-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Full name: release + chart, capped at 63 chars.
*/}}
{{- define "niota-node-red-operator.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Chart label (name + version).
*/}}
{{- define "niota-node-red-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to every resource.
*/}}
{{- define "niota-node-red-operator.labels" -}}
helm.sh/chart: {{ include "niota-node-red-operator.chart" . }}
{{ include "niota-node-red-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels used by the Deployment and its pods.
*/}}
{{- define "niota-node-red-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "niota-node-red-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ServiceAccount name.
*/}}
{{- define "niota-node-red-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
  {{- default (include "niota-node-red-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
  {{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Operator image reference (repository:tag).
The tag falls back to .Chart.AppVersion when values.operator.image.tag is empty.
*/}}
{{- define "niota-node-red-operator.image" -}}
{{- $tag := .Values.operator.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.operator.image.repository $tag }}
{{- end }}

{{/*
Namespace where all operator resources live.
*/}}
{{- define "niota-node-red-operator.namespace" -}}
{{- .Values.namespace | default "niota-system" }}
{{- end }}
