package stdlib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFunctions(t *testing.T) {
	functions, err := Functions()
	assert.NoError(t, err)

	// Check std.min
	minFunc := Function{
		Name:   "min",
		Params: []string{"a", "b"},
	}
	contains(t, functions, minFunc)

	// Check std.ceil
	ceilFunc := Function{
		Name:   "ceil",
		Params: []string{"x"},
	}
	contains(t, functions, ceilFunc)

	// Check std.isNumber
	isNumberFunc := Function{
		Name:   "isNumber",
		Params: []string{"v"},
	}
	contains(t, functions, isNumberFunc)

	// Check std.clamp
	clampFunc := Function{
		Name:                "clamp",
		AvailableSince:      "0.15.0",
		Params:              []string{"x", "minVal", "maxVal"},
		MarkdownDescription: "Clamp a value to fit within the range \\[ `minVal`, `maxVal`\\].\nEquivalent to `std.max(minVal, std.min(x, maxVal))`.",
	}
	contains(t, functions, clampFunc)

	// Check std.manifestYamlDoc
	yamlFunc := Function{
		Name:                "manifestYamlDoc",
		Params:              []string{"value", "indent_array_in_object=false", "quote_keys=true"},
		MarkdownDescription: "Convert the given value to a YAML form. Note that `std.manifestJson` could also\nbe used for this purpose, because any JSON is also valid YAML. But this function will\nproduce more canonical-looking YAML.\n\n```\nstd.manifestYamlDoc(\n  {\n      x: [1, 2, 3, true, false, null,\n          \"string\\nstring\\n\"],\n      y: { a: 1, b: 2, c: [1, 2] },\n  },\n  indent_array_in_object=false)\n```\n\nYields a string containing this YAML:\n\n```\n\"x\":\n  - 1\n  - 2\n  - 3\n  - true\n  - false\n  - null\n  - |\n      string\n      string\n\"y\":\n  \"a\": 1\n  \"b\": 2\n  \"c\":\n      - 1\n      - 2\n```\n\nThe `indent_array_in_object` param adds additional indentation which some people\nmay find easier to read.\n\nThe `quote_keys` parameter controls whether YAML identifiers are always quoted\nor only when necessary.",
	}
	contains(t, functions, yamlFunc)
}

func contains(t *testing.T, funcs []Function, expected Function) {
	for _, f := range funcs {
		if f.Name != expected.Name {
			continue
		}
		// We don't care about these. Only the markdown
		f.Description = nil
		f.RenderedDescription = ""
		assert.Equal(t, expected, f)
		return
	}
	t.Errorf("Did not find function %s", expected.Name)
}
