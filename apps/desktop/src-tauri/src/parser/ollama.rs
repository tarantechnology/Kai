use reqwest::Client;
use serde::Deserialize;
use serde_json::json;
use std::env;

use super::ParserResponse;

const OLLAMA_API_URL: &str = "http://127.0.0.1:11434/api/chat";
const DEFAULT_OLLAMA_MODEL: &str = "llama3.2:3b";
const KEEP_ALIVE: &str = "30m";

#[derive(Debug, Deserialize)]
struct OllamaChatResponse {
    model: String,
    message: OllamaMessage,
}

#[derive(Debug, Deserialize)]
struct OllamaMessage {
    content: String,
}

#[derive(Debug, Deserialize)]
struct ModelPayload {
    command: Option<serde_json::Value>,
    clarification: Option<String>,
    confidence: Option<f64>,
}

fn parser_model() -> String {
    env::var("OLLAMA_MODEL").unwrap_or_else(|_| DEFAULT_OLLAMA_MODEL.to_string())
}

fn command_schema() -> serde_json::Value {
    // ollama is asked for strict json so the frontend can validate a narrow command shape.
    json!({
        "type": "object",
        "properties": {
            "command": {
                "anyOf": [
                    { "type": "null" },
                    {
                        "type": "object",
                        "properties": {
                            "type": {
                                "type": "string",
                                "enum": [
                                    "create_reminder",
                                    "create_event",
                                    "show_tasks",
                                    "show_calendar",
                                    "sync_canvas",
                                    "sync_google_calendar"
                                ]
                            },
                            "title": { "type": "string" },
                            "datetimeLabel": { "type": "string" },
                            "startLabel": { "type": "string" },
                            "endLabel": { "type": "string" },
                            "range": {
                                "type": "string",
                                "enum": ["today", "tomorrow", "week", "all", "day"]
                            },
                            "confidence": { "type": "number" }
                        },
                        "required": ["type", "confidence"],
                        "additionalProperties": false
                    }
                ]
            },
            "clarification": {
                "anyOf": [{ "type": "string" }, { "type": "null" }]
            },
            "confidence": { "type": "number" }
        },
        "required": ["command", "clarification", "confidence"],
        "additionalProperties": false
    })
}

fn system_prompt(now: &str) -> String {
    // this prompt teaches the local model to do intent mapping, not direct app control.
    format!(
        "You are Kai's local command interpreter. Current local date and time: {now}. \
Interpret the user's raw natural-language request into exactly one supported command or return null if unsupported or genuinely ambiguous. \
Supported commands are: create_reminder, create_event, show_tasks, show_calendar, sync_canvas, sync_google_calendar. \
Your job is semantic intent classification and slot extraction, not phrase matching. \
Infer the user's meaning from ordinary language even when phrased casually, indirectly, or with uncommon wording. \
Map the request to the closest supported action based on meaning. \
Use short human-readable time labels like '7:00 PM'. \
Allowed ranges: show_tasks => today, tomorrow, week. show_calendar => day, tomorrow, week. sync_canvas => today, week, all. sync_google_calendar => today, week, all. \
Decision rules: \
- Requests about seeing what the user has to do, what is left, reminders, assignments, tasks, obligations, or workload should generally map to show_tasks. \
- Requests about seeing the user's schedule, calendar, agenda, meetings, time blocks, or what is on the calendar should generally map to show_calendar. \
- Requests to remember something at a time should map to create_reminder. \
- Requests to block time, schedule focused time, add a meeting, or create a calendar block should map to create_event. \
- Requests to import or sync Canvas coursework should map to sync_canvas. \
- Requests to sync the calendar provider should map to sync_google_calendar. \
Defaults: \
- For read actions, if the user does not specify a range and intent is otherwise clear, choose the most sensible default rather than asking for clarification. \
- Default show_tasks to today. \
- Default show_calendar to day. \
- Default sync_canvas to week. \
- Default sync_google_calendar to week. \
Clarification policy: \
- Ask for clarification only when the intended action is unclear or a write action is missing essential information that cannot be reasonably inferred. \
- Do not ask for clarification merely because the wording is conversational or the time range was omitted for a read action. \
Output policy: \
- Return valid JSON matching the supplied schema. \
- Set command to null only when you truly cannot determine a supported action with reasonable confidence. \
- Confidence should reflect how well the user's intent maps to the supported action set, not whether the wording exactly matches prior examples."
    )
}

pub async fn parse(input: String, now: String) -> Result<ParserResponse, String> {
    let client = Client::new();
    let model = parser_model();

    // the native side owns parser calls so the backend can change without rewriting react code.
    let response = client
        .post(OLLAMA_API_URL)
        .json(&json!({
            "model": model,
            "messages": [
                {
                    "role": "system",
                    "content": system_prompt(&now)
                },
                {
                    "role": "user",
                    "content": input
                }
            ],
            "stream": false,
            "keep_alive": KEEP_ALIVE,
            "options": {
                "temperature": 0
            },
            "format": command_schema()
        }))
        .send()
        .await
        .map_err(|error| format!("Failed to reach Ollama at http://127.0.0.1:11434: {error}"))?;

    if !response.status().is_success() {
        return Err(format!(
            "Ollama returned HTTP {}. Make sure model '{}' is installed and the Ollama app is running.",
            response.status(),
            model
        ));
    }

    let body: OllamaChatResponse = response
        .json()
        .await
        .map_err(|error| format!("Failed to decode Ollama response: {error}"))?;

    let parsed: ModelPayload = serde_json::from_str(&body.message.content)
        .map_err(|error| format!("Ollama returned invalid JSON for Kai to validate: {error}"))?;

    Ok(ParserResponse {
        command: parsed.command,
        clarification: parsed.clarification,
        backend: "ollama".to_string(),
        model: body.model,
        confidence: parsed.confidence.unwrap_or(0.0),
    })
}

pub async fn warm() -> Result<(), String> {
    let client = Client::new();
    let model = parser_model();

    // this lightweight request keeps the model warm so the first real command feels faster.
    let response = client
        .post(OLLAMA_API_URL)
        .json(&json!({
            "model": model,
            "messages": [
                {
                    "role": "system",
                    "content": "Return valid JSON."
                },
                {
                    "role": "user",
                    "content": "{\"ready\": true}"
                }
            ],
            "stream": false,
            "keep_alive": KEEP_ALIVE,
            "options": {
                "temperature": 0
            },
            "format": {
                "type": "object",
                "properties": {
                    "ready": { "type": "boolean" }
                },
                "required": ["ready"],
                "additionalProperties": false
            }
        }))
        .send()
        .await
        .map_err(|error| format!("Failed to warm Ollama model '{}': {error}", model))?;

    if !response.status().is_success() {
        return Err(format!(
            "Ollama warm-up returned HTTP {} for model '{}'.",
            response.status(),
            model
        ));
    }

    Ok(())
}
