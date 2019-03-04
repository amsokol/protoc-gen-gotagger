package tagger

import (
	"unicode"
)

func (p *plugin) toGolangStructName(parents []string, name string) string {
	names := make([]string, len(parents), len(parents)+1)
	copy(names, parents)
	names = append(names, name)

	var n string
	for _, v := range names {
		var uppercased bool
		r := []rune(v)
		if unicode.IsLower(r[0]) {
			uppercased = true
			r[0] = unicode.ToUpper(r[0])
		}
		v = string(r)
		if len(n) > 0 && !uppercased {
			n += "_"
		}

		n += v
	}

	return n
}
