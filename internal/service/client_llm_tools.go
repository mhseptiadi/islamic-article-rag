package service

// GetValidatorToolSchema returns the standard OpenAI-compatible tool schema.
// Works for Groq, OpenAI, and DeepInfra.
func GetValidatorToolSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "validate_islamic_text",
			"description": "Fetches verified Arabic text and translation for Quran verses or Hadith. Call this before outputting any scripture to the user.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"references": map[string]interface{}{
						"type":        "array",
						"description": "One or more scripture references to validate.",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"source_type": map[string]interface{}{
									"type":        "string",
									"enum":        []string{"quran", "hadith"},
									"description": "The type of scripture to validate.",
								},
								"reference_name": map[string]interface{}{
									"type":        "string",
									"description": "The name of the Surah (e.g., 'Al-Baqarah') or the Hadith collection (e.g., 'Bukhari').",
								},
								"reference_number": map[string]interface{}{
									"type":        "string",
									"description": "The verse number or hadith number.",
								},
							},
							"required": []string{"source_type", "reference_name", "reference_number"},
						},
					},
				},
				"required": []string{"references"},
			},
		},
	}
}
