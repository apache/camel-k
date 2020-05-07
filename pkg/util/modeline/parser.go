package modeline

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

var (
	commonModelineRegexp = regexp.MustCompile(`^\s*//\s*camel-k\s*:\s*([^\s]+.*)$`)
	yamlModelineRegexp   = regexp.MustCompile(`^\s*#+\s*camel-k\s*:\s*([^\s]+.*)$`)
	xmlModelineRegexp    = regexp.MustCompile(`^.*<!--\s*camel-k\s*:\s*([^\s]+[^>]*)-->.*$`)

	delimiter = regexp.MustCompile(`\s+`)
)

func Parse(name, content string) (res []Option, err error) {
	lang := inferLanguage(name)
	if lang == "" {
		return nil, fmt.Errorf("unsupported file type %s", name)
	}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		res = append(res, getModelineOptions(scanner.Text(), lang)...)
	}
	return res, scanner.Err()
}

func getModelineOptions(line string, lang v1.Language) (res []Option) {
	reg := modelineRegexp(lang)
	if !reg.MatchString(line) {
		return nil
	}
	strs := reg.FindStringSubmatch(line)
	if len(strs) == 2 {
		tokens := delimiter.Split(strs[1], -1)
		for _, token := range tokens {
			if len(strings.Trim(token, "\t\n\f\r ")) == 0 {
				continue
			}
			eq := strings.Index(token, "=")
			var name, value string
			if eq > 0 {
				name = token[0:eq]
				value = token[eq+1:]
			} else {
				name = token
				value = ""
			}
			opt := Option{
				Name:  name,
				Value: value,
			}
			res = append(res, opt)
		}
	}
	return res
}

func modelineRegexp(lang v1.Language) *regexp.Regexp {
	switch lang {
	case v1.LanguageYaml:
		return yamlModelineRegexp
	case v1.LanguageXML:
		return xmlModelineRegexp
	default:
		return commonModelineRegexp
	}
}

func inferLanguage(fileName string) v1.Language {
	for _, l := range v1.Languages {
		if strings.HasSuffix(fileName, fmt.Sprintf(".%s", string(l))) {
			return l
		}
	}
	if strings.HasSuffix(fileName, ".yml") {
		return v1.LanguageYaml
	}
	return ""
}
