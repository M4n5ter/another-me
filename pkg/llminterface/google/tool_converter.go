package google

import (
	"log/slog"

	json "github.com/json-iterator/go"
	"google.golang.org/genai"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// ToolCoreSchemaToGoogleFunctionDeclaration 将 ToolCore 的 Schema 转换为 google genai 的 FunctionDeclaration
func ToolCoreSchemaToGoogleFunctionDeclaration(tcSchema *toolcore.ToolSchema) *genai.FunctionDeclaration {
	declaration := &genai.FunctionDeclaration{
		Name:        tcSchema.Name,
		Description: tcSchema.Descriptions[i18n.GlobalManager.GetDefaultLanguage()],
	}

	rawSchema := toolcore.ConvertParamsToRawSchema(tcSchema.InputParameters, tcSchema.Name, tcSchema.Descriptions[i18n.GlobalManager.GetDefaultLanguage()])
	rawSchemaJSON, err := json.Marshal(rawSchema)
	if err != nil {
		slog.Error("Failed to marshal raw schema", "error", err)
		return nil
	}

	functionParams := &genai.Schema{}
	err = functionParams.UnmarshalJSON(rawSchemaJSON)
	if err != nil {
		slog.Error("Failed to unmarshal raw schema", "error", err)
		return nil
	}

	declaration.Parameters = functionParams

	return declaration
}
