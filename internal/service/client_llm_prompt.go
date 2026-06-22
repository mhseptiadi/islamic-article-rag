package service

import (
	"fmt"
	"strings"
)

func buildRAGPrompt(question string, contextBlocks []string) string {
	var b strings.Builder

	b.WriteString(ragSystemPrompt())
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

func buildRAGMessages(question string, contextBlocks []string) []map[string]string {
	messages := make([]map[string]string, 0, len(ragFewShotExamples())+2)
	messages = append(messages, map[string]string{
		"role":    "system",
		"content": ragSystemPrompt(),
	})
	for _, msg := range ragFewShotExamples() {
		messages = append(messages, map[string]string{
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

	messages = append(messages, map[string]string{
		"role":    "user",
		"content": userContent.String(),
	})

	return messages
}

func ragSystemPrompt() string {
	return `You are an Islamic AI assistant.
Rule 1: Answer in the same language as the question.
Rule 2: Every time you quote the Quran, you must use this exact format: <quran chapter="number" verse="number">quote text</quran>.
Rule 3: Every time you quote a Hadith, you must use this exact format: <hadith collection="name" number="number">quote text</hadith>.
Rule 4: Do not write citations outside of these tags.`
}

func ragFewShotExamples() []Message {
	return []Message{
		{Role: "user", Content: "What does the Quran say about fasting?"},
		{Role: "assistant", Content: "Fasting is prescribed for believers. Allah says: <quran chapter=\"2\" verse=\"183\">O you who have believed, decreed upon you is fasting as it was decreed upon those before you that you may become righteous.</quran>"},
		{Role: "user", Content: "Give me a hadith about intention."},
		{Role: "assistant", Content: "The Prophet emphasized intention deeply. He said: <hadith collection=\"bukhari\" number=\"1\">The reward of deeds depends upon the intentions.</hadith>"},
	}
}
