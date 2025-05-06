{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "pac.common.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "pac.common.fullname" -}}
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
Kubernetes standard labels
*/}}
{{- define "pac.common.labels" -}}
app.kubernetes.io/name: {{ include "pac.common.name" . | quote }}
app.kubernetes.io/instance: {{ include "pac.common.fullname" . | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service | quote }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}

helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
{{- end -}}

{{/*
Labels to use on deploy.spec.selector.matchLabels and svc.spec.selector
*/}}
{{- define "pac.common.matchLabels" -}}
app.kubernetes.io/name: {{ include "pac.common.name" . | quote }}
app.kubernetes.io/instance: {{ include "pac.common.fullname" . | quote }}
{{- end -}}

{{/*
Provide an Fully Qualified Image name from an application name & values.
{{ include "pac.container.image" (dict "name" "app" "context" $) }}
- Image registry = either `"app".image.registry` or `global.image.registry`
- Image repository = `"app".image.repository`
- Image tag =
*/}}
{{- define "pac.container.image" -}}
{{- $appValues := get .context.Values .name -}}
{{- $registry := $appValues.image.registry | default .context.Values.global.image.registry -}}
{{- $tag := $appValues.image.tag | default .context.Values.global.image.tag -}}
{{- if $appValues.image.sha }}
{{- printf "%s/%s@sha256:%s" $registry $appValues.image.repository $appValues.image.sha }}
{{- else }}
{{- printf "%s/%s:%s" $registry $appValues.image.repository $tag }}
{{- end }}
{{- end -}}

{{/*
Security context for pods
*/}}
{{- define "pac.common.securityContext" }}
securityContext:
  fsGroup: 1337
  runAsUser: 1337
  runAsGroup: 1337
  runAsNonRoot: true
{{- end }}
