---
name: rapididentity-role-mining
description: >
  Use this skill to perform role mining on a RapidIdentity group — the process of analyzing
  static group members to discover shared attributes that could power a dynamic group filter.
  Trigger this skill whenever the user mentions role mining, wants to convert a static group
  to dynamic, wants to understand what attributes define a group's membership, or asks to
  analyze a RapidIdentity group. Requires the mcp-rapidid MCP server to be connected.
compatibility: Requires mcp-rapidid MCP server
---

# RapidIdentity Role Mining

Role mining is the process of examining who is in a static group and identifying shared
attributes that could replace hand-maintained membership with an automatic dynamic filter.
The goal is to produce an LDAP filter (like `(&(idautoPersonExt9=Faculty)(idautoPersonExt5=A))`)
that correctly captures all current members — and would continue to do so as new people join.

## Step 1: Identify the Target Group

Ask the user: **"Which RapidIdentity group would you like to do role mining on?"**

If the user is unsure or says they don't know, use `mcp-rapidid:search-groups` with `criteria: "*"` to retrieve all available groups. Present the results as a list and ask the user to pick one. For example:

> "Here are the groups available in RapidIdentity — which one would you like to analyze?"

Once the user identifies a group (either by name or from the list), proceed to Step 2.

## Step 2: Search for the Group

Use `mcp-rapidid:search-groups` with the group name as the criteria.

- If an exact match is found, confirm it with the user and note the group's `id` and current `dynamicMemberFilter` (if any).
- If multiple groups are returned, list them and ask the user which one they mean.
- If no results are returned, try a broader or partial search term and inform the user.

Note: Even if the group already has a dynamic filter, role mining can still be useful to
validate the filter or discover a richer/more accurate one.

## Step 3: Retrieve Group Members

Use `mcp-rapidid:get-group-members` with:
- `groupId`: the group's `id` from Step 2
- `pageSize`: 1000 (always use this default)
- `pagingSessionId`: `""` (empty string on first call)

Extract all member IDs from the `calculatedMembership` array. Each entry is a DN like
`idautoID=<uuid>,ou=Accounts,dc=meta` — parse out just the UUID portion.

If `pagingSessionId` is returned non-empty in the response, there are more pages — call
again with that value until all members are retrieved.

Inform the user how many members were found before proceeding.

## Step 4: Select a Delegation

Use `mcp-rapidid:get-my-delegations` to retrieve available delegations.

Present them to the user in a table showing Name and Description, then ask:
**"Which delegation should I use to look up these group members?"**

Choose the most likely candidate to suggest (e.g., "Staff" for employee groups,
"Students" for student groups) but always let the user decide.

## Step 5: Fetch All Member Profiles (Bulk LDAP Query)

Use `mcp-rapidid:get-user-info-in-delegation` with:
- `delegationId`: the chosen delegation's `id`
- `filter`: a bulk LDAP OR filter combining all member IDs

### LDAP Bulk Filter Syntax

Wrap all member conditions in a single `(|...)` OR clause:

```
(|(idautoID=<id1>)(idautoID=<id2>)(idautoID=<id3>)...)
```

**Always use a single bulk call** rather than fetching members one at a time. This is
significantly more efficient and avoids unnecessary API calls.

If the response returns fewer profiles than expected, some members may be outside the
chosen delegation's scope — note this to the user and offer to try a different delegation.

## Step 6: Analyze Similarities and Suggest a Dynamic Filter

Examine the returned profile attributes for every member. Look for attributes that are:

1. **Identical across all members** — strong candidates for the filter
2. **Present on all members** (even if values differ) — useful for understanding the group
3. **Unique to this group** vs. common across all users — helps distinguish signal from noise

### Key attributes to examine

These are the official RapidIdentity Cloud schema attributes most commonly used for dynamic
group filters. Prefer these over custom `idautoPersonExt*` attributes where possible, as
they are stable and semantically meaningful.

**Role / Type Attributes**

| Attribute Name | Friendly Name | Notes |
|---|---|---|
| `employeeType` | Role / Account Type | Core role: *staff, student, teacher, sponsored, parent* — RapidIdentity calls this "Account Type" |
| `idautoPersonEmployeeTypes` | Employee Types | More granular types beyond `employeeType`, e.g. *Teacher, Admin, Para* — multi-valued, commonly used for dynamic membership |
| `idautoPersonAffiliation` | Primary Affiliation | e.g. Faculty, Staff, Emeritus, Retiree, Student Enrolled — single-valued primary affiliation |
| `idautoPersonAffiliations` | Affiliations | Multi-valued; all affiliations associated with the person |

**Status Attributes**

| Attribute Name | Friendly Name | Notes |
|---|---|---|
| `idautoDisabled` | Account Disabled | `TRUE` = disabled; absence or `FALSE` = active |
| `idautoPersonSourceStatus` | Source System Status | Arbitrary status from source system (e.g. HR). Common value: `A` = Active |
| `idautoPersonStatusOverride` | Override Source Status | `TRUE` = status is locked from auto-updates |

**Location Attributes**

| Attribute Name | Friendly Name | Notes |
|---|---|---|
| `idautoPersonLocName` | Primary Location | Single-valued primary location name — good for campus/site-scoped groups |
| `idautoPersonLocNames` | Locations | Multi-valued; all locations — use when members may have multiple sites |
| `idautoPersonLocCode` | Primary Location Code | Code-based equivalent of Primary Location — preferred for filter precision |
| `idautoPersonLocCodes` | Location Codes | Multi-valued location codes |

**Department Attributes**

| Attribute Name | Friendly Name | Notes |
|---|---|---|
| `idautoPersonDeptDescr` | Department | Primary department description — single-valued |
| `idautoPersonDeptDescrs` | Departments | Multi-valued department descriptions |
| `idautoPersonDeptCode` | Primary Department Code | More stable than name — preferred for filters |
| `idautoPersonDeptCodes` | Department Codes | Multi-valued department codes |

**Job Attributes**

| Attribute Name | Friendly Name | Notes |
|---|---|---|
| `idautoPersonJobTitle` | Job Title | Primary job title — single-valued |
| `idautoPersonJobCode` | Job Code | Primary job code — more stable than title for filters |
| `idautoPersonJobCodes` | Job Codes | Multi-valued job codes |

**Education Attributes (K-12 / Higher Ed)**

| Attribute Name | Friendly Name | Notes |
|---|---|---|
| `idautoPersonGradeLevel` | Grade Level | Student grade level — multi-valued |
| `idautoPersonSchoolNames` | School Names | Schools associated with the person |
| `idautoPersonSchoolCodes` | School Codes | Code-based school membership — preferred for filters |

**Custom / Extensible Attributes**

| Attribute Name | Friendly Name | Notes |
|---|---|---|
| `idautoPersonExt1`–`idautoPersonExt25` | Custom Attribute 1–25 | Customer-defined; meaning varies per deployment. When observed in filters, note the value and ask the user what the attribute represents in their environment. |
| `idautoPersonExtBool1`–`idautoPersonExtBool5` | Custom Boolean 1–5 | Boolean flags; customer-defined meaning |

> **Note on `idautoPersonExt*` attributes:** These are generic custom attributes whose
> meaning is defined per-deployment. If a group's existing dynamic filter uses one of
> these (e.g. `idautoPersonExt5=A` or `idautoPersonExt9=Faculty`), ask the user what
> those attributes represent in their environment before including them in a suggested filter.

### Building the filter

- Use `(&(...)(...)...)` (AND) to combine multiple required conditions
- Use `(|(...)(...)...)` (OR) when members share one of several valid values
- Combine with `(&(condition1)(|(val1)(val2)))` for mixed logic

### Output format

Present your findings in three sections:

**1. Shared Attributes (filter candidates)**
List every attribute that is 100% consistent across all members, along with its value.

**2. Variable Attributes (not suitable for filter)**
List attributes that differ across members — these explain variation within the group
but shouldn't be part of the filter.

**3. Suggested Dynamic Filter**
Propose an LDAP filter using the shared attributes. Explain what each condition means
in plain language. If the group already had a dynamic filter, compare your suggestion
to the existing one and note any differences or improvements.

### Example output

```
Suggested filter: (&(idautoPersonExt5=A)(idautoPersonExt9=Faculty))

Plain English: Users where HRMS Status is Active AND Job Type is Faculty
```

If the data is inconclusive (too few members, too much variation), say so and suggest
what additional information or a larger sample might help.
