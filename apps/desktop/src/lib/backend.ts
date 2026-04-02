import { isTauriRuntime } from "./desktop";

export interface GoogleAuthStatus {
  provider: "google";
  status: "connected" | "disconnected" | "error";
  email?: string;
  name?: string;
  message?: string;
}

export interface KaiAuthStatus {
  provider: "kai" | "google" | "email";
  status: "connected" | "disconnected" | "error";
  email?: string;
  name?: string;
  message?: string;
}

export interface EmailAuthPayload {
  email: string;
  password: string;
  name?: string;
}

export interface AuthCallbackPayload {
  provider?: string;
  flow_state?: string;
  access_token?: string;
  refresh_token?: string;
  provider_token?: string;
  provider_refresh_token?: string;
  token_type?: string;
  expires_in?: number;
  type?: string;
  intent?: string;
  status?: string;
  message?: string;
  error?: string;
  error_description?: string;
}

const BACKEND_BASE_URL = "http://127.0.0.1:8080";

export const startGoogleAuth = async (intent: "login" | "connect" = "login") => {
  const url = `${BACKEND_BASE_URL}/auth/google/start?intent=${intent}`;

  if (isTauriRuntime()) {
    const { invoke } = await import("@tauri-apps/api/core");
    await invoke("open_external_url", { url });
    return;
  }

  window.open(url, "_blank", "noopener,noreferrer");
};

export const startGoogleConnect = async () => {
  const url = `${BACKEND_BASE_URL}/auth/google/connect/start`;

  if (isTauriRuntime()) {
    const { invoke } = await import("@tauri-apps/api/core");
    await invoke("open_external_url", { url });
    return;
  }

  window.open(url, "_blank", "noopener,noreferrer");
};

export const fetchGoogleAuthStatus = async (): Promise<GoogleAuthStatus> => {
  const response = await fetch(`${BACKEND_BASE_URL}/auth/google/status`, {
    cache: "no-store",
  });

  if (!response.ok) {
    throw new Error(`Google auth status returned HTTP ${response.status}`);
  }

  return (await response.json()) as GoogleAuthStatus;
};

export const fetchKaiAuthStatus = async (): Promise<KaiAuthStatus> => {
  const response = await fetch(`${BACKEND_BASE_URL}/auth/me`, {
    cache: "no-store",
  });

  if (!response.ok) {
    throw new Error(`Kai auth status returned HTTP ${response.status}`);
  }

  return (await response.json()) as KaiAuthStatus;
};

const postAuthJSON = async <TPayload extends object>(path: string, payload: TPayload): Promise<KaiAuthStatus> => {
  const response = await fetch(`${BACKEND_BASE_URL}${path}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  const data = (await response.json()) as KaiAuthStatus;

  if (!response.ok || data.status === "error") {
    throw new Error(data.message ?? `Kai auth request failed with HTTP ${response.status}`);
  }

  return data;
};

export const signInWithEmail = (payload: EmailAuthPayload) => postAuthJSON("/auth/email/sign-in", payload);

export const signUpWithEmail = (payload: EmailAuthPayload) => postAuthJSON("/auth/email/sign-up", payload);

export const completeAuthSession = (payload: AuthCallbackPayload) => postAuthJSON("/auth/session", payload);

export const disconnectGoogle = async (): Promise<GoogleAuthStatus> => {
  const response = await fetch(`${BACKEND_BASE_URL}/auth/google/disconnect`, {
    method: "POST",
  });

  if (!response.ok) {
    throw new Error(`Google disconnect returned HTTP ${response.status}`);
  }

  return (await response.json()) as GoogleAuthStatus;
};

export const logoutKai = async (): Promise<void> => {
  const response = await fetch(`${BACKEND_BASE_URL}/auth/logout`, {
    method: "POST",
  });

  if (!response.ok) {
    throw new Error(`Kai logout returned HTTP ${response.status}`);
  }
};
