package pingdom

import (
	"regexp"
)

type tagByLabel struct {
	LabelKey   string
	LabelValue string
	Formatted  int
}

func TagLabel(n string, f string) (tagByLabel, error) {
	regex, err := regexp.Compile(f)
	tl := tagByLabel{LabelKey: "", LabelValue: "", Formatted: 0}

	if err != nil {
		return tl, err
	}

	matches := regex.FindAllStringSubmatch(n, -1)

	if len(matches) > 0 {
		tl.LabelKey = matches[0][1]
		tl.LabelValue = matches[0][2]
		tl.Formatted = 1
	}
	return tl, nil
}
