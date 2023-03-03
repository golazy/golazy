package viewtest

import (
	"github.com/pmezard/go-difflib/difflib"
)

func TextDiff(output, expectation string) (diff *string, err error) {
	d := difflib.UnifiedDiff{
		A:        difflib.SplitLines(expectation),
		B:        difflib.SplitLines(output),
		FromFile: "Expectation",
		ToFile:   "Output",
		Context:  3,
	}
	text, err := difflib.GetUnifiedDiffString(d)
	if err != nil {
		return nil, err
	}
	if text == "" {
		return nil, nil
	}
	return &text, nil

}
