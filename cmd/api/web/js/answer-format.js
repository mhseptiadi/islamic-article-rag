const CITATION_PLACEHOLDER = "@@CIT@@";

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function parseTagAttributes(attrString) {
  const attrs = {};
  const pattern = /(\w+)=["']([^"']*)["']/g;
  let match = pattern.exec(attrString);
  while (match) {
    attrs[match[1].toLowerCase()] = match[2];
    match = pattern.exec(attrString);
  }
  return attrs;
}

function titleCase(value) {
  if (!value) {
    return "";
  }
  return value.charAt(0).toUpperCase() + value.slice(1);
}

function citationLabel(citation) {
  const { type, attrs } = citation;

  if (type === "quran") {
    const parts = [];
    if (attrs.chapter) {
      parts.push(`Surah ${attrs.chapter}`);
    }
    if (attrs.verse) {
      parts.push(`verse ${attrs.verse}`);
    }
    return parts.join(", ");
  }

  if (type === "hadith") {
    const parts = [];
    if (attrs.collection) {
      parts.push(titleCase(attrs.collection));
    }
    if (attrs.number) {
      parts.push(`#${attrs.number}`);
    }
    return parts.join(" ");
  }

  return "";
}

function renderCitation(citation) {
  const label = citationLabel(citation);
  const inner = citation.inner.trim();
  const quote = inner
    ? `<strong class="islamic-citation"><em>${escapeHTML(inner)}</em></strong>`
    : "";

  if (!label && !quote) {
    return "";
  }

  if (label && quote) {
    return `<span class="islamic-citation-block">${quote} <span class="islamic-citation-ref">(${escapeHTML(label)})</span></span>`;
  }

  if (label) {
    return `<span class="islamic-citation-block"><span class="islamic-citation-ref">${escapeHTML(label)}</span></span>`;
  }

  return quote;
}

function protectCitations(text) {
  const citations = [];

  const protectedText = text.replace(
    /<(quran|hadith)\b([^>]*)>([\s\S]*?)<\/\1>/gi,
    (match, tag, attrString, inner) => {
      const id = citations.length;
      citations.push({
        type: tag.toLowerCase(),
        attrs: parseTagAttributes(attrString),
        inner: inner.trim(),
      });
      return `${CITATION_PLACEHOLDER}${id}${CITATION_PLACEHOLDER}`;
    },
  );

  return { protectedText, citations };
}

function restoreCitations(html, citations) {
  const pattern = new RegExp(`${CITATION_PLACEHOLDER}(\\d+)${CITATION_PLACEHOLDER}`, "g");
  return html.replace(pattern, (_, id) => renderCitation(citations[Number(id)]));
}

function formatInline(text) {
  return text
    .replace(/\*\*\*([^*]+)\*\*\*/g, "<strong><em>$1</em></strong>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/(?<!\*)\*([^*]+)\*(?!\*)/g, "<em>$1</em>");
}

function formatMarkdown(text) {
  const lines = text.split("\n");
  const blocks = [];
  let listItems = [];
  let listOrdered = false;

  function flushList() {
    if (listItems.length === 0) {
      return;
    }
    const tag = listOrdered ? "ol" : "ul";
    const items = listItems.map((item) => `<li>${formatInline(item)}</li>`).join("");
    blocks.push(`<${tag}>${items}</${tag}>`);
    listItems = [];
    listOrdered = false;
  }

  for (const line of lines) {
    const trimmed = line.trim();
    const orderedMatch = /^(\d+)\.\s+(.+)$/.exec(trimmed);
    if (orderedMatch) {
      if (listItems.length > 0 && !listOrdered) {
        flushList();
      }
      listOrdered = true;
      listItems.push(orderedMatch[2]);
      continue;
    }

    const unorderedMatch = /^[-*]\s+(.+)$/.exec(trimmed);
    if (unorderedMatch) {
      if (listItems.length > 0 && listOrdered) {
        flushList();
      }
      listItems.push(unorderedMatch[1]);
      continue;
    }

    flushList();
    if (trimmed) {
      blocks.push(`<p>${formatInline(trimmed)}</p>`);
    }
  }

  flushList();
  return blocks.join("\n");
}

export function formatAnswerHTML(text) {
  const { protectedText, citations } = protectCitations(text);
  const escaped = escapeHTML(protectedText);
  const markdown = formatMarkdown(escaped);
  return restoreCitations(markdown, citations);
}
