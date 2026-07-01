# RapidIdentity Connect — MCP Workflow, JSON Object Model & XML ↔ JSON Conversion

Read this when working against a live Connect instance via the `RapidIdentity MCP Server:*`
tools, or when converting action sets between XML (file mode) and JSON (API model).

## Contents
- MCP Workflow — read/explore, design, push, delete
- JSON Object Model — file JSON vs API JSON, argDef/action/arg shapes
- XML ↔ JSON Conversion — field-by-field mapping in both directions

---

## MCP Workflow

Use these operations when in API mode (`RapidIdentity MCP Server:*` tools available).

### Read / Explore

1. `get-connect-projects` — discover available projects
2. `get-connect-actions(project, metaDataOnly: true)` — list action sets in a project without
   pulling full content; pass an empty string for `project` to search all projects
3. `get-connect-action(id, metaDataOnly: false)` — fetch a single action set in full for
   reading or editing; `id` accepts a UUID or `project.name` format
4. `get-connect-actions(project: "$builtin", metaDataOnly: false)` — fetch the full catalog of
   platform built-in actions; load this before designing any new action set to know what
   actions are available to use
5. When the user names an action set without specifying a project, call
   `get-connect-actions` across all projects first to locate it before fetching

### Design (new from scratch)

The JSON object model below documents the wire shapes Connect uses for sections and nested actions.
Critical rules:

**Sections exist in the model and can be nested to any depth.** A section's `args` contains
only the fields that are set — `label` and `suppressTrace` are each optional, `do` is required.
Always include `label` and `suppressTrace: "true"` on every section you author. Sections without
them are valid but are an anti-pattern (they lose trace suppression and become unlabelled in the UI).

**Nesting args (`do`/`then`/`else`) must NOT have a `value` field.** The presence of `"value"`
(even `""`) tells Connect the arg is a scalar — it ignores `"actions"` and stores an empty body,
causing `Property must be a list of actions 'section.do'` (or `if.then`, `forEach.do`, etc.).
Use only `{"name": "do", "actions": [...]}` — no `value` key at all.

**Every `if` must have both `then` and `else`.** Always include both args. Empty `else` is a bare object: `{"name": "else"}` — no `value`, no `actions`. Omitting `else` entirely causes a compile error.

**`while` has only `condition` + `do` — no `else`.**
```json
{"id": "...", "name": "while", "args": [
  {"name": "condition", "value": "count < 5"},
  {"name": "do", "actions": [...]}
]}
```

**Disabled actions** use `"disabled": true` on the action object. This is the only case where `disabled` appears in the wire format — enabled actions omit it entirely:
```json
{"id": "...", "name": "log", "disabled": true, "args": [...]}
```

**Workflow for new action sets:**
1. Author the full action set JSON with sections intact, full nested logic, and nesting args
   using only `name` + `actions` (never `value`)
2. Push via `save-connect-action`
3. Run to verify — if a compile error names `section.do` or `if.then`, a nesting arg has a
   spurious `value` field

Follow all existing naming conventions, section order, argDef rules, and logging conventions
from this skill when designing the structure.

### Push (create or update metadata)

- Use `save-connect-action` to create or update the full action set including nested action logic
- On **update**: always use the `version` number returned from the most recent
  `get-connect-action` call — never assume or hardcode the version number
- If a version conflict error is returned, re-fetch the action set to get the current version
  before retrying

### Delete

- Use `delete-connect-action(id)`
- **Always confirm with the user before executing** — deletion is irreversible

---

## JSON Object Model

### File JSON (downloaded from Connect UI)

The minimal format Connect exports when downloading an action set as JSON. Contains only:

| Field | Type | Notes |
|---|---|---|
| `name` | string | Action set name — same naming rules as XML |
| `project` | string | Project name; empty string for the `<Main>` project |
| `category` | string | Usually empty string |
| `description` | string | Same description template as XML |
| `returnsValue` | boolean | `true` for `Fn*` functions; `false` for jobs/utilities |
| `builtIn` | boolean | Always `false` for user-defined action sets |
| `community` | boolean | Always `false` for user-defined action sets |
| `argDefs` | array | See argDef shape below |
| `actions` | array | See action shape below |

### API JSON (used with `save-connect-action` / returned by `get-connect-action`)

All file JSON fields plus these API/server fields:

| Field | Type | Value on create | Notes |
|---|---|---|---|
| `id` | UUID string | Generate new UUID v4 | Required |
| `version` | integer | `0` | Must match server value on update |
| `sensitive` | boolean | `false` | `true` only if the action set handles credentials |
| `unlicensed` | boolean | `false` | Always `false` |
| `deprecated` | string | `""` | Empty string if not deprecated |
| `changeCount` | integer | `0` | Server manages on update |
| `modifiedMs` | integer | `0` | Server manages on update |
| `modifiedBy` | string | `""` | Server manages on update |
| `modifiedByName` | string | `""` | Server manages on update |
| `httpStatus` | integer | `0` | Server field |

### argDef shape

File JSON omits fields that are absent; API JSON must include all fields.

| Field | Type | Notes |
|---|---|---|
| `name` | string | Parameter name |
| `type` | string | `string`, `boolean`, `number`, `object`, `array`, `enum:val1,val2,...` |
| `description` | string | Required — same rule as XML |
| `optional` | boolean | Only present when `true` |
| `value` | string | Default value; absent when none |

### Action shape

The wire format is minimal — only include fields that are set:

| Field | Type | Notes |
|---|---|---|
| `id` | UUID string | Always present — generate UUID v4 per action on create |
| `name` | string | Always present — built-in action name or custom action set name |
| `outputVar` | string | Only include when non-empty |
| `disabled` | boolean | Only include when `true` (disabled action); omit entirely when `false` |
| `project` | string | Omit entirely — not part of the native wire format |
| `args` | array | Always present, even if empty (`[]`) |

### Arg shapes

There are exactly two mutually exclusive arg forms. Never mix them:

*Value arg* — any arg carrying an expression, literal, or scalar:
```json
{ "name": "condition", "value": "someExpression" }
```

*Nesting arg* — `do`, `then`, `else`, or any arg containing child actions. **No `value` field.**
```json
{ "name": "do", "actions": [ ...child action objects... ] }
```

*Bare `else`* — empty else branch with no children:
```json
{ "name": "else" }
```

The `optional`, `type`, and `description` fields are omitted on both forms in the native API
wire format. The MCP tool schema previously required them but the struct has been updated to
use `omitempty` — pass them only when they carry meaningful values.

---

## XML ↔ JSON Conversion

### XML → JSON (file/disk to API)

Use when taking a downloaded or hand-authored XML action set and pushing it to the live system
via `save-connect-action`.

**Action set wrapper:**

| XML | JSON |
|---|---|
| `<actionDef name="X" returnsValue="true" description="...">` | `name`, `returnsValue`, `description` top-level fields |
| `<argDefs><argDef name="X" type="Y" optional="true" description="Z"/>` | `argDefs` array entry; add `value: ""` for API JSON |
| Standalone `<actionDefs>` import wrapper | Discard — not part of JSON model |

**Actions and args:**

| XML | JSON |
|---|---|
| `<action name="section">` | `{ "name": "section", "args": [ {label arg}, {suppressTrace arg}, {do nesting arg} ] }` — sections are preserved as-is |
| `<action name="X" outputVar="Y">` | Action object: `name`, generated UUID `id`, `outputVar: "Y"` |
| `<action name="X">` (no outputVar) | Action object: `name`, generated UUID `id`; omit `outputVar` |
| `<arg name="do">` / `<arg name="then">` / `<arg name="else">` containing child actions | `{ "name": "do", "actions": [...child action objects...] }` — **no `value` field** |
| `<arg name="else"/>` (empty else, no child actions) | `{ "name": "else" }` — bare name only, no `value`, no `actions` |
| `<arg name="X" value="someExpr"/>` (any non-nesting arg) | `{ "name": "X", "value": "someExpr" }` |

**String escaping (XML → JSON):**

| XML attribute | JSON value |
|---|---|
| `&quot;` | `"` |
| `&amp;` | `&` |
| `&lt;` | `<` |
| `&gt;` | `>` |

**API field defaults to supply when pushing file JSON to MCP:**
- Top-level: `id` (new UUID v4), `version: 0`, `sensitive: false`, `unlicensed: false`,
  `deprecated: ""`, `changeCount: 0`, `modifiedMs: 0`, `modifiedBy: ""`,
  `modifiedByName: ""`, `httpStatus: 0`
- Each action: `disabled: false`, `project: "$builtin"` (or the project name for calls to
  custom action sets)
- Each value arg: `type: ""`, `optional: false`, `description: ""`
- Each argDef: `optional: false` if absent, `value: ""` if absent

---

### JSON → XML (API to file/disk)

Use when taking a live action set fetched via `get-connect-action` and saving it as a local
XML file.

**Action set wrapper:**

| JSON | XML |
|---|---|
| `name`, `returnsValue`, `description` | `<actionDef name="X" returnsValue="true/false" description="...">` |
| `argDefs` array | `<argDefs>` block with one `<argDef>` per entry; omit `optional` attribute if `false`; omit `value` attribute if empty string |
| `id`, `version`, `sensitive`, `unlicensed`, `deprecated`, `changeCount`, `modifiedMs`, `modifiedBy`, `modifiedByName`, `httpStatus` | **Omit entirely** — API metadata, no XML equivalent |

**Actions and args:**

| JSON | XML |
|---|---|
| Action object `name` + non-empty `outputVar` | `<action name="X" outputVar="Y">` |
| Action object `name` + empty `outputVar` | `<action name="X">` — omit the attribute |
| `disabled`, `project` on action object | **Omit** — API-only fields |
| `{ "name": "do", "actions": [...] }` (file JSON nesting arg) | `<arg name="do">` containing child `<action>` elements (recurse for nested nesting args) |
| `{ "name": "do", "value": "", "actions": [...] }` (API nesting arg) | `<arg name="do">` containing child `<action>` elements (recurse for nested nesting args) |
| `{ "name": "X", "value": "expr" }` (any non-nesting arg) | `<arg name="X" value="expr"/>` |

**String escaping (JSON → XML):**

| JSON value | XML attribute |
|---|---|
| `"` | `&quot;` |
| `&` | `&amp;` |
| `<` | `&lt;` |
| `>` | `&gt;` |

**Output format:** Apply all standard XML format rules from § XML Format Rules — no `<?xml`
declaration, no indentation, single-line compact output, all sections with
`suppressTrace="true"`.
