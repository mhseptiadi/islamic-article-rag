package service

import (
	"fmt"
	"strings"
)

func buildRAGPrompt(question string, contextBlocks []string) string {
	var b strings.Builder

	b.WriteString(ragSystemPrompt(false))
	b.WriteString("\n\nExamples:\n\n")
	for _, msg := range ragFewShotExamples() {
		switch msg.Role {
		case "user":
			b.WriteString("Question: ")
		case "assistant":
			b.WriteString("Answer: ")
		}
		b.WriteString(msg.Content)
		b.WriteString("\n\n")
	}

	if len(contextBlocks) == 0 {
		b.WriteString("Articles:\n(no relevant articles found)\n\n")
	} else {
		b.WriteString("Articles:\n")
		for i, block := range contextBlocks {
			b.WriteString(fmt.Sprintf("[%d] %s\n\n", i+1, block))
		}
	}

	b.WriteString("Question: ")
	b.WriteString(question)
	b.WriteString("Answer:")

	return b.String()
}

func buildRAGMessages(question string, contextBlocks []string, needTools bool) []map[string]interface{} {
	messages := make([]map[string]interface{}, 0, len(ragFewShotExamples())+2)
	messages = append(messages, map[string]interface{}{
		"role":    "system",
		"content": ragSystemPrompt(needTools),
	})
	for _, msg := range ragFewShotExamples() {
		messages = append(messages, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	var userContent strings.Builder
	if len(contextBlocks) == 0 {
		userContent.WriteString("Articles:\n(no relevant articles found)\n\n")
	} else {
		userContent.WriteString("Articles:\n")
		for i, block := range contextBlocks {
			userContent.WriteString(fmt.Sprintf("[%d] %s\n\n", i+1, block))
		}
	}
	userContent.WriteString("Question: ")
	userContent.WriteString(question)

	messages = append(messages, map[string]interface{}{
		"role":    "user",
		"content": userContent.String(),
	})

	return messages
}

func ragSystemPromptOld() string {
	return `You are an Islamic AI assistant.
	RULE 1: Answer in the same language as the question.
	RULE 2: If Question is not related to Islamic topic, answer with "` + OffTopicAnswer + `"
	RULE 3: Every time you quote the Quran, you must use this exact format: <quran chapter="number" verse="number">quote text</quran>.
	RULE 4: Every time you quote a Hadith, you must use this exact format: <hadith collection="name" number="number">quote text</hadith>.
	RULE 5: Do not write citations outside of these tags.
	RULE 6: ADJUST THE FORMAT & LENGTH OF THE ANSWER ACCORDING TO THE QUESTION TYPE:
	- If the question asks about "Rulings", "What", or "Why": Answer concisely in a dense response (around 100 - 200 words).
	- If the question asks about "Procedures", "Guides", or "How": Explain completely but remain focused (maximum 400 words).
	`
}

func ragSystemPrompt(needTools bool) string {
	if needTools {
		// return `You are an Islamic AI assistant.
		// RULE 1: Answer in the same language as the question.

		// RULE 2: OUT-OF-DOMAIN & GROUNDING GUARDRAIL: If the question is not related to Islamic topics, OR if the information cannot be found in the provided articles, do not attempt to guess. You MUST answer exactly with: "` + OffTopicAnswer + `"

		// RULE 3: Detect if the user's question REQUIRES verifying specific Quranic verses or Hadiths using the 'validate_islamic_text' tool.
		// CRITICAL HEURISTICS CRITERIAS FOR USING THE TOOL:
		// 1. EXPLICIT REQUESTS (YES): If the user explicitly asks for a proof, verse, or hadith (e.g., "apa dalilnya", "surat apa", "sebutkan haditsnya"), you MUST use the tool.
		// 2. JURISPRUDENCE / FIQH (YES): If the question is about strict Islamic rulings, halal/haram, or obligations (e.g., "apa hukumnya", "syarat sah"), you SHOULD use the tool IF the provided articles contain a specific reference.
		// 3. HISTORY / DEFINITIONS (NO): If the question is about history (Sirah/Sejarah), names of figures, basic definitions, or general advice (e.g., "siapa nama nabi", "apa arti tarawih"), do NOT use the tool.
		// 4. LACK OF REFERENCES (NO): If the provided articles do not explicitly mention a Surah name and Verse number, or a Hadith collector and number, do NOT use the tool. Do not guess references.

		// If the tool is needed based on these criteria, trigger the JSON tool call and ignore next rule.

		// RULE 4: ADJUST FORMAT & LENGTH:
		// - Rulings/What/Why: Concise paragraph (100 - 200 words).
		// - Procedures/Guides/How: Step-by-step list (max 400 words).`
		return `You are an Islamic AI assistant.
RULE 1: Answer in the same language as the question.

RULE 2: OUT-OF-DOMAIN & GROUNDING GUARDRAIL: If the question is not related to Islamic topics, OR if the information cannot be found in the provided articles, do not attempt to guess. You MUST answer exactly with: "` + OffTopicAnswer + `"

RULE 3: SCRIPTURE VALIDATION: Before quoting a Quran verse or Hadith, you MUST call the 'validate_islamic_text' tool. Once you receive the "RESULTS FOR YOUR REQUESTED BATCH" in the chat history, you are permitted to output the verified text wrapped in <quran> or <hadith> tags.

RULE 4: ADJUST FORMAT & LENGTH:
- Rulings/What/Why: Concise paragraph (100 - 200 words).
- Procedures/Guides/How: Step-by-step list (max 400 words).`
	} else {
		return `You are an Islamic AI assistant.
		RULE 1: Answer in the same language as the question.
		RULE 2: OUT-OF-DOMAIN & GROUNDING GUARDRAIL: If the question is not related to Islamic topics, OR if the information cannot be found in the provided articles, do not attempt to guess. You MUST answer exactly with: "` + OffTopicAnswer + `"
		RULE 3: ADJUST FORMAT & LENGTH:
		- Rulings/What/Why: Concise paragraph (100 - 200 words).
		- Procedures/Guides/How: Step-by-step list (max 400 words).`
	}
}

func ragFewShotExamples() []Message {
	return []Message{
		// {Role: "user", Content: "What does the Quran say about fasting?"},
		// {Role: "assistant", Content: "Fasting is prescribed for believers. Allah says: <quran chapter=\"2\" verse=\"183\">O you who have believed, decreed upon you is fasting as it was decreed upon those before you that you may become righteous.</quran>"},
		// {Role: "user", Content: "Give me a hadith about intention."},
		// {Role: "assistant", Content: "The Prophet emphasized intention deeply. He said: <hadith collection=\"bukhari\" number=\"1\">The reward of deeds depends upon the intentions.</hadith>"},
	}
}
