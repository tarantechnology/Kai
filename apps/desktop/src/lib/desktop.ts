import type { DesktopSurface } from "./types";

const SURFACE_EVENT = "kai://surface";

declare global {
  interface Window {
    __TAURI_INTERNALS__?: unknown;
  }
}

interface SurfacePayload {
  surface: DesktopSurface;
}

// this checks whether react is running inside tauri or just in a plain browser dev session.
export const isTauriRuntime = () =>
  typeof window !== "undefined" && typeof window.__TAURI_INTERNALS__ !== "undefined";

export const hideKaiWindow = async () => {
  if (!isTauriRuntime()) {
    return;
  }

  const { invoke } = await import("@tauri-apps/api/core");
  await invoke("hide_main_window");
};

export const centerKaiWindow = async () => {
  if (!isTauriRuntime()) {
    return;
  }

  const { invoke } = await import("@tauri-apps/api/core");
  await invoke("center_main_window");
};

export const setPaletteHeight = async (height: number) => {
  if (!isTauriRuntime()) {
    return;
  }

  const { invoke } = await import("@tauri-apps/api/core");
  await invoke("set_palette_height", { height });
};

export const warmLocalParser = async () => {
  if (!isTauriRuntime()) {
    return;
  }

  const { invoke } = await import("@tauri-apps/api/core");
  await invoke("warm_ollama_model");
};

export const bindSurfaceListener = async (onSurface: (surface: DesktopSurface) => void) => {
  if (!isTauriRuntime()) {
    return () => undefined;
  }

  const { listen } = await import("@tauri-apps/api/event");

  // rust emits one event when it decides which surface should be visible.
  const unlisten = await listen<SurfacePayload>(SURFACE_EVENT, (event) => {
    onSurface(event.payload.surface);
  });

  return () => {
    unlisten();
  };
};
