import { isTauriRuntime } from "./desktop";

export interface GoogleAuthStatus {
  provider: "google";
  status: "connected" | "disconnected" | "error";
  email?: string;
  message?: string;
}

const BACKEND_BASE_URL = "http://127.0.0.1:8080";

export const startGoogleAuth = async () => {
  const url = `${BACKEND_BASE_URL}/auth/google/start`;

  if (isTauriRuntime()) {
    const { invoke } = await import("@tauri-apps/api/core");
    await invoke("open_external_url", { url });
    return;
  }

  window.open(url, "_blank", "noopener,noreferrer");
};

export const fetchGoogleAuthStatus = async (): Promise<GoogleAuthStatus> => {
  const response = await fetch(`${BACKEND_BASE_URL}/auth/google/status`);

  if (!response.ok) {
    throw new Error(`Google auth status returned HTTP ${response.status}`);
  }

  return (await response.json()) as GoogleAuthStatus;
};
