package model

// ToolCall represents a tool invocation returned by an OpenAI-compatible LLM.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ToolSchema is the OpenAI-compatible tool definition for scripture validation.
// Works for Groq, OpenAI, and DeepInfra.
type ToolSchema struct {
	Type     string             `json:"type"`
	Function ToolSchemaFunction `json:"function"`
}

type ToolSchemaFunction struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Parameters  ToolSchemaParameters `json:"parameters"`
}

type ToolSchemaParameters struct {
	Type       string                     `json:"type"`
	Properties ToolSchemaParametersProps  `json:"properties"`
	Required   []string                   `json:"required"`
}

type ToolSchemaParametersProps struct {
	References ToolSchemaReferencesProperty `json:"references"`
}

type ToolSchemaReferencesProperty struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Items       ToolSchemaReferenceItem `json:"items"`
}

type ToolSchemaReferenceItem struct {
	Type       string                        `json:"type"`
	Properties ToolSchemaReferenceItemProps  `json:"properties"`
	Required   []string                      `json:"required"`
}

type ToolSchemaReferenceItemProps struct {
	SourceType      ToolSchemaStringProperty `json:"source_type"`
	ReferenceName   ToolSchemaStringProperty `json:"reference_name"`
	ReferenceNumber ToolSchemaStringProperty `json:"reference_number"`
}

type ToolSchemaStringProperty struct {
	Type        string   `json:"type"`
	Enum        []string `json:"enum,omitempty"`
	Description string   `json:"description,omitempty"`
}

// ScriptureReference is the argument shape the LLM passes to validate_islamic_text.
type ScriptureReference struct {
	SourceType      string `json:"source_type"`
	ReferenceName   string `json:"reference_name"`
	ReferenceNumber string `json:"reference_number"`
}

// ToolArguments is the top-level arguments object for validate_islamic_text.
type ToolArguments struct {
	References []ScriptureReference `json:"references"`
}
