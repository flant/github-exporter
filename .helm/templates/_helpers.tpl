{{- define "github-exporter.name" -}}
github-exporter
{{- end -}}

{{- define "github-exporter.labels" -}}
app.kubernetes.io/name: {{ include "github-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "github-exporter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "github-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* Name of the Secret holding the private key (existing or chart-managed). */}}
{{- define "github-exporter.privateKeySecret" -}}
{{- if .Values.privateKey.existingSecret -}}
{{ .Values.privateKey.existingSecret }}
{{- else -}}
{{ include "github-exporter.name" . }}-private-key
{{- end -}}
{{- end -}}

{{/* Key within the Secret. */}}
{{- define "github-exporter.privateKeySecretKey" -}}
{{- if .Values.privateKey.existingSecret -}}
{{ .Values.privateKey.existingSecretKey }}
{{- else -}}
private-key.pem
{{- end -}}
{{- end -}}
