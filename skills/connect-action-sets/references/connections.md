# RapidIdentity Connect — Connection Actions

Typed connection actions for each target system. Always store the result in `outputVar="session"`, check `!session` before proceeding, and source all credentials from `Global.*`.

## Contents
- Active Directory, RapidIdentity Metadirectory, Portal, Google (+ callGoogleAPI), Microsoft 365, Database
- AES Community Adapter (encrypt/decrypt)
- Connection failure handling and closing connections

---


Use the correct built-in connection action for each target system. Always store the result in
`outputVar="session"` and check for failure before proceeding. Never hardcode credentials —
always use `Global.*` references.

### Active Directory — `openADConnection`

```xml
<action name="getIdBridgeConnectInfo" outputVar="bridgeInfo">
  <arg name="name" value="bridge"/>
</action>
<action name="openADConnection" outputVar="session">
  <arg name="adHost" value="bridgeInfo.host"/>
  <arg name="adPort" value="bridgeInfo.port"/>
  <arg name="useSSL" value="true"/>
  <arg name="userDn" value="Global.adUserUPN"/>
  <arg name="password" value="Global.adPwd"/>
  <arg name="extraProperties" value="{&quot;followReferrals&quot;: false}"/>
</action>
```

- Requires a bridge name resolved via `getIdBridgeConnectInfo` first.
- `Global.adUserUPN` and `Global.adPwd` are the standard credential Globals.
- Set `useSSL="true"` and `followReferrals: false` by default.

### RapidIdentity (Metadirectory LDAP) — `openMetadirLDAPConnection`

```xml
<action name="openMetadirLDAPConnection" outputVar="session"/>
```

- No arguments required — Connect resolves the internal Metadirectory connection automatically.
- Use for any action set that reads from or writes to the RI OpenLDAP Metadirectory.

**`getLDAPRecords` — `baseDn` selection:**

| Searching for | `baseDn` |
|---|---|
| Groups only | `Global.metaGroupBaseDN` |
| Users (employees/students) only | `Global.metaEmployeeBaseDN` |
| Both users and groups | `Global.metaBaseDN` |

**`getLDAPRecords` — `attributes` arg values:**

| Value | Returns |
|---|---|
| `"*,+"` | All standard attributes plus operational attributes (objectClass, createTimestamp, etc.) |
| `"*"` | All standard user-defined attributes only |
| `"cn,mail,idautoID"` | Only the named attributes (comma-separated, quoted string) |

Never use a bare `[]` array literal — this fails the Connect expression compiler. Always use a quoted string.

```xml
<action id="A1B2C3D4-E5F6-7890-ABCD-EF1234567890" name="getLDAPRecords" outputVar="results" disabled="false">
  <arg name="ldapConnection" value="sessionRI"/>
  <arg name="baseDn" value="Global.metaBaseDN"/>
  <arg name="scope" value="&quot;sub&quot;"/>
  <arg name="filter" value="&quot;(idautoPersonUserNameMV=jkirk)&quot;"/>
  <arg name="attributes" value="&quot;*,+&quot;"/>
  <arg name="maxResults" value="1"/>
</action>
```

### RapidIdentity Portal — `defineCloudPortalConnection`

```xml
<action name="defineCloudPortalConnection" outputVar="session"/>
```

- No arguments required — Connect resolves portal URL and token automatically.
- Returns a record with `.url` and `.token` properties used for subsequent REST calls:
  ```
  sessionPortal.url + "/api/rest/..."
  {Authorization: "Bearer " + sessionPortal.token}
  ```
- Close when done with: `<action name="close"><arg name="closeable" value="session"/></action>`

### Google — `defineGoogleExtendedOAuthConnection`

```xml
<action name="defineGoogleExtendedOAuthConnection" outputVar="session">
  <arg name="domain" value="Global.googleDomain"/>
  <arg name="credentialName" value="Global.googleOAuthCredentialName"/>
  <arg name="scopes" value="Global.googleOAuthScopes"/>
  <arg name="impersonateUserId" value="Global.googleImpersonateUserId"/>
</action>
```

- All four args are required; all sourced from Globals.
- Used with `callGoogleAPI` actions for Directory, Classroom, Drive, etc.

**Generic `callGoogleAPI` invocation** — for any endpoint not covered by a dedicated action:

```xml
<action name="callGoogleAPI" outputVar="result">
  <arg name="connection" value="sessionGoogle"/>
  <arg name="method" value="&quot;PATCH&quot;"/>
  <arg name="url" value="&quot;https://admin.googleapis.com/admin/directory/v1/groups/&quot; + groupId + &quot;/members/&quot; + memberId"/>
  <arg name="data" value="data"/>
  <arg name="contentType" value="&quot;application/json&quot;"/>
</action>
```

- `connection` is the session from `defineGoogleExtendedOAuthConnection`.
- `data` is the request body (a record/object); `method`, `url`, `contentType` are quoted strings.
- **Not closeable** — this is an OAuth2 connection; do not call `close` on it (see § Closing Connections).

### Microsoft 365 — OAuth2 via `httpPOST`

Microsoft 365 has no built-in connection action. Obtain a bearer token manually:

```xml
<action name="setVariable">
  <arg name="name" value="postBody"/>
  <arg name="value" value="&quot;grant_type=client_credentials&amp;client_id=&quot;+Global.microsoftApplicationClientID+&quot;&amp;client_secret=&quot;+Global.microsoftClientSecretValue+&quot;&amp;resource=https://graph.microsoft.com&amp;resource_type=token&quot;"/>
</action>
<action name="httpPOST" outputVar="token">
  <arg name="url" value="&quot;https://login.microsoftonline.com/&quot;+Global.microsoftDirectoryTenantID+&quot;/oauth2/token&quot;"/>
  <arg name="headers" value="{&quot;Content-Type&quot;:&quot;application/x-www-form-urlencoded&quot;}"/>
  <arg name="data" value="postBody"/>
</action>
<action name="if">
  <arg name="condition" value="token.statusCode == 200"/>
  <arg name="then">
    <action name="setVariable">
      <arg name="name" value="session"/>
      <arg name="value" value="token.data.access_token"/>
    </action>
  </arg>
</action>
```

- Globals required: `microsoftApplicationClientID`, `microsoftClientSecretValue`, `microsoftDirectoryTenantID`.
- `session` will hold the raw bearer token string for use in subsequent Graph API calls.
- **Not closeable** — this is an OAuth2 / bearer-token connection, not an open handle; do not call `close` on it (see § Closing Connections).

### Database — `openDatabaseConnection`

```xml
<action name="stringFromTemplate" outputVar="dbConnString">
  <arg name="format" value="Global.dbConnStringTemplate"/>
  <arg name="args" value="{&quot;bridgeHostName&quot;:bridgeInfo.host,&quot;bridgePort&quot;:bridgeInfo.port,&quot;databaseName&quot;:Global.dbName}"/>
</action>
<action name="openDatabaseConnection" outputVar="session">
  <arg name="jdbcDriverClass" value="Global.dbDriverClass"/>
  <arg name="jdbcURL" value="dbConnString"/>
  <arg name="user" value="Global.dbUser"/>
  <arg name="password" value="Global.dbPwd"/>
  <arg name="extraProperties" value="{&quot;socketTimeout&quot;:&quot;9000&quot;}"/>
</action>
```

- Requires a bridge resolved via `getIdBridgeConnectInfo`.
- Connection string is built from `Global.dbConnStringTemplate` using `stringFromTemplate`.

### AES Community Adapter — encryption / decryption

The AES community adapter provides actions to keep sensitive values encrypted (e.g. national IDs,
claim codes). Generate a key once and persist it; thereafter load the key to encrypt/decrypt.

```xml
<!-- One-time: generate a key and save it to a file -->
<action name="GenerateAESKey" outputVar="aesKey"/>
<action name="ConvertAESKeyToBytes" outputVar="keyBytes"><arg name="secretKey" value="aesKey"/></action>
<action name="saveToFile"><arg name="path" value="&quot;/keys/AESEncryptionKey.key&quot;"/><arg name="value" value="keyBytes"/></action>

<!-- Load and parse the key for use -->
<action name="loadFileAsBytes" outputVar="keyBytes"><arg name="path" value="&quot;/keys/AESEncryptionKey.key&quot;"/></action>
<action name="GetAESKey" outputVar="secretKey"><arg name="keyData" value="keyBytes"/></action>

<!-- Encrypt: AESEncrypt returns a [initializationVector, cipherText] array -->
<action name="AESEncrypt" outputVar="encrypted">
  <arg name="secretKey" value="aesKey"/>
  <arg name="blockCipherMode" value="&quot;CBC&quot;"/>
  <arg name="data" value="textToEncrypt"/>
</action>
<action name="setVariable"><arg name="name" value="iv"/><arg name="value" value="encrypted[0]"/></action>
<action name="setVariable"><arg name="name" value="cipherText"/><arg name="value" value="encrypted[1]"/></action>

<!-- Decrypt: needs secretKey, cipherText, and the same IV -->
<action name="AESDecrypt" outputVar="plainText">
  <arg name="secretKey" value="secretKey"/>
  <arg name="blockCipherMode" value="&quot;CBC&quot;"/>
  <arg name="cipherText" value="cipherText"/>
  <arg name="initializationVector" value="iv"/>
</action>
```

- `AESEncrypt` returns a two-element array: index `0` is the initialization vector, index `1` is the
  cipher text. Persist both — decryption requires the IV.
- Use `CBC` block cipher mode on both encrypt and decrypt.
- Store the key file outside the action set (e.g. project `files/keys/`), never inline.

### Connection Failure Handling

After any connection attempt, check `!session` and log with audit before returning:

```xml
<action name="if">
  <arg name="condition" value="!session"/>
  <arg name="then">
    <action name="log">
      <arg name="message" value="target + &quot; Connection Failed: &quot; + now()"/>
      <arg name="level" value="&quot;ERROR&quot;"/>
      <arg name="color" value="logColors.fail"/>
    </action>
    <action name="return"><arg name="value" value="false"/></action>
  </arg>
</action>
```

Pass the open session to LDAP/directory actions via `ldapConnection`.

### Closing Connections — `close`

Only **closeable** connection/IO actions are closed, and they all use the same action:

```xml
<action id="A1B2C3D4-E5F6-7890-ABCD-EF1234567890" name="close" outputVar="" disabled="false">
  <arg name="closeable" value="session"/>
</action>
```

- The arg name is `closeable`, not `ldapConnection` or `connection`.
- Always close in a `closeConnections` section (or wherever the session goes out of scope), and in
  function mode only when this action set opened the connection itself.

**Closeable connection/IO actions** — call `close` on the session returned by any of these:

| Group | Actions |
|---|---|
| Text / file I/O | `openDelimitedTextInput`, `openDelimitedTextOutput`, `openTextInput`, `openTextOutput`, `openLDIFOutput` |
| Directory / LDAP | `openADConnection`, `openMetadirLDAPConnection`, `openLDAPConnection` |
| Portal | `definePortalConnection`, `defineCloudPortalConnection` |
| Database | `openDatabaseConnection` |
| Cloud / app | `openForceComConnection`, `openExchangeConnection`, `openOffice365Connection`, `openAWSIAMConnection` |
| Remote execution / tunneling | `openRemoteCLI`, `openRemoteCLIWithCert`, `openRemoteWindowsCLI`, `openPortForwardingSession`, `openPortForwardingSessionWithCert` |

**Not closeable — never call `close` on these.** OAuth2 or HTTP Basic style connections hold a token
or credential rather than an open handle, so there is nothing to close. This includes:

- The Microsoft Graph API connection (the bearer token obtained via `httpPOST` above).
- `defineGoogleExtendedOAuthConnection` (Google OAuth).
- Any other OAuth2 / HTTP Basic connection.

If a connection action is not in the closeable table above, do not call `close` on it.

---

