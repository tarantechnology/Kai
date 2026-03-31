#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod parser;

use std::sync::Mutex;

use serde::Serialize;
use tauri::{AppHandle, Emitter, LogicalSize, Manager, Size, WebviewWindow, WindowEvent};
use tauri_plugin_global_shortcut::{Code, GlobalShortcutExt, Modifiers, Shortcut, ShortcutState};

const PALETTE_WIDTH: f64 = 980.0;
const PALETTE_COLLAPSED_HEIGHT: f64 = 88.0;
const DASHBOARD_WIDTH: f64 = 1280.0;
const DASHBOARD_HEIGHT: f64 = 860.0;

// this enum is the native source of truth for which surface the window should show.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum SurfaceKind {
    Palette,
    Dashboard,
}

// tauri stores app-wide state here so shortcut handlers know what is currently open.
struct ActiveSurface(Mutex<SurfaceKind>);

#[derive(Clone, Serialize)]
struct SurfacePayload {
    surface: &'static str,
}

#[tauri::command]
fn hide_main_window(app: AppHandle) -> Result<(), String> {
    let window = app
        .get_webview_window("main")
        .ok_or_else(|| "main window not available".to_string())?;

    window.hide().map_err(|error| error.to_string())
}

#[tauri::command]
fn center_main_window(app: AppHandle) -> Result<(), String> {
    let window = app
        .get_webview_window("main")
        .ok_or_else(|| "main window not available".to_string())?;

    window.center().map_err(|error| error.to_string())
}

#[tauri::command]
fn set_palette_height(app: AppHandle, height: f64) -> Result<(), String> {
    let window = app
        .get_webview_window("main")
        .ok_or_else(|| "main window not available".to_string())?;

    window
        .set_size(Size::Logical(LogicalSize::new(PALETTE_WIDTH, height)))
        .map_err(|error| error.to_string())?;
    window.center().map_err(|error| error.to_string())?;

    Ok(())
}

#[tauri::command]
async fn parse_command_with_ollama(
    input: String,
    now: String,
) -> Result<parser::ParserResponse, String> {
    // the frontend sends raw text here, and rust forwards it to the active parser backend.
    parser::parse(input, now).await
}

#[tauri::command]
async fn warm_ollama_model() -> Result<(), String> {
    parser::warm().await
}

fn configure_window(window: &WebviewWindow, surface: SurfaceKind) -> tauri::Result<()> {
    // each surface has its own native size and behavior.
    let (width, height, always_on_top) = match surface {
        SurfaceKind::Palette => (PALETTE_WIDTH, PALETTE_COLLAPSED_HEIGHT, true),
        SurfaceKind::Dashboard => (DASHBOARD_WIDTH, DASHBOARD_HEIGHT, false),
    };

    window.set_size(Size::Logical(LogicalSize::new(width, height)))?;
    window.set_resizable(false)?;
    window.set_maximizable(false)?;
    window.set_always_on_top(always_on_top)?;
    window.set_shadow(false)?;
    window.show()?;
    window.center()?;
    window.set_focus()?;

    Ok(())
}

fn toggle_surface(app: &AppHandle, surface: SurfaceKind) -> tauri::Result<()> {
    let window = app
        .get_webview_window("main")
        .ok_or_else(|| tauri::Error::AssetNotFound("main window".into()))?;

    let is_visible = window.is_visible()?;
    let active_surface = app.state::<ActiveSurface>();
    let mut active_surface = active_surface
        .0
        .lock()
        .expect("active surface state poisoned");

    if is_visible && *active_surface == surface {
        // pressing the same shortcut twice hides the window instead of reopening a new one.
        window.hide()?;
        return Ok(());
    }

    configure_window(&window, surface)?;

    *active_surface = surface;

    app.emit(
        "kai://surface",
        SurfacePayload {
            surface: match surface {
                SurfaceKind::Palette => "palette",
                SurfaceKind::Dashboard => "dashboard",
            },
        },
    )?;

    Ok(())
}

fn main() {
    // these are the global shortcuts that can open kai from anywhere on the machine.
    let palette_shortcut = Shortcut::new(Some(Modifiers::SUPER), Code::Slash);
    let dashboard_shortcut = Shortcut::new(Some(Modifiers::SUPER), Code::Semicolon);
    let dashboard_shift_shortcut =
        Shortcut::new(Some(Modifiers::SUPER | Modifiers::SHIFT), Code::Semicolon);

    tauri::Builder::default()
        .manage(ActiveSurface(Mutex::new(SurfaceKind::Palette)))
        .invoke_handler(tauri::generate_handler![
            hide_main_window,
            center_main_window,
            set_palette_height,
            parse_command_with_ollama,
            warm_ollama_model
        ])
        .setup(move |app| {
            let handle = app.handle().clone();

            app.handle().plugin(
                tauri_plugin_global_shortcut::Builder::new()
                    .with_handler(move |app_handle, shortcut, event| {
                        if event.state() != ShortcutState::Pressed {
                            return;
                        }

                        // shortcuts map to a surface, then toggle_surface handles window logic.
                        let target_surface = if shortcut == &palette_shortcut {
                            Some(SurfaceKind::Palette)
                        } else if shortcut == &dashboard_shortcut
                            || shortcut == &dashboard_shift_shortcut
                        {
                            Some(SurfaceKind::Dashboard)
                        } else {
                            None
                        };

                        if let Some(target_surface) = target_surface {
                            let _ = toggle_surface(app_handle, target_surface);
                        }
                    })
                    .build(),
            )?;

            handle.global_shortcut().register(palette_shortcut)?;
            handle.global_shortcut().register(dashboard_shortcut)?;
            handle.global_shortcut().register(dashboard_shift_shortcut)?;

            let window = app
                .get_webview_window("main")
                .expect("main window should exist");

            window.hide()?;

            let window_for_close = window.clone();
            window.on_window_event(move |event| {
                if let WindowEvent::CloseRequested { api, .. } = event {
                    // the app behaves like a background utility, so close hides instead of exiting.
                    api.prevent_close();
                    let _ = window_for_close.hide();
                }
            });

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running kai desktop");
}
