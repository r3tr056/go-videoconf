{{/*
Expand the name of the chart.
*/}}
{{- define "videoconf.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Full release-qualified name.
*/}}
{{- define "videoconf.fullname" -}}
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
Common labels.
*/}}
{{- define "videoconf.labels" -}}
helm.sh/chart: {{ include "videoconf.chart" . }}
{{ include "videoconf.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "videoconf.selectorLabels" -}}
app.kubernetes.io/name: {{ include "videoconf.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Chart name and version label.
*/}}
{{- define "videoconf.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Secret name: a pre-created Secret when secrets.existingSecret is set,
otherwise the chart-managed one.
*/}}
{{- define "videoconf.secretName" -}}
{{- .Values.secrets.existingSecret | default (printf "%s-secrets" (include "videoconf.fullname" .)) }}
{{- end }}

{{/*
First-party image reference: registry/repository/name:tag.
*/}}
{{- define "videoconf.image" -}}
{{- $tag := .tag | default .root.Values.image.tag -}}
{{- printf "%s/%s/%s:%s" .root.Values.global.imageRegistry .root.Values.global.imageRepository .name $tag }}
{{- end }}
