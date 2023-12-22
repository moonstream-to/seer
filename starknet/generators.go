package starknet

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

// Common parameters required for the generation of all types of artifacts.
type GenerationParameters struct {
	OriginalName string
	GoName       string
}

// The output of the code generation process for enum items in a Starknet ABI.
type GeneratedEnum struct {
	GenerationParameters
	ParseFunctionName string
	Definition        *Enum
	Code              string
}

// The output of the code generation process for struct items in a Starknet ABI.
type GeneratedStruct struct {
	GenerationParameters
	Definition   *Struct
	ParamsLength int
	Code         string
}

// Generates a Go name for a Starknet ABI item given its fully qualified ABI name.
// Qualified names for Starknet ABI items are of the form:
// `core::starknet::contract_address::ContractAddress`
func GenerateGoNameForType(qualifiedName string) string {
	if qualifiedName == "core::integer::u8" || qualifiedName == "core::integer::u16" || qualifiedName == "core::integer::u32" || qualifiedName == "core::integer::u64" {
		return "uint64"
	} else if strings.HasPrefix(qualifiedName, "core::integer::") {
		return "lol"
	} else if qualifiedName == "core::starknet::contract_address::ContractAddress" {
		return "string"
	} else if qualifiedName == "core::felt252" {
		return "string"
	} else if strings.HasPrefix(qualifiedName, "core::array::Array::<") {
		s1, _ := strings.CutPrefix(qualifiedName, "core::array::Array::<")
		s2, _ := strings.CutSuffix(s1, ">")
		return fmt.Sprintf("[]%s", GenerateGoNameForType(s2))
	}
	return strcase.ToCamel(strings.Replace(qualifiedName, "::", "_", -1))
}

// Generate generates Go code for each of the items in a Starknet contract ABI.
// Returns a mapping of the go name of each object to a specification of the generated artifact.
// Currently supports:
// - Enums
// - Structs
// - Events
//
// ABI names are used to depuplicate code snippets. The assumption is that the Starknet fully
// qualified name for a type uniquely determines that type across the entire ABI. This way
// even if the ABI passed into the code generator contains duplicate instances of an ABI item,
// the Go code will only contain one definition of that item.
func GenerateSnippets(parsed *ParsedABI) (map[string]string, error) {
	result := map[string]string{}

	enumTemplate, enumTemplateParseErr := template.New("enum").Parse(EnumTemplate)
	if enumTemplateParseErr != nil {
		return result, enumTemplateParseErr
	}

	structTemplateFuncs := map[string]any{
		"CamelCase":             strcase.ToCamel,
		"GenerateGoNameForType": GenerateGoNameForType,
	}

	structTemplate, structTemplateParseErr := template.New("struct").Funcs(structTemplateFuncs).Parse(StructTemplate)
	if structTemplateParseErr != nil {
		return result, structTemplateParseErr
	}

	for _, enum := range parsed.Enums {
		goName := GenerateGoNameForType(enum.Name)
		parseFunctionName := fmt.Sprintf("Parse%s", goName)

		generated := GeneratedEnum{
			GenerationParameters: GenerationParameters{
				OriginalName: enum.Name,
				GoName:       goName,
			},
			ParseFunctionName: parseFunctionName,
			Definition:        enum,
			Code:              "",
		}

		var b bytes.Buffer
		templateErr := enumTemplate.Execute(&b, generated)
		if templateErr != nil {
			return result, templateErr
		}

		generated.Code = b.String()

		result[enum.Name] = generated.Code
	}

	for _, structItem := range parsed.Structs {
		goName := GenerateGoNameForType(structItem.Name)

		generated := GeneratedStruct{
			GenerationParameters: GenerationParameters{
				OriginalName: structItem.Name,
				GoName:       goName,
			},
			Definition:   structItem,
			ParamsLength: len(structItem.Members),
			Code:         "",
		}

		var b bytes.Buffer
		templateErr := structTemplate.Execute(&b, generated)
		if templateErr != nil {
			return result, templateErr
		}

		generated.Code = b.String()

		result[structItem.Name] = generated.Code
	}

	return result, nil
}

// Generates a single string consisting of the Go code for all the artifacts in a parsed Starknet ABI.
func Generate(parsed *ParsedABI) (string, error) {
	snippets, snippetsErr := GenerateSnippets(parsed)
	if snippetsErr != nil {
		return "", snippetsErr
	}

	sections := make([]string, len(snippets))
	currentSection := 0
	for _, section := range snippets {
		sections[currentSection] = section
		currentSection++
	}

	return strings.Join(sections, "\n\n"), nil
}

// This is the Go template which is used to generate the function corresponding to an Enum.
// This template should be applied to a GeneratedEnum struct.
var EnumTemplate string = `// {{.GoName}} is an alias for string
type {{.GoName}} = string

// {{.OriginalName}}
// This function maps a Felt corresponding to the index of an enum variant to the name of that variant.
func {{.ParseFunctionName}}(parameter *felt.Felt) {{.GoName}} {
	parameterInt := parameter.Uint64()
	switch parameterInt {
	{{range .Definition.Variants}}case {{.Index}}:
		return "{{.Name}}"
	{{end}}
	}
	return "UNKNOWN"
}`

// This is the Go template which is used to generate the struct.
// This template should be applied to a GeneratedStruct struct.
var StructTemplate string = `// {{.OriginalName}}
// {{.GoName}} is the Go struct corresponding to the {{.OriginalName}} struct.
type {{.GoName}} struct {
	{{range .Definition.Members}}
	{{(CamelCase .Name)}} {{(GenerateGoNameForType .Type)}}
	{{- end}}
}
`
