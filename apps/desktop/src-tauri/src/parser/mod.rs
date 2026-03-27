pub mod ollama;

use serde::Serialize;
use serde_json::Value;
use std::env;

const DEFAULT_PARSER_BACKEND: &str = "ollama";

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ParserResponse {
    pub command: Option<Value>,
    pub clarification: Option<String>,
    pub backend: String,
    pub model: String,
    pub confidence: f64,
}

pub fn active_backend() -> String {
    env::var("KAI_PARSER_BACKEND").unwrap_or_else(|_| DEFAULT_PARSER_BACKEND.to_string())
}

pub async fn parse(input: String, now: String) -> Result<ParserResponse, String> {
    match active_backend().as_str() {
        "ollama" => ollama::parse(input, now).await,
        other => Err(format!(
            "Unsupported parser backend '{}'. Set KAI_PARSER_BACKEND=ollama.",
            other
        )),
    }
}

pub async fn warm() -> Result<(), String> {
    match active_backend().as_str() {
        "ollama" => ollama::warm().await,
        other => Err(format!(
            "Unsupported parser backend '{}'. Set KAI_PARSER_BACKEND=ollama.",
            other
        )),
    }
}

