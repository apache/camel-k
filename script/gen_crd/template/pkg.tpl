{{ define "packages" -}}
{{ range .packages -}}

[#{{ sanitizeId (packageAnchorID .) }}]
== {{ packageDisplayName . }}

    {{- with (index .GoPackages 0 ) -}}
        {{- with .DocComments }}

{{ renderComments . }}
        {{- end -}}
    {{- end }}

==  Resource Types

    {{- range (visibleTypes (sortedTypes .Types)) -}}
        {{- if isExportedType . -}}
            {{- template "type" .  }}
        {{- end -}}
    {{- end }}

== Internal Types

    {{- range (visibleTypes (sortedTypes .Types)) -}}
        {{- if not (isExportedType .) -}}
            {{- template "type" .  }}
        {{- end -}}
    {{- end -}}

{{- end -}}

{{- end }}
