# Connector configuration reads and edits

Payments configuration reads return safe metadata only. This applies to the
legacy connector configuration route and the v3 configuration and connector list
routes. The legacy handler also serves clients that use v1/v2 upstream routing.

The `config` object remains present in v3 list rows. Readable fields are `name`
and `pollingPeriod`, when stored as strings. Configuration detail responses also
include the existing `provider` discriminator. Provider-specific fields are
omitted, including unknown and nested fields. Malformed stored configuration
returns no configuration values. The internal connector model and encrypted
storage representation are unchanged.

## Update semantics

Legacy configuration POST and v3 configuration PATCH accept partial objects.
The server merges the object with the stored configuration before provider
validation and persistence. Installation still requires the full provider
configuration.

| Input | Result |
| --- | --- |
| Omitted field | Preserve the stored value. |
| Omitted field inside an object | Preserve the stored nested value. |
| Empty object | Preserve existing object fields. |
| Supplied value | Replace the stored value, then apply provider validation. |
| Empty string | Explicit replacement; required credentials fail provider validation, while optional strings can be cleared. |
| Null field | Explicit replacement; the existing provider validation and normalization determine whether it is accepted. |
| Supplied array | Replace the whole array; elements are not merged by position. |
| Empty body, null body, or non-object body | Reject the update. |

The merge preserves omitted provider fields, including non-secret fields. This
conservative rule supports opaque provider configurations without guessing which
field names contain credentials. The merge retains JSON number precision. Existing provider validation and
normalization still determine the persisted configuration; unsupported fields
can be discarded during that step.
The existing update path does not serialize concurrent edits; callers should
serialize configuration edits and rotations for the same connector.

## Migration

This is an intentional security change to existing read behavior. Clients must
stop using configuration reads to recover provider settings or stored secrets.
The generated SDK uses `ConnectorConfigRead` and `V3ConnectorConfigRead` for reads.
Update methods accept `map[string]any` so they send only explicitly changed
fields, without injecting provider defaults or requiring stored credentials.

Console and other external consumers must display omitted credentials as
unchanged and omit them from update requests unless the user supplies a
replacement. Do not send masked placeholders as credential values. Consumers
that need to display other provider settings require a separately reviewed safe
metadata contract; they must not rely on the old read response.

No stored-secret reveal endpoint or creation/rotation disclosure flow is added.
The caller already knows credentials it supplies on install or replacement.
Operators must assess rotation of credentials exposed before this change; this
patch does not rotate credentials or verify deployed consumers.
