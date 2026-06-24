import { formatAnswerHTML } from "./answer-format.js";

const form = document.getElementById("ask-form");
const questionInput = document.getElementById("question");
const charCount = document.getElementById("char-count");
const submitBtn = document.getElementById("submit-btn");
const btnLabel = submitBtn.querySelector(".btn-label");
const btnSpinner = submitBtn.querySelector(".btn-spinner");
const statusEl = document.getElementById("status");
const answerSection = document.getElementById("answer-section");
const answerDisclaimer = document.getElementById("answer-disclaimer");
const answerEl = document.getElementById("answer");
const feedbackSection = document.getElementById("feedback-section");
const feedbackForm = document.getElementById("feedback-form");
const feedbackTypesEl = document.getElementById("feedback-types");
const feedbackComment = document.getElementById("feedback-comment");
const feedbackCharCount = document.getElementById("feedback-char-count");
const feedbackSubmitBtn = document.getElementById("feedback-submit-btn");
const feedbackBtnLabel = feedbackSubmitBtn.querySelector(".btn-label");
const feedbackBtnSpinner = feedbackSubmitBtn.querySelector(".btn-spinner");
const feedbackSuccessEl = document.getElementById("feedback-success");
const sourcesSection = document.getElementById("sources-section");
const sourcesEl = document.getElementById("sources");
const MAX_CHARS = Number(questionInput.getAttribute("maxlength")) || 300;
const MAX_FEEDBACK_CHARS = Number(feedbackComment.getAttribute("maxlength")) || 1000;

const FEEDBACK_TYPES = [
  { value: "good", label: "Answer is good." },
  { value: "hallucinated", label: "The answer is hallucinated (fake info not in the text)." },
  { value: "wrong_context", label: "The answer is true, but it doesn't match the user's actual question." },
  { value: "wrong_citation", label: "Wrong Quran or Hadith recitation/translation." },
  { value: "incomplete", label: "The answer is incomplete." },
  { value: "false_refusal", label: "The LLM refused to answer the question." },
  { value: "other", label: "Something else." },
];

let currentRecordID = null;

function updateCharCount() {
  const length = questionInput.value.length;
  charCount.textContent = `${length} / ${MAX_CHARS}`;
  charCount.classList.toggle("near-limit", length >= MAX_CHARS * 0.85 && length < MAX_CHARS);
  charCount.classList.toggle("at-limit", length >= MAX_CHARS);
}

function setLoading(isLoading) {
  submitBtn.disabled = isLoading;
  questionInput.disabled = isLoading;
  btnSpinner.hidden = !isLoading;
  btnLabel.textContent = isLoading ? "Processing..." : "Submit question";
}

function showStatus(message, type = "error") {
  statusEl.hidden = false;
  statusEl.textContent = message;
  statusEl.className = `status ${type}`;
}

function clearStatus() {
  statusEl.hidden = true;
  statusEl.textContent = "";
  statusEl.className = "status";
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function renderAnswer(text, offTopic = false) {
  if (!text) {
    answerSection.hidden = true;
    answerEl.innerHTML = "";
    answerDisclaimer.hidden = false;
    return;
  }

  answerSection.hidden = false;
  answerDisclaimer.hidden = offTopic;
  answerEl.innerHTML = formatAnswerHTML(text);
}

function renderSources(data) {
  const items = [];

  for (const chunk of data.chunks || []) {
    const meta = chunk.payload?.metadata || {};
    const title = meta.title || "Article source";
    const url = meta.source_url;
    const score = typeof chunk.score === "number" ? chunk.score.toFixed(3) : null;
    const excerpt = chunk.payload?.text || "";

    items.push({ title, url, score, excerpt, kind: "chunk" });
  }

  for (const article of data.full_articles || []) {
    items.push({
      title: "Full article",
      url: article.url,
      score: null,
      excerpt: article.text,
      kind: "article",
    });
  }

  if (items.length === 0) {
    sourcesSection.hidden = true;
    sourcesEl.innerHTML = "";
    return;
  }

  sourcesSection.hidden = false;
  sourcesEl.innerHTML = items
    .map((item) => {
      const scoreLabel = item.score ? `<span class="source-meta">Score: ${item.score}</span>` : "";
      const link = item.url
        ? `<a class="source-link" href="${escapeHTML(item.url)}" target="_blank" rel="noopener noreferrer">${escapeHTML(item.url)}</a>`
        : "";
      const excerpt = item.excerpt
        ? `<p class="source-excerpt">${escapeHTML(truncate(item.excerpt, 320))}</p>`
        : "";

      return `
        <article class="source-item">
          <div class="source-header">
            <h3 class="source-title">${escapeHTML(item.title)}</h3>
            ${scoreLabel}
          </div>
          ${link}
          ${excerpt}
        </article>
      `;
    })
    .join("");
}

function truncate(text, maxLength) {
  if (text.length <= maxLength) {
    return text;
  }
  return `${text.slice(0, maxLength).trimEnd()}...`;
}

function renderFeedbackTypes() {
  feedbackTypesEl.innerHTML = FEEDBACK_TYPES.map((type) => `
    <label class="feedback-option">
      <input type="radio" name="feedback_type" value="${type.value}" required>
      <span class="feedback-option-text">${escapeHTML(type.label)}</span>
    </label>
  `).join("");
}

function resetFeedbackForm() {
  currentRecordID = null;
  feedbackSection.hidden = true;
  feedbackSection.classList.remove("is-submitted");
  feedbackForm.hidden = false;
  feedbackForm.reset();
  feedbackSuccessEl.hidden = true;
  setFeedbackLoading(false);
  updateFeedbackCharCount();
}

function showFeedbackForm(recordID) {
  currentRecordID = recordID;
  feedbackSection.hidden = false;
  feedbackSection.classList.remove("is-submitted");
  feedbackForm.hidden = false;
  feedbackSuccessEl.hidden = true;
  feedbackForm.reset();
  setFeedbackLoading(false);
  updateFeedbackCharCount();
}

function setFeedbackLoading(isLoading) {
  feedbackSubmitBtn.disabled = isLoading;
  feedbackComment.disabled = isLoading;
  feedbackTypesEl.querySelectorAll("input").forEach((input) => {
    input.disabled = isLoading;
  });
  feedbackBtnSpinner.hidden = !isLoading;
  feedbackBtnLabel.textContent = isLoading ? "Submitting..." : "Submit feedback";
}

function updateFeedbackCharCount() {
  const length = feedbackComment.value.length;
  feedbackCharCount.textContent = `${length} / ${MAX_FEEDBACK_CHARS}`;
  feedbackCharCount.classList.toggle("near-limit", length >= MAX_FEEDBACK_CHARS * 0.85 && length < MAX_FEEDBACK_CHARS);
  feedbackCharCount.classList.toggle("at-limit", length >= MAX_FEEDBACK_CHARS);
}

async function submitFeedback(recordID, feedbackType, comment) {
  const response = await fetch("/api/v1/feedback", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify({
      record_id: recordID,
      feedback_type: feedbackType,
      comment,
    }),
  });

  const contentType = response.headers.get("content-type") || "";
  const isJSON = contentType.includes("application/json");
  const payload = isJSON ? await response.json() : await response.text();

  if (!response.ok) {
    const message = typeof payload === "string" && payload.trim()
      ? payload.trim()
      : payload?.message || `Request failed (${response.status})`;
    throw new Error(message);
  }

  return payload;
}

async function askQuestion(question) {
  const response = await fetch("/api/v1/ask", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify({ question }),
  });

  const contentType = response.headers.get("content-type") || "";
  const isJSON = contentType.includes("application/json");
  const payload = isJSON ? await response.json() : await response.text();

  if (!response.ok) {
    const message = typeof payload === "string" && payload.trim()
      ? payload.trim()
      : payload?.message || `Request failed (${response.status})`;
    throw new Error(message);
  }

  return payload;
}

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  clearStatus();

  const question = questionInput.value.trim();
  if (!question) {
    showStatus("Please enter a question first.");
    questionInput.focus();
    return;
  }

  setLoading(true);
  answerSection.hidden = true;
  sourcesSection.hidden = true;
  resetFeedbackForm();

  try {
    const data = await askQuestion(question);
    renderAnswer(data.answer || "", data.off_topic);
    renderSources(data);
    if (data.record_id && !data.off_topic) {
      showFeedbackForm(data.record_id);
    }
  } catch (error) {
    renderAnswer("");
    renderSources({ chunks: [], full_articles: [] });
    resetFeedbackForm();
    showStatus(error.message || "An error occurred while processing your question.");
  } finally {
    setLoading(false);
  }
});

feedbackForm.addEventListener("submit", async (event) => {
  event.preventDefault();

  if (!currentRecordID) {
    showStatus("There is no answer to give feedback on.");
    return;
  }

  const selectedType = feedbackForm.querySelector('input[name="feedback_type"]:checked');
  if (!selectedType) {
    showStatus("Please select a feedback type first.");
    return;
  }

  setFeedbackLoading(true);

  try {
    await submitFeedback(currentRecordID, selectedType.value, feedbackComment.value.trim());
    feedbackSection.classList.add("is-submitted");
    feedbackSuccessEl.hidden = false;
    clearStatus();
  } catch (error) {
    showStatus(error.message || "An error occurred while submitting feedback.");
  } finally {
    setFeedbackLoading(false);
  }
});

questionInput.addEventListener("input", updateCharCount);
feedbackComment.addEventListener("input", updateFeedbackCharCount);
renderFeedbackTypes();
updateCharCount();
updateFeedbackCharCount();
