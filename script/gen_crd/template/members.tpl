{{ define "members" -}}

{{ range .Members -}}
  {{- if not (hiddenMember .) -}}
|`{{ fieldName . }}` +
{{ if linkForType .Type -}}
  {{- if isLocalType .Type -}}
*xref:{{ linkForType .Type}}[{{ asciiDocAttributeEscape (typeDisplayName .Type) }}]*
  {{- else -}}
*{{ linkForType .Type}}[{{ asciiDocAttributeEscape (typeDisplayName .Type) }}]*
  {{- end -}}
{{- else -}}
  {{- typeDisplayName .Type -}}
{{- end }}
|{{ if fieldEmbedded . -}}
(Members of `{{ fieldName . }}` are embedded into this type.)
{{- end }}
{{ if isOptionalMember . -}}
*(Optional)*
{{- end }}

{{ renderComments .CommentLines }}

{{ if and (eq (.Type.Name.Name) "ObjectMeta") -}}
Refer to the Kubernetes API documentation for the fields of the `metadata` field.
{{ end -}}
{{- end -}}
{{- end -}}

{{- end }}
