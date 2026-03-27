# Kai V1 Design Document

## 1. Overview

**Kai** is a local-first personal command layer for managing calendar events, tasks, reminders, and school commitments through a fast desktop interface. The product combines a **Spotlight-style quick command palette** with a **full CRM-style dashboard**, allowing users to create reminders, block time, import Canvas assignments, view their schedule, and manage tasks through natural language. Kai is designed around three principles: **privacy**, **low latency**, and **trust**.

For V1, Kai will support:
- **Google Calendar**
- **Canvas**
- **On-device natural language parsing**
- **Offline capture with queued sync**
- **A quick command UI (`cmd + /`)**
- **A main dashboard (`cmd + :`)**
- **A liquid glass visual system** with frosted panels, subtle blur, layered translucency, and premium low-noise motion

Kai is **not** a cloud-first productivity app. It is a **local operating layer for time and commitments**.

---

## 2. Product Goals

### Primary goals
- Let users interact with scheduling and tasks through natural language
- Make all common actions feel instant
- Keep sensitive interpretation on-device
- Support offline usage for capture and later sync
- Unify Google Calendar and Canvas into one clean view
- Make the UI feel premium, minimal, and trustworthy
- Build around a **liquid glass UI language** that feels modern, calm, and OS-native

### Non-goals for V1
- Email parsing
- Outlook integration
- Full multi-agent orchestration
- Open-ended autonomous planning
- Mobile app
- Cross-device sync
- Team collaboration
- Full contact/relationship CRM

---

## 3. Core User Experience

### Surface 1: Quick Command Palette (`cmd + /`)
This is the fast-action layer.

#### Purpose
- Create reminders
- Create events
- Show remaining tasks
- Show schedule
- Trigger syncs
- Ask simple, structured questions

#### Example commands
- “remind me at 9 pm to finish homework”
- “block 2 hours tomorrow for interview prep”
- “what else do i have today”
- “show me my calendar tomorrow”
- “import my canvas deadlines for this week”

#### UX behavior
- Opens instantly
- Minimal glass input UI
- Accepts natural language
- Shows one of:
  - loading state
  - success toast/checkmark
  - expandable list
  - confirmation card
  - error/clarification state

### Surface 2: Main Dashboard (`cmd + :`)
This is the full CRM/home.

#### Purpose
- View today’s schedule
- View calendar
- View/manage tasks
- See imported Canvas items
- Manage sync status
- Manage settings and connected accounts
- Review privacy settings and logs

---

## 4. Product Principles

### Privacy-first
- Natural language interpretation happens on-device
- Sensitive command text does not need to be sent to the cloud
- Google and Canvas data are only fetched as needed
- Only minimal structured state is stored locally
- Tokens are stored in OS keychain, not plain database fields

### Low latency
- All UI-critical actions happen locally
- The backend is never on the hot path for normal commands
- The app writes local state first and syncs later
- Quick actions should feel instant even offline

### Trustworthy behavior
- The LLM interprets only
- The app executes only validated structured actions
- Important actions can require confirmation
- Users can see what happened and why

### Liquid glass UI
- Use dark premium backgrounds with frosted translucent panels
- Use layered blur, subtle edge highlights, and low-opacity fills
- Use restrained animation, not flashy motion
- Use soft depth, clean typography, and sparse blue/white accents
- The app should feel like a native operating layer, not a chatbot window

---

## 5. V1 Functional Requirements

### 5.1 Natural language command support
Kai must support:
- create reminder
- create event / time block
- show tasks for a time range
- show calendar for a time range
- sync Canvas assignments
- sync Google Calendar
- mark task complete
- delete task
- reschedule task or event if structured enough

### 5.2 Calendar support
Kai must:
- read Google Calendar events
- create Google Calendar events
- update Kai-created Google events
- cache calendar data locally
- show day/week views

### 5.3 Canvas support
Kai must:
- connect to Canvas
- fetch assignments and due dates
- optionally fetch course calendar items
- normalize Canvas items into Kai task/event format
- show source labels like “Canvas”

### 5.4 Offline support
Kai must:
- accept commands offline
- create local tasks/reminders/events offline
- queue external sync actions
- retry syncing once online

### 5.5 Settings/auth
Kai must:
- connect/disconnect Google
- connect/disconnect Canvas
- expose privacy controls
- show local-first status
- show pending sync state

---

## 6. V1 Non-Functional Requirements

### Performance
- Quick palette open time should feel near-instant
- Parsing and local execution should feel responsive
- Task and calendar views should load from local cache immediately

### Reliability
- No lost offline actions
- Retriable sync queue
- Idempotent remote writes
- Safe handling of network failures

### Security
- Secure token storage
- Minimal remote persistence
- Local data isolation

### Maintainability
- Clear command schema
- Deterministic executor
- Thin backend
- Modular sync engine

### UI quality
- Consistent liquid glass system across quick palette and full dashboard
- Consistent spacing, typography, and motion rules
- Low visual clutter
- Strong affordance for trust, source labeling, and sync state

---

## 7. System Architecture

### 7.1 High-level architecture

Kai consists of three major parts:

#### A. Desktop app
The desktop app is the primary product runtime.

Responsibilities:
- UI
- local LLM parsing
- local storage
- command execution
- sync queue
- Google/Canvas sync
- notifications
- dashboard rendering

#### B. Thin backend
The backend exists only for support infrastructure.

Responsibilities:
- OAuth callbacks
- token exchange helpers
- webhook endpoints later
- optional minimal metadata
- device/account registration if needed

#### C. External systems
- Google Calendar API
- Canvas API

---

## 8. Chosen Stack

### Desktop
- **Tauri**
- **React**
- **TypeScript**
- **Tailwind CSS**
- **Zustand** for UI state
- **SQLite** for local persistence
- **OS keychain** for tokens/secrets

### Local AI
- **On-device LLM**
- Prefer **MLX** if Mac-first
- Alternative: Ollama if easier dev velocity is needed

### Backend
- **Go**
- **Chi** router
- **Postgres** for minimal backend persistence
- Deploy on Fly / Railway / Render / AWS

---

## 9. Why These Choices

### Why Tauri
- Lower overhead than Electron
- Better fit for low-latency desktop utility
- Good for Mac-first polished app
- Rust side is strong for native integration and background work

### Why SQLite
- Local-first persistence
- Offline support
- Instant reads/writes
- Stores structured state, queue, cached tasks/events, sync metadata

### Why Go backend
- Thin, fast, reliable service
- Great for OAuth, webhooks, token lifecycle, and operational endpoints
- Low memory use and easy deployment
- Keeps backend boring and production-friendly

### Why on-device LLM
- Better privacy
- Offline support
- Natural input without sending every query to cloud infra
- Still allows strict local validation before execution

---

## 10. Core Architectural Pattern

### Key principle
**Natural language understanding is flexible. Execution is strict.**

### Flow
1. User enters command
2. On-device LLM parses into strict command schema
3. Validator checks schema
4. Command executor performs local action
5. Local DB updates immediately
6. External sync is queued if needed
7. UI updates instantly
8. Background sync completes later

The LLM does **not**:
- directly call Google APIs
- directly write the DB
- directly manage queue logic
- directly resolve sync conflicts

It only produces structured output.

---

## 11. Command Model

### 11.1 Strict command schema

```ts
type KaiCommand =
  | {
      type: "create_reminder";
      title: string;
      datetime: string;
      sourceText: string;
      confidence: number;
    }
  | {
      type: "create_event";
      title: string;
      start: string;
      end: string;
      sourceText: string;
      confidence: number;
    }
  | {
      type: "show_tasks";
      range: "today" | "tomorrow" | "week";
      sourceText: string;
      confidence: number;
    }
  | {
      type: "show_calendar";
      range: "day" | "tomorrow" | "week";
      sourceText: string;
      confidence: number;
    }
  | {
      type: "sync_canvas";
      range: "today" | "week" | "all";
      sourceText: string;
      confidence: number;
    }
  | {
      type: "sync_google_calendar";
      range?: "today" | "week" | "all";
      sourceText: string;
      confidence: number;
    }
  | {
      type: "complete_task";
      taskId: string;
      sourceText: string;
      confidence: number;
    }
  | {
      type: "delete_task";
      taskId: string;
      sourceText: string;
      confidence: number;
    };
```

### 11.2 Validation rules
- type must be one of the allowed actions
- required fields must be present
- date/time must parse cleanly
- confidence below threshold triggers clarification or confirmation
- no executor action runs on invalid schema

---

## 12. UI Design Requirements

### 12.1 Visual language
Kai should use:
- dark liquid-glass aesthetic
- frosted, semi-transparent panels
- subtle motion
- restrained blue/white accents
- clean typography
- minimal clutter

### 12.2 Liquid glass design system
Use these visual rules consistently:

#### Background
- Deep dark gradient base
- Soft vignette
- Optional low-opacity noise texture
- Slight atmospheric glow behind primary surfaces

#### Panels
- Semi-transparent fill
- Frosted blur
- Thin translucent border
- Soft drop shadow
- Rounded corners, premium spacing

#### Motion
- Quick palette should scale and fade in softly
- Success states should use a subtle slide/check animation
- Expanded task/schedule views should grow fluidly
- Avoid heavy springy animations or flashy transitions

#### Color treatment
- Text: near-white primary, muted gray secondary
- Accent: restrained blue
- Success: green used sparingly
- Keep palette calm and trust-oriented

### 12.3 Main UI surfaces

#### Quick Command Palette
Used for:
- fast command entry
- instant response
- success toasts
- compact result lists
- confirmation cards

States:
- idle
- listening/typing
- understanding/loading
- success toast
- confirmation card
- task list result
- schedule result
- error/clarification

#### Main Dashboard
Sections:
- Today
- Calendar
- Tasks
- Canvas
- Settings
- Account connections
- Privacy
- Sync status

---

## 13. Key UX Flows

### 13.1 Create reminder
User command:
“remind me at 9 pm to finish homework”

Flow:
1. quick palette opens
2. user types or speaks
3. local LLM parses command
4. validator approves
5. reminder/task created locally
6. UI shows loading then success checkmark
7. if sync target exists, queue sync action
8. sync worker later creates Google event/reminder if needed

### 13.2 Show remaining tasks
User command:
“what else do i have today”

Flow:
1. local LLM returns `show_tasks today`
2. app queries SQLite
3. quick palette expands into task list view
4. user can close or click into dashboard

### 13.3 Create calendar block
User command:
“block 2 hours tomorrow for interview prep”

Flow:
1. parse into event schema
2. local event written immediately
3. success view shown
4. Google event sync queued
5. background sync writes to Google when online

### 13.4 Import Canvas assignments
User command:
“import my canvas deadlines for this week”

Flow:
1. parse into `sync_canvas week`
2. app queues sync job
3. sync worker fetches assignments from Canvas
4. app normalizes them into local task/event records
5. UI shows imported items

---

## 14. Data Model

### 14.1 Local database overview
SQLite is used to store:
- tasks
- events
- reminders
- Canvas items
- linked account metadata
- sync cursors/state
- action queue
- action logs
- settings

### 14.2 Tables

#### users
Minimal local profile.

Fields:
- id
- display_name
- created_at
- updated_at

#### linked_accounts
Stores connected provider metadata.

Fields:
- id
- provider (`google`, `canvas`)
- account_identifier
- status
- connected_at
- last_sync_at
- created_at
- updated_at

#### tasks
Stores local and imported tasks.

Fields:
- id
- title
- due_at
- status (`todo`, `done`, `archived`)
- source_type (`kai`, `canvas`, `google`)
- source_ref
- notes
- created_at
- updated_at

#### reminders
If separated from tasks, store reminder-specific behavior.

Fields:
- id
- task_id
- remind_at
- notification_type
- synced_externally
- created_at
- updated_at

#### events
Stores calendar events and blocks.

Fields:
- id
- title
- start_at
- end_at
- timezone
- status
- source_type (`kai`, `google`, `canvas`)
- source_ref
- google_event_id
- created_locally
- needs_review
- created_at
- updated_at

#### canvas_items
Stores imported Canvas entities.

Fields:
- id
- canvas_id
- course_name
- title
- due_at
- url
- item_type
- last_synced_at
- hash
- created_at
- updated_at

#### action_queue
Stores pending sync actions.

Fields:
- id
- type
- payload_json
- status (`pending`, `processing`, `synced`, `failed`, `needs_user_action`)
- retry_count
- idempotency_key
- last_error
- created_at
- updated_at

#### sync_state
Stores cursors/timestamps for incremental sync.

Fields:
- id
- provider
- cursor
- last_sync_at
- sync_scope
- created_at
- updated_at

#### action_log
Stores user-visible or diagnostic actions.

Fields:
- id
- action_type
- description
- source_type
- source_ref
- created_at

#### settings
Stores local app settings.

Fields:
- key
- value
- updated_at

---

## 15. What Is Stored Where

### Stored in SQLite
- tasks
- reminders
- events
- imported Canvas assignments
- Google event mappings
- queue state
- sync state
- settings
- action log

### Stored in OS keychain
- Google access token
- Google refresh token
- Canvas token
- any sensitive auth secrets

### Not stored unless necessary
- full emails
- unnecessary raw remote content
- large content dumps
- private content unrelated to command execution

---

## 16. Sync Architecture

### 16.1 Core idea
Kai is local-first. Remote systems are synced asynchronously.

### 16.2 Sync directions

#### Google
- read remote calendar events
- write Kai-created events
- update Kai-owned Google events
- cache local copies for fast display

#### Canvas
- read assignments and course calendar items
- normalize into local task/event forms
- read-only in V1

### 16.3 Queue-driven sync
All remote writes should be queued.

Benefits:
- offline support
- retries
- crash recovery
- observable sync state
- non-blocking UI

---

## 17. Action Queue Design

### Queue item structure
Each queue item contains:
- type
- payload
- status
- retry count
- idempotency key
- timestamps
- last error

### Queue types
Examples:
- `create_google_event`
- `update_google_event`
- `delete_google_event`
- `sync_canvas_week`
- `sync_google_calendar`
- `refresh_google_token`
- `refresh_canvas_token`

### Retry policy
- use exponential backoff
- max retry threshold
- user-visible error if permanently failed
- idempotency keys prevent duplicate remote writes

---

## 18. Conflict Handling

### V1 approach
Keep conflict handling simple and conservative.

### Rules
- local unsynced edits should not be overwritten silently
- Canvas imports are read-only
- if both local and remote changed, mark item as `needs_review`
- Kai-owned Google events can be updated by Kai
- imported remote items should retain source metadata

### Source precedence
1. explicit local user action
2. remote synced Google data
3. Canvas imported data

---

## 19. Local LLM Design

### Role of the LLM
Only interpret language into structured commands.

### It must not
- directly call APIs
- directly mutate the database
- invent unsupported actions
- do open-ended autonomous planning in V1

### Input
- user command text
- optional lightweight local context
  - current date/time
  - current quick palette mode
  - maybe a small shortlist of today’s tasks for reference

### Output
Strict JSON-like schema matching allowed command types.

### Confidence handling
- high confidence: execute immediately if low risk
- medium confidence: show confirmation card
- low confidence: ask follow-up

---

## 20. Backend Design

### 20.1 Purpose
The backend is support infrastructure, not the product brain.

### 20.2 Responsibilities
- Google OAuth start/callback
- Canvas OAuth start/callback
- token exchange helpers
- webhook endpoints later
- health/status endpoints
- optional account/device registration
- optional encrypted operational metadata

### 20.3 Suggested routes

#### Auth
- `GET /auth/google/start`
- `GET /auth/google/callback`
- `GET /auth/canvas/start`
- `GET /auth/canvas/callback`

#### Health
- `GET /health`

#### Webhooks later
- `POST /webhooks/google`

#### Optional device/account
- `POST /device/register`
- `POST /device/heartbeat`

### 20.4 Backend persistence
Postgres should store only what is operationally needed:
- user/device records if used
- OAuth flow/session metadata
- subscription/webhook metadata
- audit/ops metadata if needed

It should not be the main task/calendar store for V1.

---

## 21. Tauri App Architecture

### Main process / native side responsibilities
- global shortcuts
- native window management
- system notifications
- keychain access
- file/local DB access
- background workers
- network detection
- auth callback handoff
- IPC bridge to UI

### Renderer responsibilities
- command palette UI
- dashboard UI
- calendar/task views
- settings
- list rendering
- local interaction state

### IPC guidelines
- keep interfaces typed
- expose only necessary operations
- avoid putting secret-handling in renderer

---

## 22. Security and Privacy Model

### Security requirements
- use OS keychain for secrets
- avoid storing raw tokens in SQLite
- validate all command outputs before execution
- use secure OAuth flows
- isolate backend to minimal operational use

### Privacy requirements
- parse commands on-device
- store minimal structured data
- avoid full raw content storage when not required
- clearly show local-first/privacy language in settings
- allow user to disconnect and clear imported data

### User trust features
- action log
- source badges on tasks/events
- sync status indicators
- “why did this show up?” details
- “clear synced Canvas items” option

---

## 23. Performance Plan

### Goal
The app should feel instant.

### Strategies
- local-first command path
- SQLite for cached reads
- background sync only
- do not block UI on Google/Canvas/network
- lazy-load heavier views only when needed
- keep quick palette extremely light

### Critical rule
**No normal quick command should require waiting on the backend.**

---

## 24. Error Handling Strategy

### Parse errors
- show clarification UI
- do not guess silently

### Sync failures
- keep local item intact
- mark queue item failed
- retry in background
- show status if persistent

### Auth failures
- show reconnect prompt
- preserve local state
- do not delete user-created items

### Remote API failures
- log error
- retry if safe
- allow manual retry from settings/status

---

## 25. UI Screens to Design and Build

### Quick palette
- idle
- command typing
- understanding/loading
- success toast
- confirmation card
- task list result
- schedule result
- error/clarification

### Main dashboard
- Today view
- Calendar view
- Tasks view
- Canvas/imported items view
- Settings
- Auth/connect accounts
- Privacy/sync status

---

## 26. Engineering Modules

### Frontend/UI
- command palette
- dashboard shell
- calendar components
- task list components
- settings/auth pages
- toast/result components

### Local core
- command parser client
- command validator
- command executor
- SQLite layer
- queue manager
- sync engine
- notification manager
- source normalization logic

### Integrations
- Google Calendar client
- Canvas client
- auth session handlers

### Backend
- auth server
- operational DB layer
- webhook server
- device/account endpoints if needed

---

## 27. Suggested Repository Structure

```txt
kai/
  apps/
    desktop/
      src/
        ui/
          components/
          pages/
          layouts/
          hooks/
          store/
        core/
          commands/
          executor/
          queue/
          sync/
          db/
          integrations/
          notifications/
          security/
        tauri/
  services/
    backend/
      cmd/
      internal/
        auth/
        handlers/
        middleware/
        db/
        config/
        webhooks/
      migrations/
  packages/
    shared/
      types/
      schemas/
```

---

## 28. Build Order

### Phase 1: foundations
- finalize command schema
- initialize Tauri app
- set up React + Tailwind
- design quick palette + main shell
- set up SQLite
- set up keychain integration

### Phase 2: local execution
- local LLM integration
- command parser wrapper
- command validation
- executor for local task/reminder/event creation
- action log
- quick palette state handling

### Phase 3: Google integration
- Go backend auth flow
- Google connection
- create/read Google Calendar events
- queue remote writes
- sync state persistence

### Phase 4: Canvas integration
- Canvas auth flow
- fetch/import assignments
- normalize Canvas items
- display source-tagged tasks/events

### Phase 5: polish and trust
- sync status UI
- privacy settings
- error handling and retry flows
- confirmation flows
- action log UI
- performance tuning

---

## 29. MVP Definition

Kai V1 is complete when a user can:
- open quick palette with shortcut
- type natural language commands
- create local reminders and events
- see tasks and schedule locally
- connect Google Calendar
- sync Kai-created events to Google
- connect Canvas
- import Canvas assignments
- use the app offline for capture
- have actions sync once online again
- open main dashboard with shortcut
- review settings and connected accounts

---

## 30. Risks and Mitigations

### Risk: command misinterpretation
Mitigation:
- strict schema
- confidence thresholds
- confirmation UI

### Risk: sync duplication
Mitigation:
- idempotency keys
- source refs
- local-remote mapping table

### Risk: conflict handling complexity
Mitigation:
- conservative V1 conflict policy
- `needs_review` state

### Risk: local model performance
Mitigation:
- keep prompt/context small
- support smaller model
- optimize command set for structured tasks

### Risk: auth complexity
Mitigation:
- thin backend only
- implement one provider at a time
- keep operational metadata minimal

---

## 31. Open Decisions Still Needed

You still need to decide:
- exact local model and serving setup
- whether reminders are separate from tasks or part of task model
- whether Google reminders are mapped to events or separate notifications
- whether Canvas course calendar items are included in V1 or just assignments
- exact confirmation threshold logic
- exact dashboard information hierarchy
- whether voice input is in or out for V1

---

## 32. Final Architectural Summary

Kai is a **local-first desktop command layer** that lets users manage their schedule and responsibilities through natural language while preserving privacy and speed. The desktop app is the true runtime and source of truth. An on-device LLM interprets user commands into a strict schema. The app validates and executes those commands locally, stores structured state in SQLite, and queues remote sync actions for Google Calendar and Canvas. A thin Go backend supports OAuth and operational plumbing only. The system is designed so that natural input feels human, while execution remains deterministic, safe, and low latency.

---

## 33. Codex Handoff Notes

Use this document as the source of truth for V1 implementation.

### Must preserve
- local-first architecture
- on-device command parsing
- Tauri desktop shell
- React + TypeScript frontend
- Go backend
- SQLite local persistence
- Google + Canvas only for V1
- queue-based sync
- liquid glass UI language

### Must avoid for V1
- Electron
- Outlook
- email parsing
- cloud-first command execution
- open-ended agent loops
- heavy backend business logic
- blocking UI on remote calls

### Coding priorities
1. establish architecture and local data flow
2. make quick palette feel instant
3. get local create/show actions working
4. build queue and offline behavior
5. add Google integration
6. add Canvas integration
7. polish dashboard and settings
8. polish liquid glass UI system
