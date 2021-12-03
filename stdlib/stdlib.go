package stdlib

import (
	_ "embed"
	"encoding/json"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/google/go-jsonnet"
)

var (
	//go:embed html.libsonnet
	htmlLib string
	//go:embed stdlib-content.jsonnet
	stdLib string
)

type Function struct {
	Name                string      `json:"name"`
	Params              []string    `json:"params"`
	Description         interface{} `json:"description"`
	RenderedDescription string      `json:"rendered_description"`
	MarkdownDescription string
}

type group struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Fields []Function `json:"fields"`
}

type stdlib struct {
	Prefix string  `json:"prefix"`
	Groups []group `json:"groups"`
}

func Functions() ([]Function, error) {
	var lib stdlib

	vm := jsonnet.MakeVM()
	vm.Importer(&jsonnet.MemoryImporter{
		Data: map[string]jsonnet.Contents{
			"html.libsonnet": jsonnet.MakeContents(htmlLib),
		},
	})

	// Hack. Remove the examples, they use some new functions that may not be ready yet in the go lib
	stdLib = strings.ReplaceAll(stdLib, "examples:", "examples::")
	stdLib = strings.ReplaceAll(stdLib, "description:", "rendered_description: html.render(self.description), \ndescription:")

	jsonContent, err := vm.EvaluateAnonymousSnippet("", stdLib)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(jsonContent), &lib); err != nil {
		return nil, err
	}

	converter := md.NewConverter("", true, nil)
	allFunctions := []Function{}
	for _, group := range lib.Groups {
		for _, field := range group.Fields {
			field.MarkdownDescription, err = converter.ConvertString(field.RenderedDescription)
			if err != nil {
				return nil, err
			}
			allFunctions = append(allFunctions, field)
		}
	}

	return allFunctions, nil
}
