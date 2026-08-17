{{/* Name of the Secret holding the private key (existing or chart-managed). */}}
{{- define "github-exporter.privateKeySecret" -}}
{{- if .Values.privateKey.existingSecret -}}
{{ .Values.privateKey.existingSecret }}
{{- else -}}
github-exporter-private-key
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
