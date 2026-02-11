//revive:disable-next-line:var-naming package name is a project convention
package utils

import "regexp"

var wordRegexp = regexp.MustCompile(`\w+`)

func FirstWord(s string) string {
	words := wordRegexp.FindAllString(s, 1)
	if len(words) == 0 {
		return ""
	}
	return words[0]
}
