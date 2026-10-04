package format

import (
	"reflect"

	"gopkg.in/yaml.v3"
)

// verifyEdit reports whether the edited frontmatter decodes to the original
// mapping with exactly the given keys set to the given scalars. It catches key
// spellings the line editor does not match (quoted keys, `key :`, explicit
// `?` keys), which would otherwise produce a duplicate key.
func verifyEdit(original, edited []byte, values map[string]string) bool {
	before, ok := decodeMapping(original)
	if !ok {
		return false
	}
	after, ok := decodeMapping(edited)
	if !ok {
		return false
	}
	for k, v := range values {
		var want any
		if err := yaml.Unmarshal([]byte(v), &want); err != nil {
			return false
		}
		before[k] = want
	}
	return reflect.DeepEqual(before, after)
}

func decodeMapping(content []byte) (map[string]any, bool) {
	fm, ok := locate(content)
	if !ok {
		return nil, false
	}
	m := map[string]any{}
	if err := yaml.Unmarshal(fm.yamlText(), &m); err != nil {
		return nil, false
	}
	return m, true
}
