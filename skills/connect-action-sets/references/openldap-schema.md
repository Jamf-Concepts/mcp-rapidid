# RI OpenLDAP Schema — idautoPerson & idautoGroup Attributes

Source: `RICOpenLDAPSchema.json` (live tenant subschema dump)

**Syntax key:**
- `string` — DirectoryString (1.3.6.1.4.1.1466.115.121.1.15)
- `dn` — Distinguished Name (1.3.6.1.4.1.1466.115.121.1.12)
- `boolean` — Boolean (1.3.6.1.4.1.1466.115.121.1.7)
- `datetime` — GeneralizedTime (1.3.6.1.4.1.1466.115.121.1.24)
- `integer` — Integer (1.3.6.1.4.1.1466.115.121.1.27)
- `octet` — OctetString / binary (1.3.6.1.4.1.1466.115.121.1.40)
- `uuid` — UUID (1.3.6.1.1.16.1)

**Cardinality:** `single` = SINGLE-VALUE, `multi` = multi-valued

---

## Shared / Top-level

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `idautoID` | single | uuid | Primary key / RDN for all idauto entries |
| `idautoDisabled` | single | boolean | `TRUE` = disabled; absent = enabled |
| `idautoSchemaVersion` | single | string | |
| `idautoChallengeSet` | multi | string | Challenge/response sets |
| `idautoChallengeSetTimestamp` | single | datetime | |
| `idauto-pwdPrivate` | single | octet | RSA-encrypted copy of userPassword |
| `idauto-pwdPrivateTS` | single | datetime | Timestamp when idauto-pwdPrivate was last set |

---

## idautoPerson Attributes

**Required:** `idautoID`, `idautoPersonUserNameMV`

### Identity & Names

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `idautoPersonUserNameMV` | multi | string | All usernames (primary + alternates). Used for authentication. |
| `idautoPersonSAMAccountName` | single | string | Primary sAMAccountName synced from/to AD |
| `idautoPersonPrevSAMAccountNames` | multi | string | Username history — prevents reuse |
| `idautoPersonRenameUsername` | multi | string | Staged new username pending rename |
| `idautoPersonRenameFlagDate` | single | string | Date rename was triggered |
| `idautoPersonRenameOverride` | single | boolean | `TRUE` = block automated renames |
| `idautoPersonMiddleName` | single | string | |
| `idautoPersonPreferredName` | single | string | Preferred first name |
| `idautoPersonPreferredLastName` | single | string | Preferred last name |
| `idautoPersonPronouns` | multi | string | |
| `idautoPersonGender` | single | string | |
| `idautoPersonBirthdate` | single | string | |
| `idautoPersonPhotoURL` | single | string | |
| `idautoPersonProfileUrl` | single | string | |

### IDs & External Keys

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `idautoPersonDistrictID` | single | string | District-assigned identifier |
| `idautoPersonStuID` | single | string | Student ID |
| `idautoPersonHRID` | single | string | HR system ID |
| `idautoPersonPayrollID` | single | string | Payroll ID |
| `idautoPersonNationalID` | single | string | National/government ID |
| `idautoPersonStateID` | single | string | State-assigned ID |
| `idautoPersonSchoolID` | single | string | Primary school ID |
| `idautoPersonManagerID` | single | string | Manager's idautoPersonDistrictID |
| `idautoPersonSystem1ID` | single | string | External system 1 ID |
| `idautoPersonSystem2ID` | single | string | External system 2 ID |
| `idautoPersonSystem3ID` | single | string | External system 3 ID |
| `idautoPersonSystem4ID` | single | string | External system 4 ID |
| `idautoPersonSystem5ID` | single | string | External system 5 ID |

### Dates & Lifecycle

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `idautoPersonStartDate` | single | datetime | Student activation date |
| `idautoPersonEndDate` | single | datetime | Student deactivation date |
| `idautoPersonEnrollDate` | single | datetime | Enrollment date |
| `idautoPersonActivationDate` | single | string | |
| `idautoPersonTermDate` | single | string | Termination date |
| `idautoPersonGraduationDate` | single | string | |
| `idautoPersonStaffStartDate` | single | datetime | Staff activation date |
| `idautoPersonStaffEndDate` | single | datetime | Staff deactivation date |
| `idautoPersonStaffLastDateWorked` | single | datetime | |
| `idautoPersonStaffAccessTermDate` | single | datetime | |
| `idautoPersonContractStartDate` | single | datetime | Sponsored/contractor activation |
| `idautoPersonContractEndDate` | single | datetime | Sponsored/contractor deactivation |
| `idautoPersonContractLastDateWorked` | single | datetime | |
| `idautoPersonContractAccessTermDate` | single | datetime | |
| `idautoPersonAllAccessTermDate` | single | datetime | |
| `idautoPersonSafeIdCompromisedDate` | single | datetime | |

### Status & Overrides

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `idautoPersonSourceStatus` | single | string | Status value from source system |
| `idautoPersonSponsoredAccountStatus` | single | string | Sponsorship status |
| `idautoPersonStatusOverride` | single | boolean | `TRUE` = block all automated status changes |
| `idautoPersonStatusOverrideExpiration` | single | datetime | When override expires |
| `idautoPersonStatusOverrideReason` | single | string | |
| `idautoPersonPasswordSet` | single | boolean | |
| `idautoPersonClaimCode` | single | string | Account claim code |
| `idautoPersonClaimFlag` | single | boolean | |

### Role & Employment

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `idautoPersonAffiliation` | single | string | Primary affiliation |
| `idautoPersonAffiliations` | multi | string | All affiliations |
| `idautoPersonEmployeeTypes` | multi | string | All employee types (use `[].concat()` when iterating) |
| `idautoPersonJobCode` | single | string | Primary job code |
| `idautoPersonJobCodes` | multi | string | |
| `idautoPersonJobTitle` | multi | string | |
| `idautoPersonJobTitles` | multi | string | |
| `idautoPersonDeptCode` | single | string | Primary department code |
| `idautoPersonDeptCodes` | multi | string | |
| `idautoPersonDeptDescr` | single | string | Primary department description |
| `idautoPersonDeptDescrs` | multi | string | |
| `idautoPersonActivityCodes` | multi | string | |
| `idautoPersonManagedOrgs` | multi | string | Orgs this person manages |

### Location & School

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `idautoPersonLocCode` | single | string | Primary location code |
| `idautoPersonLocCodes` | multi | string | |
| `idautoPersonLocName` | single | string | Primary location name |
| `idautoPersonLocNames` | multi | string | |
| `idautoPersonSchoolCodes` | multi | string | All school codes |
| `idautoPersonSchoolNames` | multi | string | |
| `idautoPersonGradeLevel` | multi | string | |
| `idautoPersonCourseCodes` | multi | string | |
| `idautoPersonCourseIDs` | multi | string | |

### Contact

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `idautoPersonEmailAddresses` | multi | string | Additional email addresses |
| `idautoPersonHomeEmail` | single | string | |
| `idautoPersonHomePhone` | single | string | |
| `idautoPersonOfficePhone` | single | string | |
| `idautoPersonPhoneExtension` | single | string | |
| `idautoPersonStreetAddress` | multi | string | |
| `idautoPersonWorkStreetAddress` | multi | string | |
| `idautoPersonWorkCity` | single | string | |
| `idautoPersonWorkState` | single | string | |
| `idautoPersonWorkPostalCode` | single | string | |
| `idautoPersonWorkCountry` | single | string | |
| `idautoPersonCountry` | multi | string | |
| `idautoPersonADProfilePath` | single | string | |
| `idautoPersonPreferredLanguage` | single | string | |

### Relationships

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `idautoPersonStudents` | multi | dn | DNs of students associated with this person |
| `idautoPersonTeachers` | multi | dn | DNs of teachers associated with this person |

### Provisioning Targets

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `idautoPersonToSystem1` | single | boolean | Provisioned to system 1 |
| `idautoPersonToSystem2` | single | boolean | |
| `idautoPersonToSystem3` | single | boolean | |
| `idautoPersonToSystem4` | single | boolean | |
| `idautoPersonToSystem5` | single | boolean | |

### App Roles

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `idautoPersonAppRoleFriendlyNames` | multi | string | Human-readable role names |
| `idautoPersonAppRoles1` through `idautoPersonAppRoles10` | multi | string | App role values per system slot |

### Extension Attributes

| Attribute | Cardinality | Type |
|---|---|---|
| `idautoPersonExt1` through `idautoPersonExt25` | multi | string |
| `idautoPersonExtBool1` through `idautoPersonExtBool5` | single | string |

---

## idautoGroup Attributes

**Required:** `idautoID` (plus `member` from parent `groupOfNames` — may be empty DN `""`)

### Membership

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `member` | multi | dn | Static member DNs (inherited from groupOfNames). Use `[].concat(record.member)` when iterating — may be string when single value. |
| `idautoGroupStaticIncludes` | multi | dn | Additional static include DNs |
| `idautoGroupStaticExcludes` | multi | dn | Static exclude DNs |

### Dynamic Filters

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `idautoGroupIncludeFilter` | single | string | LDAP filter for dynamic include |
| `idautoGroupIncludeBaseDN` | single | dn | Base DN for include filter search |
| `idautoGroupExcludeFilter` | single | string | LDAP filter for dynamic exclude |
| `idautoGroupExcludeBaseDN` | single | dn | Base DN for exclude filter search |

### Ownership & Contact

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `idautoGroupOwners` | multi | dn | Owner DNs |
| `idautoGroupCoOwners` | multi | dn | Co-owner DNs |
| `idautoGroupCoOwnerEditable` | single | boolean | Whether co-owners can edit membership |
| `idautoGroupEmailAddress` | single | string | Group email address |
| `idautoGroupEmailAliases` | multi | string | Additional email aliases |

### Sync & Provisioning

| Attribute | Cardinality | Type | Notes |
|---|---|---|---|
| `idautoGroupLastSynced` | single | datetime | Last sync timestamp |
| `idautoGroupSyncInterval` | single | integer | Sync interval in minutes |
| `idautoGroupToSystem1` through `idautoGroupToSystem10` | single | boolean | Provisioned to system slot |

### Extension Attributes

| Attribute | Cardinality | Type |
|---|---|---|
| `idautoGroupExt1` through `idautoGroupExt5` | multi | string |
