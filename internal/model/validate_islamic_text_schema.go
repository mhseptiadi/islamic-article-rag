package model

// GetValidateIslamicTextSchema returns the OpenAI-compatible tool schema for validate_islamic_text.
func GetValidateIslamicTextSchema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: ToolSchemaFunction{
			Name:        "validate_islamic_text",
			Description: "Fetches verified Arabic text and translation for Quran verses or Hadith. Call this before outputting any scripture to the user.",
			Parameters: ToolSchemaParameters{
				Type: "object",
				Properties: ToolSchemaParametersProps{
					References: ToolSchemaReferencesProperty{
						Type:        "array",
						Description: "One or more scripture references to validate.",
						Items: ToolSchemaReferenceItem{
							Type: "object",
							Properties: ToolSchemaReferenceItemProps{
								SourceType: ToolSchemaStringProperty{
									Type:        "string",
									Enum:        []string{"quran", "hadith"},
									Description: "The type of scripture to validate.",
								},
								ReferenceName: ToolSchemaStringProperty{
									Type:        "string",
									Description: "The name of the Surah (e.g., 'Al-Baqarah') or the Hadith collection (e.g., 'Bukhari').",
								},
								ReferenceNumber: ToolSchemaStringProperty{
									Type:        "string",
									Description: "The verse number or hadith number.",
								},
							},
							Required: []string{"source_type", "reference_name", "reference_number"},
						},
					},
				},
				Required: []string{"references"},
			},
		},
	}
}
