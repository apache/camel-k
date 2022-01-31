{{ define "type" }}

[#{{ sanitizeId (anchorIDForType .) }}]
=== {{ .Name.Name }}{{ if eq .Kind "Alias" }}(`{{.Underlying}}` alias){{ end }}
{{- with (typeReferences .) }}

*Appears on:*
{{ range . }}
* <<#{{ sanitizeId (anchorIDForType .) }}, {{ typeDisplayName . }}>>
{{- end -}}
{{- end }}

{{ renderComments .CommentLines }}
{{ if .Members }}
[cols="2,2a",options="header"]
|===
|Field
|Description
{{ if isExportedType . }}
|`apiVersion` +
string
|`{{apiGroup .}}`

|`kind` +
string
|`{{.Name.Name}}`
{{- end }}
{{ template "members" . }}
|===
{{- end -}}

{{- end -}}
