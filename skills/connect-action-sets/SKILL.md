---
name: connect-action-sets
description: >
  Write, review, edit, validate, or refactor RapidIdentity (idauto) Connect action sets in XML.
  Use this skill whenever the user asks to create a new action set, fix an existing one, add a
  section, refactor logic, apply coding standards, rename action sets or section labels, write
  logging/counting code, build a scheduled job or function, or do anything else that involves
  Connect XML. Also trigger when the user asks about naming conventions, section structure,
  suppressTrace, logging, counters, argDefs, parameter descriptions, or any other RapidIdentity
  Connect platform detail. If the user shows XML or references a .dssproject file in any way,
  use this skill.
---

# Connect Action Sets — Authoring Skill

RapidIdentity Connect is an XML-based identity workflow engine by idauto. Action sets are the
core unit of logic. This skill encodes Connect coding standards and XML authoring rules based
on built-in platform actions only.

**Always read `CLAUDE.md` (at the root of the files folder) first if present** — it is the authoritative,
session-specific source of truth and may define project-specific conventions that override or
extend this skill.

---

## Working Mode Detection

Detect which mode to use based on available tools and the user's request:

| Context | Mode |
|---|---|
| `RapidIdentity MCP Server:*` tools are available and the user is working against a live Connect instance | **API mode** — use MCP tools for discovery and simple flat creates |
| Working with `.dssproject` files or XML strings on disk / in the conversation | **File mode** — use XML |
| Both available | Follow what the user's request implies — but author all real logic in file mode (see caveat) |

**In API mode:** Always call `get-connect-projects` and `get-connect-actions` to orient before
doing any design or editing work.

> **MCP capability caveat.** The current `RapidIdentity MCP Server` read path (`get-connect-action`)
> returns nesting args (`do`/`then`/`else`) as empty scalars and **omits all child action bodies** —
> you get the top-level section/action skeleton but not the logic inside. The `save-connect-action`
> schema is correspondingly flat. So the MCP is reliable for **discovery, metadata, and simple flat
> action sets**, but **anything with sections, branches, or loops (i.e. essentially all real action
> sets) must be authored and delivered as XML in file mode** — produce the importable `.dssproject`
> XML and have the user import it. Do **not** round-trip an existing action set through
> `get-connect-action` → `save-connect-action`: it will silently drop every nested body. (A fix to
> the MCP is expected; re-verify before relying on nested reads/writes.)

**In file mode:** All existing XML authoring rules in this skill apply unchanged.

---

## Quick Reference

| Need | Go to |
|---|---|
| XML format rules, root element, quoting | § XML Format Rules |
| setVariable expression compiler rules, JSON strings, StackOverflow avoidance | § setVariable Expression Compiler Rules |
| setVariable vs copyRecord — reference vs deep copy, array mutation rule | § setVariable vs copyRecord |
| Naming (prefixes, camelCase, reserved labels) | § Naming Scheme |
| `about` section content | § about Section |
| `defineDefaultVariables` content | § defineDefaultVariables Section |
| Logging — built-in `log` action | § Logging |
| Counting — variable-based counters | § Counting |
| Control flow — forEach, if/while, break, labeled loops/continue | § Control Flow |
| JavaScript & engine idioms — arrow fns, named fns, Set, filter, stringify | § JavaScript & Engine Idioms |
| Try/Catch — error handling via a JS function | § Try/Catch |
| String & record built-in actions (split, contains, equals, pad, record fields) | § String & Record Built-in Actions |
| Calling other action sets (function mode) | § Function Mode Pattern |
| Connections — typed actions per target system (incl. AES adapter, callGoogleAPI) | § Connections → `references/connections.md` |
| Global property references | § Global Properties |
| HTTP actions — REST API patterns | § HTTP Actions |
| `delay` action — timed pauses | § delay Action |
| LDAP polling loop pattern | § LDAP Polling Loop Pattern |
| RI Sponsorship API | `references/ri-sponsorship-api.md` |
| WFM action set pattern | § WFM Action Set Pattern |
| Full skeleton for a new action set | § Skeleton Templates |
| Validation checklist before delivery | § Validation Checklist |
| Common pitfalls (quick fixes) | § Common Pitfalls |
| Working mode detection (MCP vs file) | § Working Mode Detection |
| MCP workflow, JSON object model, XML ↔ JSON conversion | § MCP Workflow → `references/mcp-and-json.md` |

---

## XML Format Rules

- **No `<?xml ...?>` declaration** — omit entirely
- **No indentation or newlines between elements** — compact single-line output only
- **No XML comments** — use Connect `comment` actions instead
- **Double-hyphen (`--`) is forbidden** inside any `comment` action value
- `<arg>` values that are JS string literals must use `&quot;` for embedded double-quotes
  - e.g. `value="&quot;quiet&quot;"` not `value='"quiet"'`
- **Dot notation** for all variable/record field access: `record.fieldName`, `session.session`
- **Bracket notation only** when the key contains `@` or `-`: `record['@dn']`, `record['idauto-pwdPrivate']`
  - Never bracket plain identifiers: ❌ `record['idautoID']` → ✓ `record.idautoID`
- **Every `<action>` element must have `id` and `disabled` attributes:**
  - `id` — an uppercase UUID v4: `id="A1B2C3D4-E5F6-7890-ABCD-EF1234567890"`
  - `disabled="false"` — always present, always `false` unless intentionally disabling an action
  - Without these, the Connect editor renders actions as read-only and non-interactive
  - Example: `<action id="A1B2C3D4-E5F6-7890-ABCD-EF1234567890" name="setVariable" outputVar="" disabled="false">`

### Root Element

Inside a `.dssproject` archive (`actions/` folder) — bare `<actionDef>`:
```
<actionDef xmlns="urn:idauto.net:dss:actiondef" name="MyActionSet" returnsValue="false" description="...">
```

Standalone Connect import file — wrap in `<actionDefs>`:
```
<actionDefs xmlns="urn:idauto.net:dss:actiondef"><actionDef name="MyActionSet" ...>
```

**Never** include `builtIn` or `community` attributes on user-defined action sets.

---

## setVariable Expression Compiler Rules

The Connect expression compiler **parses `setVariable value=` as JavaScript before executing it**.
This is a compile-time check, not a runtime check. Expressions that would be valid JS at runtime
can still fail at compile time if the compiler cannot parse them syntactically.

### Characters that break compilation

The following characters are syntactically significant in JS and will cause a compile error if
they appear bare (unquoted) in a `value=` expression:

- `{` `}` — object literal delimiters
- `[` `]` — array literal delimiters
- `"` — string delimiter (inside a double-quoted XML attribute this also ends the attribute early)

**This means you cannot embed a JSON object or array as a raw literal value.** The compiler sees
`{"id":"foo","label":"bar"}` and tries to parse `{` as a block statement, fails immediately.

### String delimiter choice — prefer single quotes

Inside an XML `value="..."` attribute, single quotes are literal characters requiring no escaping.
Use single-quoted JS strings whenever a string constant contains double quotes:

```xml
<!-- WRONG: backslash escapes are literal in Rhino, not escape sequences -->
<arg name="value" value="&quot;\&quot;groups\&quot;:&quot; + someVar"/>
<!-- Runtime result: \"groups\":value  ← backslash is literal, invalid JSON -->

<!-- RIGHT: single-quote outer delimiter, &quot; for embedded double quotes -->
<arg name="value" value="'&quot;groups&quot;:' + someVar"/>
<!-- Runtime result: "groups":value  ← correct JSON key -->
```

### Serialising structured data — use JSON.stringify

Any time you need a variable to hold a JSON-serialised object or array, use `JSON.stringify` on
a **fresh single-object literal** in the same expression. Never accumulate properties on an object
across multiple `setVariable` calls and then pass the accumulated object to `JSON.stringify` — the
Rhino engine adds internal scope references to the object with each assignment, causing
`NativeJSON.str` to recurse infinitely and throw a `java.lang.StackOverflowError`.

```xml
<!-- WRONG: accumulate then stringify → StackOverflowError on large objects -->
<action name="setVariable"><arg name="name" value="obj"/><arg name="value" value="{}"/></action>
<action name="setVariable"><arg name="name" value="obj.a"/><arg name="value" value="&quot;foo&quot;"/></action>
<action name="setVariable"><arg name="name" value="obj.b"/><arg name="value" value="&quot;bar&quot;"/></action>
<action name="setVariable"><arg name="name" value="result"/><arg name="value" value="JSON.stringify(obj)"/></action>

<!-- RIGHT: stringify a fresh inline literal — safe, one call, no scope leak -->
<action name="setVariable">
  <arg name="name" value="result"/>
  <arg name="value" value="JSON.stringify({&quot;a&quot;:&quot;foo&quot;,&quot;b&quot;:&quot;bar&quot;})"/>
</action>
```

### Building large JSON strings — string concatenation pattern

When you need to produce a large JSON string (e.g. a manifest with 35 sections), build it by
concatenating pre-serialised string fragments rather than assembling one large object:

1. **Per-entry**: `JSON.stringify` on one small flat object per entry → store in an accumulator string
2. **Separator**: use `','` (single-quote delimiter, no escaping needed)
3. **Wrapper**: assemble with string concatenation using `[parts].join(',')` or `'{' + ... + '}'`

```xml
<!-- Step 1: init accumulator -->
<action name="setVariable"><arg name="name" value="sectionsStr"/><arg name="value" value="&quot;&quot;"/></action>

<!-- Step 2: first entry (no leading comma) -->
<action name="setVariable">
  <arg name="name" value="sectionsStr"/>
  <arg name="value" value="'&quot;sectionKey&quot;:' + JSON.stringify({&quot;label&quot;:&quot;My Label&quot;,&quot;rowCount&quot;:counts.myKey})"/>
</action>

<!-- Step 3: subsequent entries (prepend comma with single-quote string) -->
<action name="setVariable">
  <arg name="name" value="sectionsStr"/>
  <arg name="value" value="sectionsStr + ',' + '&quot;nextKey&quot;:' + JSON.stringify({&quot;label&quot;:&quot;Next Label&quot;,&quot;rowCount&quot;:counts.nextKey})"/>
</action>

<!-- Step 4: wrap in outer structure via array join -->
<action name="setVariable">
  <arg name="name" value="finalJSON"/>
  <arg name="value" value="'{' + [fieldA, fieldB, '&quot;sections&quot;:{' + sectionsStr + '}'].join(',') + '}'"/>
</action>
```

### Arrays and nested objects inside JSON.stringify

When a `JSON.stringify` call needs to include an array of string keys, write it inline as a
JS array literal — this is safe inside `JSON.stringify(...)` because it's not a bare expression:

```xml
<!-- Safe: array literal inside JSON.stringify -->
<action name="setVariable">
  <arg name="name" value="groupJSON"/>
  <arg name="value" value="JSON.stringify({&quot;id&quot;:&quot;myGroup&quot;,&quot;sections&quot;:[&quot;key1&quot;,&quot;key2&quot;,&quot;key3&quot;]})"/>
</action>
```

For multiple sibling objects (e.g. an array of groups), serialise each individually then
concatenate with `'['` / `']'` wrappers:

```xml
<action name="setVariable"><arg name="name" value="g1"/><arg name="value" value="JSON.stringify({&quot;id&quot;:&quot;group1&quot;,&quot;sections&quot;:[&quot;a&quot;,&quot;b&quot;]})"/></action>
<action name="setVariable"><arg name="name" value="g2"/><arg name="value" value="JSON.stringify({&quot;id&quot;:&quot;group2&quot;,&quot;sections&quot;:[&quot;c&quot;,&quot;d&quot;]})"/></action>
<action name="setVariable"><arg name="name" value="groupsArray"/><arg name="value" value="'[' + g1 + ',' + g2 + ']'"/></action>
```

### Summary table

| Situation | Correct approach |
|---|---|
| String constant containing `"` | Single-quote outer delimiter: `'&quot;key&quot;:value'` |
| Serialise a small flat object | `JSON.stringify({&quot;k&quot;:&quot;v&quot;,...})` inline |
| Serialise an object with many properties | Build one field at a time, join with `','`, wrap with `'{'` / `'}'` |
| Array of string keys | Inline `[&quot;k1&quot;,&quot;k2&quot;]` inside `JSON.stringify(...)` |
| Array of objects | Serialise each with `JSON.stringify`, then `'[' + a + ',' + b + ']'` |
| Accumulated object → `JSON.stringify` | **Forbidden** — StackOverflowError. Use the concatenation pattern. |

---

## setVariable vs copyRecord — Reference vs Deep Copy

`setVariable` does **not** copy an object or array. It creates a second variable name that
points to the **same object in memory**. Both names are aliases for the same data structure.

```xml
<!-- WRONG: this does NOT make a copy -->
<action name="copyRecord" outputVar="oldRecord"><arg name="record" value="connectOperator[0]"/></action>
<action name="setVariable"><arg name="name" value="newRecord"/><arg name="value" value="oldRecord"/></action>

<!-- oldRecord and newRecord are now the same object -->
<!-- Modifying newRecord also modifies oldRecord -->
<action name="setRecordFieldValue"><arg name="record" value="newRecord"/><arg name="field" value="&quot;someAttr&quot;"/><arg name="value" value="&quot;test&quot;"/></action>
<!-- FnHasRecordChanged will return false — they're the same object, nothing "changed" -->
```

Use `copyRecord` (for records) or `copyArray` (for arrays) to get a true independent copy:

```xml
<!-- RIGHT: copyRecord allocates a new object in memory -->
<action name="copyRecord" outputVar="oldRecord"><arg name="record" value="connectOperator[0]"/></action>
<action name="copyRecord" outputVar="newRecord"><arg name="record" value="oldRecord"/></action>

<!-- Now modifying newRecord does NOT affect oldRecord -->
<action name="setRecordFieldValue"><arg name="record" value="newRecord"/><arg name="field" value="&quot;someAttr&quot;"/><arg name="value" value="&quot;test&quot;"/></action>
<!-- FnHasRecordChanged will correctly detect the difference -->
```

### When each applies

| Situation | Use |
|---|---|
| Normalize `getLDAPRecords` results to always be an array | `copyArray outputVar="results"` with `array="results"` — handles 0, 1, or N records safely |
| Extract a single record from a results array | `copyRecord outputVar="record"` with `record="results[0]"` — never `setVariable` |
| Snapshot a record before processing to compare later | `copyRecord` |
| Snapshot an array before processing to compare later | `copyArray` |
| Iterate an array and remove items inside the loop | `copyArray` — iterate the copy, remove from original |
| Normalize a multi-valued LDAP attribute inline (e.g. in `forEach collection`) | `[].concat(record.member)` — use directly in the expression; no intermediate variable needed |
| Pass a record into logic you own and will not mutate | `setVariable` is fine |
| Any path where `setRecordFieldValue` / `setRecordFieldValues` will run on the copy | `copyRecord` |

### Normalizing getLDAPRecords results

Connect returns a single object (not an array) when exactly one record matches. Always run
`copyArray` immediately after `getLDAPRecords` to normalize:

```xml
<!-- In XML: -->
<action name="getLDAPRecords" outputVar="results">...</action>
<action name="copyArray" outputVar="results"><arg name="array" value="results"/></action>
<action name="if">
  <arg name="condition" value="results &amp;&amp; results.length &gt; 0"/>
  ...
</action>
```

Then extract a record with `copyRecord`, not `setVariable`:
```xml
<action name="copyRecord" outputVar="groupRecord"><arg name="record" value="results[0]"/></action>
```

### Normalizing multi-valued LDAP attributes inline

A multi-valued attribute (e.g. `member`, `objectClass`) comes back as a string when only one
value is present. Use `[].concat(attr)` inline — directly in the `forEach collection` arg or
in an expression. No intermediate variable needed:

```xml
<!-- Count: -->
<arg name="value" value="([].concat(groupRecord.member)).length"/>

<!-- Iterate: -->
<action name="forEach">
  <arg name="variable" value="memberDn"/>
  <arg name="collection" value="[].concat(groupRecord.member)"/>
  ...
</action>

<!-- Check existence before concat: -->
<action name="if">
  <arg name="condition" value="groupRecord.member"/>
  <arg name="then">
    <action name="forEach">
      <arg name="variable" value="memberDn"/>
      <arg name="collection" value="[].concat(groupRecord.member)"/>
      ...
    </action>
  </arg>
</action>
```

### Array mutation rule

When removing items from an array inside a `forEach`, always `copyArray` first, iterate the
copy, and call `removeArrayItem` with `indexOf` on the **original**. Mutating the array being
iterated breaks the loop.

```xml
<action name="copyArray" outputVar="array1Copy"><arg name="array" value="array1"/></action>
<action name="forEach">
  <arg name="collection" value="array1Copy"/>
  <arg name="variable" value="item"/>
  <arg name="do">
    <action name="if">
      <arg name="condition" value="/* condition to remove */"/>
      <arg name="then">
        <action name="removeArrayItem">
          <arg name="array" value="array1"/>
          <arg name="index" value="array1.indexOf(item)"/>
        </action>
      </arg>
    </action>
  </arg>
</action>
```

---

## Naming Scheme

### Action Set Prefixes

| Type | Prefix |
|---|---|
| Scheduled Job — sync/import/export | `Sync` |
| Scheduled Job — maintenance/events | `Manage` |
| Function — reusable library | `Fn` |
| REST Endpoint | `REST` |
| Report | `Report` |
| Utility | `Util` |
| Alternate Action (portal post-action hooks) | `AA` — do not rename |
| Dynamic List (portal dropdown population) | `Dynamic_` — do not rename |
| Project meta | `_` |

Rules:
- No underscores except `Dynamic_*` and `_*` names
- CamelCase throughout: `SyncADSToRI`, `ManageGroupMemberships`, `FnGetUser`

### Section Labels

- All section labels: `camelCase`
- **Never** name a section label the same as a Connect built-in action or reserved variable
  - `return` is forbidden → use `returnSuccess`, `returnRecord`, `returnResult`, etc.
  - `log`, `section`, `if`, `forEach` are also forbidden as section labels

### Description Template

```
[Type] - [Source] to [Target]: One-sentence description.
```

Examples:
- `Scheduled Job - ADS to RI: Imports user accounts from Active Directory into RapidIdentity.`
- `Function - RI: Looks up a user record by idautoID and returns it.`
- `Utility: Monitors connection errors and sends an alert when a downstream system is unreachable.`

---

## Action Set Structure

Every action set must follow this top-level section order — no bare top-level actions:

```
about                   (suppressTrace="true")
defineDefaultVariables  (suppressTrace="true")
[functional sections]   (suppressTrace="true" — all of them, no exceptions)
```

**All sections must have `suppressTrace="true"` — including inner/nested sections.**

---

## about Section

Required first section. Contains `comment` actions only, in this order:

```xml
<action name="section">
  <arg name="label" value="about"/>
  <arg name="suppressTrace" value="true"/>
  <arg name="do">
    <action name="comment"><arg name="comment" value="Last Modified By: Name"/></action>
    <action name="comment"><arg name="comment" value="Last Modified Date: YYYY-MM-DD HH:mm"/></action>
    <action name="comment"><arg name="comment" value="Purpose: One-sentence description."/></action>
    <action name="comment"><arg name="comment" value="Parameters:"/></action>
    <action name="comment"><arg name="comment" value="  paramName: Description."/></action>
    <action name="comment"><arg name="comment" value="  paramName [optional]: Description."/></action>
    <action name="comment"><arg name="comment" value="================== Change Log =================="/></action>
    <action name="comment"><arg name="comment" value="YYYY-MM-DD (Name): Initial version."/></action>
  </arg>
</action>
```

---

## defineDefaultVariables Section

Required second section. Must always contain at minimum:

```xml
<action name="section">
  <arg name="label" value="defineDefaultVariables"/>
  <arg name="suppressTrace" value="true"/>
  <arg name="do">
    <action name="getCurrentActionSetName" outputVar="actionSetName"/>
    <action name="setVariable">
      <arg name="name" value="logLevel"/>
      <arg name="value" value="logLevel || &quot;quiet&quot;"/>
    </action>
  </arg>
</action>
```

If counters are used, initialize them here:
```xml
<action name="setVariable"><arg name="name" value="counts"/><arg name="value" value="{processed:0,add:0,update:0,skip:0,error:0}"/></action>
```

---

## Logging — built-in `log` action

Use the native `log` action for all logging.

### Basic log call

```xml
<action name="log">
  <arg name="message" value="&quot;Processing: &quot; + record.cn"/>
  <arg name="level" value="&quot;INFO&quot;"/>
</action>
```

### log level values

`TRACE` | `DEBUG` | `INFO` | `WARN` | `ERROR` — always uppercase on the `log` action's `level` arg.

**Two distinct vocabularies — do not conflate them.** The `log` action's `level` arg uses the
uppercase severities above (`INFO`, `ERROR`, …). The `logLevel` *parameter* an action set accepts
to gate its own verbosity uses the lowercase words `quiet | normal | debug` (see below). They are
unrelated: `level` is the severity stamped on one log line; `logLevel` is the caller-controlled
threshold that decides whether a given line runs at all.

### logLevel gate pattern

When action sets accept a `logLevel` parameter, gate verbose log calls with an `if`:

```xml
<action name="if">
  <arg name="condition" value="logLevel === &quot;debug&quot;"/>
  <arg name="then">
    <action name="log">
      <arg name="message" value="&quot;Record dump: &quot; + JSON.stringify(record)"/>
      <arg name="level" value="&quot;DEBUG&quot;"/>
    </action>
  </arg>
</action>
```

Recommended `logLevel` convention:

| logLevel | What to log |
|---|---|
| `quiet` (default) | Errors and failures only |
| `normal` | Progress, counts, key state changes |
| `debug` | Everything including record dumps |

### Log color schema — `Global.connectLogColorSchema`

Always initialize `logColors` in `defineDefaultVariables` using `Object.assign` with defaults first, then `Global.connectLogColorSchema` layered on top. This ensures all keys are always present even if the Global is missing or only partially defined:

```xml
<action name="setVariable">
  <arg name="name" value="logColors"/>
  <arg name="value" value="Object.assign({changedData:&quot;chocolate&quot;,complete:&quot;teal&quot;,counts:&quot;black&quot;,data:&quot;blue&quot;,debug:&quot;purple&quot;,error:&quot;red&quot;,fail:&quot;darkred&quot;,info:&quot;royalBlue&quot;,logOnly:&quot;slateGray&quot;,processing:&quot;steelBlue&quot;,query:&quot;darkcyan&quot;,skipped:&quot;mediumpurple&quot;,sourceData:&quot;dimGray&quot;,success:&quot;green&quot;,targetData:&quot;darkslategray&quot;,test:&quot;darkorange&quot;,warn:&quot;goldenrod&quot;,whitespace:&quot;white&quot;},Global.connectLogColorSchema||{})"/>
</action>
```

Then reference colors as `logColors.keyName` on every `log` action's `color` arg:

| Key | Color | Use for |
|---|---|---|
| `changedData` | chocolate | Values that were changed |
| `complete` | teal | Final success / job complete |
| `counts` | black | Count summaries |
| `data` | blue | General data values |
| `debug` | purple | Debug dumps |
| `error` | red | Errors |
| `fail` | darkred | Fatal failures / abort |
| `info` | royalBlue | General informational progress |
| `logOnly` | slateGray | Suppressed writes (logOnly mode) |
| `processing` | steelBlue | In-progress work |
| `query` | darkcyan | LDAP/DB filter strings |
| `skipped` | mediumpurple | Skipped records |
| `sourceData` | dimGray | Raw source/input data |
| `success` | green | Successful operations |
| `targetData` | darkslategray | Target system data |
| `test` | darkorange | Test/dry-run output |
| `warn` | goldenrod | Warnings |
| `whitespace` | white | Visual spacers |

---

## Counting — variable-based counters

Use a plain object variable for counts. Never create individual top-level counter variables.

Initialize in `defineDefaultVariables`:
```xml
<action name="setVariable">
  <arg name="name" value="counts"/>
  <arg name="value" value="{processed:0,add:0,update:0,skip:0,error:0}"/>
</action>
```

Increment during processing:
```xml
<action name="setVariable">
  <arg name="name" value="counts.processed"/>
  <arg name="value" value="counts.processed + 1"/>
</action>
```

Log summary at the end:
```xml
<action name="log">
  <arg name="message" value="&quot;Counts -- processed: &quot; + counts.processed + &quot; | add: &quot; + counts.add + &quot; | update: &quot; + counts.update + &quot; | skip: &quot; + counts.skip + &quot; | error: &quot; + counts.error"/>
  <arg name="level" value="&quot;INFO&quot;"/>
</action>
```

---

## Control Flow

Connect provides built-in control-flow actions. None of them are JavaScript statements — they are
actions with nesting args, so the same `do`/`then`/`else` rules from § XML Format Rules apply.

### forEach

Loop arg names are **`variable`** (the loop-variable name) and **`collection`** (the array). Never
`item`/`items`. Normalize a possibly-single value with `[].concat(...)` directly in `collection`.

```xml
<action name="forEach">
  <arg name="variable" value="member"/>
  <arg name="collection" value="[].concat(groupRecord.member)"/>
  <arg name="do">
    <action name="log"><arg name="message" value="member"/></action>
  </arg>
</action>
```

### if / while / break

- `if` takes `condition` + `then` (+ optional `else`).
- `while` takes only `condition` + `do` — no `else`.
- `break` exits the nearest loop.

### Labeled loops and continue

A `forEach` (or `while`) may carry a `label`; `continue` and `break` can then target that label by
name. Use this when an inner condition should skip to the next iteration of a specific outer loop.

```xml
<action name="forEach">
  <arg name="label" value="FieldIterator"/>
  <arg name="variable" value="field"/>
  <arg name="collection" value="fields"/>
  <arg name="do">
    <action name="if">
      <arg name="condition" value="field == &quot;IGNORE&quot;"/>
      <arg name="then">
        <action name="continue"><arg name="label" value="FieldIterator"/></action>
      </arg>
      <arg name="else"/>
    </action>
    <!-- process field -->
  </arg>
</action>
```

---

## JavaScript & Engine Idioms

The Connect expression engine is at least ES6-capable. The following are confirmed working and are
the established idioms used in the ConnectLibrary examples.

### Arrow functions

Arrow functions evaluate inside expressions, including as callbacks to `filter`, `map`, etc.:

```xml
<arg name="value" value="arrayA.filter(x =&gt; !setB.has(x))"/>
<arg name="value" value="exArray.filter((value, index) =&gt; exArray.indexOf(value) === index)"/>
```

### Named functions via setVariable

Define a reusable function by assigning a **named function** to a variable with `setVariable`. The
function binds to that variable name and may recurse. This is the house idiom (e.g. `arrayDeepCopy`,
`recursiveArraySort` in `FnHasRecordChanged`). Multi-line bodies are allowed inside the `value`
attribute — the editor stores the newlines; the "single-line compact" XML rule governs element
structure, not attribute contents.

```xml
<action name="setVariable">
  <arg name="name" value="arrayDeepCopy"/>
  <arg name="value" value="function arrayDeepCopy(arr){ if(Array.isArray(arr)){ var c = arr.slice(0); for(var i=0;i&lt;c.length;i++){ c[i]=arrayDeepCopy(c[i]); } return c; } else { return arr; } }"/>
</action>
<action name="setVariable">
  <arg name="name" value="deep"/>
  <arg name="value" value="arrayDeepCopy(original)"/>
</action>
```

### Set — unique values and array differencing

`new Set(arr)` plus `.has()` is the clean way to compute membership differences — exactly what group
add/remove reconciliation needs:

```xml
<action name="setVariable"><arg name="name" value="setB"/><arg name="value" value="new Set(arrayB)"/></action>
<!-- items only in A (to add) -->
<action name="setVariable"><arg name="name" value="onlyInA"/><arg name="value" value="arrayA.filter(x =&gt; !setB.has(x))"/></action>
```

### Array idioms

| Need | Expression |
|---|---|
| Unique values | `arr.filter((v,i) =&gt; arr.indexOf(v) === i)` |
| Duplicate values | `arr.filter((v,i) =&gt; arr.indexOf(v) !== i)` |
| Unique as CSV string | `arr.filter((v,i) =&gt; arr.indexOf(v) === i).join(',')` |
| Merge two arrays | `arrayA.concat(arrayB)` |
| Normalize single value/record → array | `[].concat(value)` |
| Set difference (A not in B) | `new Set(b)` then `a.filter(x =&gt; !setB.has(x))` |

### JSON.stringify for logging

Logging an object as the sole `message` arg renders it readably, but **concatenating an object into
a string** (`"x: " + obj`) yields `[object Object]`. Use `JSON.stringify(obj)` — or
`JSON.stringify(obj, null, 4)` for indented multi-line output — when embedding an object in a
message:

```xml
<arg name="message" value="&quot;Record: &quot; + JSON.stringify(record, null, 4)"/>
```

(The StackOverflow caveat in § setVariable Expression Compiler Rules still applies: never pass an
object you accumulated across multiple `setVariable` calls to `JSON.stringify`.)

---

## Try/Catch

Connect has **no native try/catch action**. To trap a JavaScript error (e.g. a null-pointer or a
`JSON.parse` failure) instead of letting it abort the whole action set, define a JS function via
`setVariable` that wraps the risky operation in a real `try/catch`, then call it. This follows the
named-function idiom above. Use `catch(e)` (with the binding) for engine compatibility — prefer it
over the bare ES2019 `catch {}` form.

```xml
<!-- define once, e.g. in defineDefaultVariables -->
<action name="setVariable">
  <arg name="name" value="isValidJSON"/>
  <arg name="value" value="function isValidJSON(text){ try { JSON.parse(text); return true; } catch(e) { return false; } }"/>
</action>

<!-- call it; the failure path returns cleanly instead of crashing the action set -->
<action name="setVariable"><arg name="name" value="ok"/><arg name="value" value="isValidJSON(payload)"/></action>
<action name="if">
  <arg name="condition" value="!ok"/>
  <arg name="then">
    <action name="log">
      <arg name="message" value="&quot;Invalid JSON, handled without crashing: &quot; + payload"/>
      <arg name="level" value="&quot;ERROR&quot;"/>
      <arg name="color" value="logColors.error"/>
    </action>
  </arg>
  <arg name="else"/>
</action>
```

The `catch` block is also where you enhance diagnostics — log the offending input, or return a
sentinel like `"ERROR"` so the caller can branch on it (halt, send mail, etc.). Reference: the PSO
"Try Catch" page,
`https://idauto.atlassian.net/wiki/spaces/PSO/pages/3145465977/Try+Catch`.

---

## String & Record Built-in Actions

Connect ships built-in actions for common string and record operations. Prefer these (or the
equivalent JS methods) over hand-rolled logic.

### String actions

| Action | Args | Notes |
|---|---|---|
| `splitString` | `string`, `delimiter` | Returns an array. A value with no delimiter yields a single-item array. |
| `stringContains` | `string`, `pattern`, `ignoreCase` | `pattern` accepts a plain string **or a regex literal** (e.g. `/\d/`). |
| `stringEquals` | `string`, `pattern`, `ignoreCase` | With a regex `pattern` this is a *match*, not literal equality — e.g. `/^\d+$/` (all digits), an email regex, etc. |
| `stringRepeat` | `text`, `count` | Returns `text` repeated `count` times. |

JS string methods also work, e.g. fixed-width padding for IDs:

```xml
<arg name="value" value="exString.padStart(3,'0')"/>  <!-- '5' -> '005' -->
<arg name="value" value="exString.padEnd(3,'0')"/>    <!-- '5' -> '500' -->
```

### Record / array introspection actions

| Action | Args | Returns |
|---|---|---|
| `createRecord` | (none) | A new empty record (`outputVar`). |
| `setRecordFieldValue` | `record`, `field`, `value` | Sets one field. |
| `setRecordFieldValues` | `record`, `field`, `values` | Sets a multi-valued field from an array. |
| `getRecordFieldNames` | `record` | Array of field names. |
| `getRecordFieldValues` | `record`, `field` | Array of values for one field (handles multi-valued). |
| `arrayContains` | `array`, `value` | Boolean — membership test. |

---

## Function Mode Pattern

If an optional `session` parameter is provided, use it directly — never open/close connections.
Only open/close when no session is provided. Use the correct typed connection action for the
target system (see § Connections).

```xml
<action name="if">
  <arg name="condition" value="!session"/>
  <arg name="then">
    <!-- e.g. for RapidIdentity: -->
    <action name="openMetadirLDAPConnection" outputVar="session"/>
    <!-- e.g. for Active Directory: openADConnection (with bridgeInfo args) -->
    <!-- e.g. for Google: defineGoogleExtendedOAuthConnection (with Global args) -->
    <!-- e.g. for Portal: defineCloudPortalConnection -->
    <action name="setVariable">
      <arg name="name" value="closeSession"/>
      <arg name="value" value="true"/>
    </action>
  </arg>
</action>
```

Close at the end only when `closeSession` is true:
```xml
<action name="if">
  <arg name="condition" value="closeSession"/>
  <arg name="then">
    <action name="close">
      <arg name="closeable" value="session"/>
    </action>
  </arg>
</action>
```

---

## Connections

RapidIdentity Connect targets many systems, each with its own typed connection action. Always store
the result in `outputVar="session"`, check `!session` before proceeding, and source all credentials
from `Global.*` — never hardcode them.

| Target system | Connection action |
|---|---|
| Active Directory | `openADConnection` (needs `getIdBridgeConnectInfo` first) |
| RapidIdentity Metadirectory (OpenLDAP) | `openMetadirLDAPConnection` (no args) |
| RapidIdentity Portal | `defineCloudPortalConnection` (no args) |
| Google | `defineGoogleExtendedOAuthConnection` (+ generic `callGoogleAPI`) |
| Microsoft 365 | OAuth2 bearer token via `httpPOST` (no built-in action) |
| Database | `openDatabaseConnection` (needs a bridge + connection-string template) |
| AES encrypt/decrypt | AES Community Adapter actions (`GenerateAESKey`, `AESEncrypt`, `AESDecrypt`, …) |

All connection types close with `<action name="close"><arg name="closeable" value="session"/></action>`.

**Full per-system details — required args, Global keys, `getLDAPRecords` `baseDn`/`attributes`
selection, the AES sequence, failure handling, and closing — are in
`references/connections.md`. Read it before authoring any connection logic.**

---
## Global Properties

**Never hardcode** DNs, hostnames, paths, filenames, or credentials. Reference via:
- `Global.propertyKey` — project-level Globals
- `SharedGlobal.propertyKey` — cross-project SharedGlobals

Common patterns: base DNs, server hostnames, file paths, and API endpoints should all be
stored as Globals and referenced by key. The specific key names depend on project configuration
— check `CLAUDE.md` or the project's Globals for the canonical names.

---

## argDef Rules

Every parameter must have a `description` attribute. Types:
`string` | `boolean` | `number` | `object` | `array` | `enum:val1,val2,...`

```xml
<argDefs>
  <argDef name="logOnly"   type="boolean" optional="true"  description="Suppress all write operations when true."/>
  <argDef name="logLevel"  type="enum:quiet,normal,debug" optional="true" description="Controls logging verbosity."/>
  <argDef name="session"   type="object" optional="true" description="Existing directory session. Opens one if not provided."/>
</argDefs>
```

---

## Skeleton Templates

### Function action set

```xml
<actionDef xmlns="urn:idauto.net:dss:actiondef" name="FnMyFunction" returnsValue="true" description="Function - RI: One-sentence description."><argDefs><argDef name="session" type="object" optional="true" description="Existing directory session; opens one if omitted."/><argDef name="logOnly" type="boolean" optional="true" description="Suppress writes when true."/><argDef name="logLevel" type="enum:quiet,normal,debug" optional="true" description="Logging verbosity."/></argDefs><actions><action id="00000001-0000-0000-0000-000000000001" name="section" outputVar="" disabled="false"><arg name="label" value="about"/><arg name="suppressTrace" value="true"/><arg name="do"><action id="00000001-0000-0000-0000-000000000002" name="comment" outputVar="" disabled="false"><arg name="comment" value="Last Modified By: "/></action><action id="00000001-0000-0000-0000-000000000003" name="comment" outputVar="" disabled="false"><arg name="comment" value="Last Modified Date: YYYY-MM-DD"/></action><action id="00000001-0000-0000-0000-000000000004" name="comment" outputVar="" disabled="false"><arg name="comment" value="Purpose: One-sentence description."/></action><action id="00000001-0000-0000-0000-000000000005" name="comment" outputVar="" disabled="false"><arg name="comment" value="Parameters:"/></action><action id="00000001-0000-0000-0000-000000000006" name="comment" outputVar="" disabled="false"><arg name="comment" value="  session [optional]: Existing directory session."/></action><action id="00000001-0000-0000-0000-000000000007" name="comment" outputVar="" disabled="false"><arg name="comment" value="  logOnly [optional]: Suppress writes when true."/></action><action id="00000001-0000-0000-0000-000000000008" name="comment" outputVar="" disabled="false"><arg name="comment" value="  logLevel [optional]: quiet | normal | debug."/></action><action id="00000001-0000-0000-0000-000000000009" name="comment" outputVar="" disabled="false"><arg name="comment" value="================== Change Log =================="/></action><action id="00000001-0000-0000-0000-000000000010" name="comment" outputVar="" disabled="false"><arg name="comment" value="YYYY-MM-DD (Name): Initial version."/></action></arg></action><action id="00000002-0000-0000-0000-000000000001" name="section" outputVar="" disabled="false"><arg name="label" value="defineDefaultVariables"/><arg name="suppressTrace" value="true"/><arg name="do"><action id="00000002-0000-0000-0000-000000000002" name="getCurrentActionSetName" outputVar="actionSetName" disabled="false"/><action id="00000002-0000-0000-0000-000000000003" name="setVariable" outputVar="" disabled="false"><arg name="name" value="logLevel"/><arg name="value" value="logLevel || &quot;quiet&quot;"/></action><action id="00000002-0000-0000-0000-000000000004" name="setVariable" outputVar="" disabled="false"><arg name="name" value="logColors"/><arg name="value" value="Object.assign({changedData:&quot;chocolate&quot;,complete:&quot;teal&quot;,counts:&quot;black&quot;,data:&quot;blue&quot;,debug:&quot;purple&quot;,error:&quot;red&quot;,fail:&quot;darkred&quot;,info:&quot;royalBlue&quot;,logOnly:&quot;slateGray&quot;,processing:&quot;steelBlue&quot;,query:&quot;darkcyan&quot;,skipped:&quot;mediumpurple&quot;,sourceData:&quot;dimGray&quot;,success:&quot;green&quot;,targetData:&quot;darkslategray&quot;,test:&quot;darkorange&quot;,warn:&quot;goldenrod&quot;,whitespace:&quot;white&quot;},Global.connectLogColorSchema||{})"/></action><action id="00000002-0000-0000-0000-000000000005" name="setVariable" outputVar="" disabled="false"><arg name="name" value="counts"/><arg name="value" value="{processed:0,add:0,update:0,skip:0,error:0}"/></action></arg></action><action id="00000003-0000-0000-0000-000000000001" name="section" outputVar="" disabled="false"><arg name="label" value="openConnections"/><arg name="suppressTrace" value="true"/><arg name="do"><action id="00000003-0000-0000-0000-000000000002" name="comment" outputVar="" disabled="false"><arg name="comment" value="Function mode: use provided session or open one."/></action></arg></action><action id="00000004-0000-0000-0000-000000000001" name="section" outputVar="" disabled="false"><arg name="label" value="mainLogic"/><arg name="suppressTrace" value="true"/><arg name="do"><action id="00000004-0000-0000-0000-000000000002" name="comment" outputVar="" disabled="false"><arg name="comment" value="Core logic here."/></action></arg></action><action id="00000005-0000-0000-0000-000000000001" name="section" outputVar="" disabled="false"><arg name="label" value="closeConnections"/><arg name="suppressTrace" value="true"/><arg name="do"><action id="00000005-0000-0000-0000-000000000002" name="comment" outputVar="" disabled="false"><arg name="comment" value="Close only if we opened (check closeSession flag)."/></action></arg></action><action id="00000006-0000-0000-0000-000000000001" name="section" outputVar="" disabled="false"><arg name="label" value="outputCounts"/><arg name="suppressTrace" value="true"/><arg name="do"><action id="00000006-0000-0000-0000-000000000002" name="log" outputVar="" disabled="false"><arg name="message" value="&quot;Counts -- processed: &quot; + counts.processed + &quot; | add: &quot; + counts.add + &quot; | update: &quot; + counts.update + &quot; | skip: &quot; + counts.skip + &quot; | error: &quot; + counts.error"/><arg name="level" value="&quot;INFO&quot;"/></action></arg></action></actions></actionDef>
```

### Scheduled Job (Manage/Sync)

Same skeleton but `returnsValue="false"` and `description` uses the `Scheduled Job -` prefix.
Common additional parameters: `logOnly`, `logLevel`, `resetCookie`, `fullSync`.

---

## Validation Checklist

Before delivering any XML:

1. **`xmllint --noout file.xml`** — must pass with zero errors
2. **No `<?xml` declaration** at the top
3. **All sections** have `suppressTrace="true"`
4. **No bare top-level actions** — all actions inside a section
5. **No hardcoded DNs, hostnames, or credentials** — use Global/SharedGlobal references
6. **No `builtIn` or `community` attributes** on user-defined action sets
7. **No `--` sequences** inside comment action values
8. **Bracket notation only** for keys with `@` or `-`; plain identifiers use dot notation
9. **`&quot;` used** for embedded double-quotes in `<arg value="...">` attributes
10. **Tag balance** — count `<arg name="do">`, `<arg name="then">`, `<arg name="else">` opens
    vs `</arg>` closes; they must match
11. **All `argDef` elements** have a `description` attribute
12. **No bare object/array literals** in `setVariable value=` — use `JSON.stringify({...})` for
    structured data; never pass an accumulated multi-property object to `JSON.stringify`
13. **String constants containing `"` use single-quote delimiters** — `'&quot;key&quot;:value'`
    not `&quot;\\&quot;key\\&quot;:&quot;value&quot;` (backslash escapes are literal in Rhino)
14. **Every `<action>` has `id` (uppercase UUID) and `disabled="false"`** — missing these makes
    actions read-only and non-interactive in the Connect editor

---

## Common Pitfalls

| Pitfall | Fix |
|---|---|
| Section label `return` | Rename to `returnSuccess`, `returnRecord`, etc. |
| Hardcoded base DNs or hostnames | Use `Global.*` or `SharedGlobal.*` references |
| `builtIn="false"` on user action set | Remove the attribute entirely |
| `record['idautoID']` bracket notation | Change to `record.idautoID` |
| Individual counter variables (`addCount`, `updateCount`) | Use a single `counts` object |
| Opening a connection inside a function when a session was passed | Check `!session` first |
| Using generic `openConnection` for AD, RI, Google, Portal | Use the typed action: `openADConnection`, `openMetadirLDAPConnection`, `defineCloudPortalConnection`, `defineGoogleExtendedOAuthConnection` |
| Hardcoding Google OAuth args | Always source from `Global.googleDomain`, `Global.googleOAuthCredentialName`, `Global.googleOAuthScopes`, `Global.googleImpersonateUserId` |
| Hardcoding AD credentials | Always use `Global.adUserUPN` and `Global.adPwd` |
| Microsoft 365 — using a built-in connection action | No built-in exists; use `httpPOST` to obtain a bearer token from the OAuth2 endpoint |
| Double-hyphen in a comment | Rephrase to avoid `--` |
| XML written with pretty-print indentation | Flatten to single-line compact output |
| Bare `{...}` or `[...]` literal in `setVariable value=` | Wrap in `JSON.stringify({...})` — bare object/array literals fail at compile time |
| Accumulated object passed to `JSON.stringify` | Use the string concatenation pattern — each entry gets its own inline `JSON.stringify` call |
| `"\\"key\\"":` style string (backslash-escaped quotes) | Use single-quote outer delimiter instead: `'"key":'` — backslash in Rhino is literal |
| `setVariable` to copy a record before mutating it | Use `copyRecord` — `setVariable` creates an alias, not a copy; both names point to the same object |
| `setVariable` to copy an array before removing items in a loop | Use `copyArray` — same alias problem; mutating the iterated array breaks `forEach` |
| `FnHasRecordChanged` returns false when changes were made | The "old" and "new" records are the same object due to `setVariable` aliasing; use `copyRecord` for the snapshot |
| `<action>` missing `id` or `disabled` | Actions render read-only in Connect editor — every action needs `id="UPPERCASE-UUID"` and `disabled="false"` |
| `forEach` with `item` / `items` args | Wrong arg names — use `variable` (loop var name) and `collection` (the array) |
| Closing LDAP with `closeConnection` + `ldapConnection` | Wrong action — use `<action name="close"><arg name="closeable" value="session"/></action>` for all connection types |
| `getLDAPRecords` with `attributes` value `[]` | Bare array literal fails the expression compiler — use `"*,+"` (all attrs), `"*"` (standard only), or `"attr1,attr2"` (specific) |
| `Array.isArray(ldapResults)` to check query results | Connect returns a single object when only one record matches — `Array.isArray` returns `false`. Always run `copyArray outputVar="results"` immediately after `getLDAPRecords`, then check `results.length > 0` |
| `setVariable` to extract a record from results array (`results[0]`) | Use `copyRecord outputVar="record"` with `record="results[0]"` — `setVariable` creates an alias, not a copy |
| `Array.isArray(record.member)` or pre-building a `members` variable | A single-valued attribute is a string, not an array. Use `[].concat(record.member)` inline in `forEach collection` or expressions — no intermediate variable needed. Gate with `if (record.member)` first |
| Querying groups with `Global.metaBaseDN` | Use `Global.metaGroupBaseDN` for group-only queries, `Global.metaEmployeeBaseDN` for user-only queries, `Global.metaBaseDN` only when searching across both |
| Nesting arg (`do`/`then`/`else`) has a `"value"` field | Remove `value` entirely from nesting args — its presence (even `""`) tells Connect the arg is a scalar, causing `Property must be a list of actions`. Correct form: `{"name": "do", "actions": [...]}` |

---

## HTTP Actions — REST API Patterns

### `httpPOST` — JSON body requirement

The `data` argument of `httpPOST` is sent as-is as the request body. A raw JS object will
serialize as `[object Object]`, causing a 400 parse error. **Always wrap the body in `toJSON()`**
when the endpoint expects JSON:

```xml
<action name="setVariable">
  <arg name="name" value="body"/>
  <arg name="value" value="toJSON({field1: value1, field2: value2})"/>
</action>
<action name="httpPOST" outputVar="apiResponse">
  <arg name="url" value="sessionPortal.url + &quot;api/rest/v2/someEndpoint&quot;"/>
  <arg name="headers" value="{&quot;Accept&quot;: &quot;application/json&quot;, &quot;Content-Type&quot;: &quot;application/json&quot;, &quot;Authorization&quot;: &quot;Bearer &quot; + sessionPortal.token}"/>
  <arg name="data" value="body"/>
</action>
```

### Date formatting for REST APIs

Workflow `DATE_TIME` form fields emit a full ISO 8601 timestamp (e.g. `2026-05-01T01:24:14.000Z`).
Most RI REST APIs expect plain `YYYY-MM-DD`. Use `.substring(0,10)` to trim:

```xml
<arg name="value" value="toJSON({expirationDate: (dateField ? dateField.substring(0,10) : null)})"/>
```

### HTTP status code handling

Always check `statusCode` explicitly. Key RI Sponsorship API codes:

| Code | Meaning |
|---|---|
| `200` | Success |
| `400` | Bad request — malformed body or invalid field value |
| `409` | Conflict — duplicate detected (when `checkForDuplicates: true`) |

---

## `delay` Action — Timed Pauses

Use the built-in `delay` action to pause execution. Required in polling loops where a downstream
system may not have committed a record yet.

```xml
<action name="delay">
  <arg name="seconds" value="1"/>
</action>
```

---

## LDAP Polling Loop Pattern

Use this pattern when an LDAP record may not yet exist after an upstream write (e.g. after a
REST API creates a sponsored account, the metadirectory sync has not completed yet).

```xml
<!-- Init loop state in defineDefaultVariables or before the loop -->
<action name="setVariable"><arg name="name" value="lookupAttempt"/><arg name="value" value="0"/></action>
<action name="setVariable"><arg name="name" value="lookupMaxAttempts"/><arg name="value" value="15"/></action>
<action name="setVariable"><arg name="name" value="foundRecord"/><arg name="value" value="null"/></action>

<action name="while">
  <arg name="condition" value="!foundRecord &amp;&amp; lookupAttempt &lt; lookupMaxAttempts"/>
  <arg name="do">
    <action name="setVariable"><arg name="name" value="lookupAttempt"/><arg name="value" value="lookupAttempt + 1"/></action>
    <action name="getLDAPRecords" outputVar="ldapResults">
      <arg name="ldapConnection" value="sessionRI"/>
      <arg name="baseDn" value="Global.metaEmployeeBaseDN"/>
      <arg name="scope" value="&quot;sub&quot;"/>
      <arg name="filter" value="&quot;(idautoPersonUserNameMV=&quot; + username + &quot;)&quot;"/>
      <arg name="attributes" value="&quot;idautoPersonClaimCode,mail,idautoPersonUserNameMV&quot;"/>
      <arg name="maxResults" value="1"/>
    </action>
    <action name="copyArray" outputVar="ldapResults"><arg name="array" value="ldapResults"/></action>
    <action name="if">
      <arg name="condition" value="ldapResults &amp;&amp; ldapResults.length &gt; 0"/>
      <arg name="then">
        <action name="copyRecord" outputVar="foundRecord"><arg name="record" value="ldapResults[0]"/></action>
        <action name="break"/>
      </arg>
      <arg name="else">
        <action name="delay"><arg name="seconds" value="1"/></action>
      </arg>
    </action>
  </arg>
</action>

<!-- Warn if record never found after all attempts -->
<action name="if">
  <arg name="condition" value="!foundRecord"/>
  <arg name="then">
    <action name="log">
      <arg name="message" value="actionSetName + &quot;: lookupRecord -- record not found after &quot; + lookupMaxAttempts + &quot; attempts.&quot;"/>
      <arg name="level" value="&quot;WARN&quot;"/>
      <arg name="color" value="logColors.warn"/>
    </action>
  </arg>
</action>
```

**Rules:**
- Always initialize `foundRecord = null` before the loop
- Run `copyArray` on the results immediately after `getLDAPRecords` to normalize the single-vs-array
  case, then test `ldapResults.length > 0` — never `Array.isArray` (a single match comes back as an object)
- Extract the record with `copyRecord`, never `setVariable` (which would alias, not copy)
- Use `break` inside the `then` branch once the record is found — do not continue polling
- Put `delay(1)` in the `else` branch only, so there is no delay on the successful attempt
- Log a `WARN` (not `ERROR`) if the loop exhausts — the caller decides how to handle a null result
- Cap at 15 attempts for a 1-second delay (15 seconds max wait); adjust for longer sync windows

---

## RI Sponsorship API

The RI Sponsorship API creates and manages sponsored accounts. It has several use-case-specific
details: the create-account request body shape, resolving custom attribute UUIDs from
`bootstrapInfo`, `409` duplicate handling, and the relevant endpoints.

**See `references/ri-sponsorship-api.md` for endpoints, the request body, and the custom-attribute
UUID resolution pattern.**

---
## WFM Action Set Pattern

Action sets that back portal workflows use the `WFM` prefix and follow a specific pattern:

- **Always return** `JSON.stringify(results)` from every exit path — the workflow reads `%{dss.fieldName}`
- **Accept `validateOnly`** — when true, validate all inputs and return without writing
- **Accept `logOnly`** — when true, skip all writes and return a logOnly message
- **Initialize a `results` record** in `defineDefaultVariables` with `success=false` and all output fields as empty strings
- **Every return path** must set `results.message` before returning

Standard results fields for a WFM action set:

```xml
<action name="createRecord" outputVar="results"/>
<action name="setRecordFieldValue"><arg name="record" value="results"/><arg name="field" value="&quot;success&quot;"/><arg name="value" value="false"/></action>
<action name="setRecordFieldValue"><arg name="record" value="results"/><arg name="field" value="&quot;message&quot;"/><arg name="value" value="&quot;&quot;"/></action>
```

Workflow variable references from DSS output: `%{dss.success}`, `%{dss.message}`, `%{dss.fieldName}`.

---

## Reference Files

Always-on lookups:
- `references/connect-builtin-actions.json` — Full catalog of all built-in Connect action definitions (name, argDefs, description, returnsValue). **Consult this when you need to verify an action's exact parameter names, types, or whether it returns a value.** Use targeted reads or grep — do not load the entire file into context at once. (The count grows as the platform adds actions, so don't rely on a fixed number.)
- `references/openldap-schema.md` — Full RI OpenLDAP schema: all `idautoPerson` and `idautoGroup` attributes with cardinality (single/multi) and type. **Always consult this before referencing any LDAP attribute** — it determines whether `[].concat()` is needed and which `baseDn` Global to use.

Load-when-relevant (these were split out of SKILL.md to keep it lean):
- `references/connections.md` — Per-system connection details: AD, RI Metadirectory (incl. `getLDAPRecords` `baseDn`/`attributes` selection), Portal, Google + `callGoogleAPI`, Microsoft 365, Database, the AES Community Adapter encrypt/decrypt sequence, failure handling, and closing. **Read before authoring any connection logic.**
- `references/mcp-and-json.md` — MCP workflow (read/explore, design, push, delete), the JSON object model (file JSON vs API JSON, argDef/action/arg shapes), and field-by-field XML ↔ JSON conversion. **Read when working against a live instance via the MCP or converting formats.**
- `references/ri-sponsorship-api.md` — RI Sponsorship API endpoints, create-account request body, and custom-attribute UUID resolution. **Read only for sponsored-account work.**

Load the always-on files when you need a specific action's parameters or an LDAP attribute; load the
others only for the matching task. The SKILL.md body covers everything needed for day-to-day authoring.

---

## MCP Workflow

The `RapidIdentity MCP Server:*` tools cover discovery and metadata: `get-connect-projects`,
`get-connect-actions` (list/metadata), `get-connect-action` (single, top-level only — see the MCP
capability caveat in § Working Mode Detection), `save-connect-action`, and `delete-connect-action`
(always confirm before deleting). Author real logic as XML in file mode.

**Full MCP workflow (read/explore, the JSON object model, and field-by-field XML ↔ JSON conversion
rules) is in `references/mcp-and-json.md`.**

---
