# RapidIdentity Sponsorship API


### Key endpoints

| Method | Path | Purpose |
|---|---|---|
| `POST` | `api/rest/v2/sponsoredAccounts?checkForDuplicates=true` | Create sponsored account |
| `GET` | `api/rest/portal/bootstrapInfo` | Tenant config including sponsorship custom attribute UUIDs |
| `GET` | `api/rest/admin/sponsorship/attributeMapItems` | List registered sponsorship custom attributes |
| `PUT` | `api/rest/admin/sponsorship/attributeMapItems` | Register a new sponsorship custom attribute |
| `GET` | `api/rest/admin/gal/items` | Full GAL catalogue for attribute UUID lookup |

### Create sponsored account — request body

```javascript
toJSON({
  dn: null,
  username: idautopersondistrictid,   // requested username
  status: null,
  givenName: givenname,
  surname: sn,
  email: (mail || null),
  expirationDate: (idautopersonenddate ? idautopersonenddate.substring(0,10) : null),
  disabled: false,
  sponsorDN: manager,                 // full metadirectory DN of the sponsor
  customAttributes: resolvedCustomAttrs,  // array of {id: uuid, values: [...]}
  message: null,
  checkForDuplicates: true,
  profileId: null,
  sponsor: null,
  sponsorUser: null
})
```

- `customAttributes` UUIDs come from the `bootstrapInfo` response — never hardcode them
- `expirationDate` must be `YYYY-MM-DD` — strip the time component from workflow date fields
- A `409` response means a duplicate account was detected; handle separately from other errors
- On success: `apiResponse.data.sponsoredAccounts[0]` contains the created account record

### Resolving custom attribute UUIDs

The Sponsorship API requires custom attribute objects by UUID, not by LDAP attribute name.
The UUID-to-name map must be built at runtime from `bootstrapInfo`:

```javascript
// Build lookup map: ldapAttributeName -> UUID
// configuredCustomAttrs = bootstrapInfo.data.sponsorshipSettings.sponsorshipCustomAttributes
forEach(configAttr in configuredCustomAttrs) {
  attrNameToId[configAttr.galItem.attributeName] = configAttr.id
}

// Resolve caller-supplied attrs
forEach(callerAttr in callerAttrs) {
  resolvedId = attrNameToId[callerAttr.ldapAttributeName]
  if (resolvedId) {
    appendArrayItem(resolvedCustomAttrs, {id: resolvedId, values: [].concat(callerAttr.values)})
  }
}
```

---

