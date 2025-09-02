<!-- Generator: Widdershins v4.0.1 -->

<h1 id="payments-api">Payments API v1</h1>

> Scroll down for code samples, example requests and responses. Select a language for code samples from the tabs above or the mobile navigation menu.

<h1 id="payments-api-payments-v1">payments.v1</h1>

## Get server info

<a id="opIdgetServerInfo"></a>

> Code samples

```http
GET /_info HTTP/1.1

Accept: application/json

```

`GET /_info`

> Example responses

<h3 id="get-server-info-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Server information|None|
|default|Default|none|None|

<h3 id="get-server-info-responseschema">Response Schema</h3>

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

<h1 id="payments-api-payments-v3">payments.v3</h1>

## Create a formance account object. This object will not be forwarded to the connector. It is only used for internal purposes.

<a id="opIdv3CreateAccount"></a>

> Code samples

```http
POST /v3/accounts HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`POST /v3/accounts`

> Body parameter

```json
{
  "reference": "string",
  "connectorID": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "accountName": "string",
  "type": "UNKNOWN",
  "defaultAsset": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}
```

<h3 id="create-a-formance-account-object.-this-object-will-not-be-forwarded-to-the-connector.-it-is-only-used-for-internal-purposes.
-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[V3CreateAccountRequest](#schemav3createaccountrequest)|false|none|

> Example responses

> 201 Response

```json
{
  "data": {
    "id": "string",
    "connectorID": "string",
    "provider": "string",
    "reference": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "type": "UNKNOWN",
    "name": "string",
    "defaultAsset": "string",
    "metadata": {
      "property1": "string",
      "property2": "string"
    },
    "raw": {}
  }
}
```

<h3 id="create-a-formance-account-object.-this-object-will-not-be-forwarded-to-the-connector.-it-is-only-used-for-internal-purposes.
-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|201|[Created](https://tools.ietf.org/html/rfc7231#section-6.3.2)|Created|[V3CreateAccountResponse](#schemav3createaccountresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## List all accounts

<a id="opIdv3ListAccounts"></a>

> Code samples

```http
GET /v3/accounts HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`GET /v3/accounts`

> Body parameter

```json
{}
```

<h3 id="list-all-accounts-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|
|body|body|[V3QueryBuilder](#schemav3querybuilder)|false|none|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "connectorID": "string",
        "provider": "string",
        "reference": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "type": "UNKNOWN",
        "name": "string",
        "defaultAsset": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        },
        "raw": {}
      }
    ]
  }
}
```

<h3 id="list-all-accounts-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3AccountsCursorResponse](#schemav3accountscursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Get an account by ID

<a id="opIdv3GetAccount"></a>

> Code samples

```http
GET /v3/accounts/{accountID} HTTP/1.1

Accept: application/json

```

`GET /v3/accounts/{accountID}`

<h3 id="get-an-account-by-id-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|accountID|path|string|true|The account ID|

> Example responses

> 200 Response

```json
{
  "data": {
    "id": "string",
    "connectorID": "string",
    "provider": "string",
    "reference": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "type": "UNKNOWN",
    "name": "string",
    "defaultAsset": "string",
    "metadata": {
      "property1": "string",
      "property2": "string"
    },
    "raw": {}
  }
}
```

<h3 id="get-an-account-by-id-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3GetAccountResponse](#schemav3getaccountresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Get account balances

<a id="opIdv3GetAccountBalances"></a>

> Code samples

```http
GET /v3/accounts/{accountID}/balances HTTP/1.1

Accept: application/json

```

`GET /v3/accounts/{accountID}/balances`

<h3 id="get-account-balances-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|accountID|path|string|true|The account ID|
|asset|query|string|false|The asset to filter by|
|fromTimestamp|query|string(date-time)|false|The start of the time range to filter by|
|toTimestamp|query|string(date-time)|false|The end of the time range to filter by|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "accountID": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "lastUpdatedAt": "2019-08-24T14:15:22Z",
        "asset": "string",
        "balance": 0
      }
    ]
  }
}
```

<h3 id="get-account-balances-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3BalancesCursorResponse](#schemav3balancescursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Create a formance bank account object. This object will not be forwarded to the connector until you called the forwardBankAccount method.

<a id="opIdv3CreateBankAccount"></a>

> Code samples

```http
POST /v3/bank-accounts HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`POST /v3/bank-accounts`

> Body parameter

```json
{
  "name": "string",
  "accountNumber": "string",
  "iban": "string",
  "swiftBicCode": "string",
  "country": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}
```

<h3 id="create-a-formance-bank-account-object.-this-object-will-not-be-forwarded-to-the-connector-until-you-called-the-forwardbankaccount-method.
-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[V3CreateBankAccountRequest](#schemav3createbankaccountrequest)|false|none|

> Example responses

> 201 Response

```json
{
  "data": "string"
}
```

<h3 id="create-a-formance-bank-account-object.-this-object-will-not-be-forwarded-to-the-connector-until-you-called-the-forwardbankaccount-method.
-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|201|[Created](https://tools.ietf.org/html/rfc7231#section-6.3.2)|Created|[V3CreateBankAccountResponse](#schemav3createbankaccountresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## List all bank accounts

<a id="opIdv3ListBankAccounts"></a>

> Code samples

```http
GET /v3/bank-accounts HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`GET /v3/bank-accounts`

> Body parameter

```json
{}
```

<h3 id="list-all-bank-accounts-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|
|body|body|[V3QueryBuilder](#schemav3querybuilder)|false|none|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "name": "string",
        "accountNumber": "string",
        "iban": "string",
        "swiftBicCode": "string",
        "country": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        },
        "relatedAccounts": [
          {
            "accountID": "string",
            "createdAt": "2019-08-24T14:15:22Z"
          }
        ]
      }
    ]
  }
}
```

<h3 id="list-all-bank-accounts-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3BankAccountsCursorResponse](#schemav3bankaccountscursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Get a Bank Account by ID

<a id="opIdv3GetBankAccount"></a>

> Code samples

```http
GET /v3/bank-accounts/{bankAccountID} HTTP/1.1

Accept: application/json

```

`GET /v3/bank-accounts/{bankAccountID}`

<h3 id="get-a-bank-account-by-id-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|bankAccountID|path|string|true|The bank account ID|

> Example responses

> 200 Response

```json
{
  "data": {
    "id": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "name": "string",
    "accountNumber": "string",
    "iban": "string",
    "swiftBicCode": "string",
    "country": "string",
    "metadata": {
      "property1": "string",
      "property2": "string"
    },
    "relatedAccounts": [
      {
        "accountID": "string",
        "createdAt": "2019-08-24T14:15:22Z"
      }
    ]
  }
}
```

<h3 id="get-a-bank-account-by-id-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3GetBankAccountResponse](#schemav3getbankaccountresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="success">
This operation does not require authentication
</aside>

## Update a bank account's metadata

<a id="opIdv3UpdateBankAccountMetadata"></a>

> Code samples

```http
PATCH /v3/bank-accounts/{bankAccountID}/metadata HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`PATCH /v3/bank-accounts/{bankAccountID}/metadata`

> Body parameter

```json
{
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}
```

<h3 id="update-a-bank-account's-metadata-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|bankAccountID|path|string|true|The bank account ID|
|body|body|[V3UpdateBankAccountMetadataRequest](#schemav3updatebankaccountmetadatarequest)|false|none|

> Example responses

> default Response

```json
{
  "errorCode": "VALIDATION",
  "errorMessage": "[VALIDATION] missing required config field: pollingPeriod",
  "details": "string"
}
```

<h3 id="update-a-bank-account's-metadata-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|No Content|None|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="success">
This operation does not require authentication
</aside>

## Forward a Bank Account to a PSP for creation

<a id="opIdv3ForwardBankAccount"></a>

> Code samples

```http
POST /v3/bank-accounts/{bankAccountID}/forward HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`POST /v3/bank-accounts/{bankAccountID}/forward`

> Body parameter

```json
{
  "connectorID": "string"
}
```

<h3 id="forward-a-bank-account-to-a-psp-for-creation-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|bankAccountID|path|string|true|The bank account ID|
|body|body|[V3ForwardBankAccountRequest](#schemav3forwardbankaccountrequest)|false|none|

> Example responses

> 202 Response

```json
{
  "data": {
    "taskID": "string"
  }
}
```

<h3 id="forward-a-bank-account-to-a-psp-for-creation-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|202|[Accepted](https://tools.ietf.org/html/rfc7231#section-6.3.3)|Accepted|[V3ForwardBankAccountResponse](#schemav3forwardbankaccountresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="success">
This operation does not require authentication
</aside>

## List all connectors

<a id="opIdv3ListConnectors"></a>

> Code samples

```http
GET /v3/connectors HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`GET /v3/connectors`

> Body parameter

```json
{}
```

<h3 id="list-all-connectors-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|
|body|body|[V3QueryBuilder](#schemav3querybuilder)|false|none|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "reference": "string",
        "name": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "provider": "string",
        "scheduledForDeletion": true,
        "config": {}
      }
    ]
  }
}
```

<h3 id="list-all-connectors-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3ConnectorsCursorResponse](#schemav3connectorscursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Install a connector

<a id="opIdv3InstallConnector"></a>

> Code samples

```http
POST /v3/connectors/install/{connector} HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`POST /v3/connectors/install/{connector}`

> Body parameter

```json
{
  "apiKey": "string",
  "companyID": "string",
  "liveEndpointPrefix": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Adyen",
  "webhookPassword": "string",
  "webhookUsername": "string"
}
```

<h3 id="install-a-connector-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|connector|path|string|true|The connector to filter by|
|body|body|[V3ConnectorConfig](#schemav3connectorconfig)|false|none|

> Example responses

> 202 Response

```json
{
  "data": "string"
}
```

<h3 id="install-a-connector-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|202|[Accepted](https://tools.ietf.org/html/rfc7231#section-6.3.3)|Accepted|[V3InstallConnectorResponse](#schemav3installconnectorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## List all connector configurations

<a id="opIdv3ListConnectorConfigs"></a>

> Code samples

```http
GET /v3/connectors/configs HTTP/1.1

Accept: application/json

```

`GET /v3/connectors/configs`

> Example responses

> 200 Response

```json
{
  "data": {
    "property1": {
      "property1": {
        "dataType": "string",
        "required": true,
        "defaultValue": "string"
      },
      "property2": {
        "dataType": "string",
        "required": true,
        "defaultValue": "string"
      }
    },
    "property2": {
      "property1": {
        "dataType": "string",
        "required": true,
        "defaultValue": "string"
      },
      "property2": {
        "dataType": "string",
        "required": true,
        "defaultValue": "string"
      }
    }
  }
}
```

<h3 id="list-all-connector-configurations-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3ConnectorConfigsResponse](#schemav3connectorconfigsresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Uninstall a connector

<a id="opIdv3UninstallConnector"></a>

> Code samples

```http
DELETE /v3/connectors/{connectorID} HTTP/1.1

Accept: application/json

```

`DELETE /v3/connectors/{connectorID}`

<h3 id="uninstall-a-connector-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|connectorID|path|string|true|The connector ID|

> Example responses

> 202 Response

```json
{
  "data": {
    "taskID": "string"
  }
}
```

<h3 id="uninstall-a-connector-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|202|[Accepted](https://tools.ietf.org/html/rfc7231#section-6.3.3)|Accepted|[V3UninstallConnectorResponse](#schemav3uninstallconnectorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Get a connector configuration by ID

<a id="opIdv3GetConnectorConfig"></a>

> Code samples

```http
GET /v3/connectors/{connectorID}/config HTTP/1.1

Accept: application/json

```

`GET /v3/connectors/{connectorID}/config`

<h3 id="get-a-connector-configuration-by-id-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|connectorID|path|string|true|The connector ID|

> Example responses

> 200 Response

```json
{
  "data": {
    "apiKey": "string",
    "companyID": "string",
    "liveEndpointPrefix": "string",
    "name": "string",
    "pageSize": 25,
    "pollingPeriod": "2m",
    "provider": "Adyen",
    "webhookPassword": "string",
    "webhookUsername": "string"
  }
}
```

<h3 id="get-a-connector-configuration-by-id-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3GetConnectorConfigResponse](#schemav3getconnectorconfigresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Update the config of a connector

<a id="opIdv3UpdateConnectorConfig"></a>

> Code samples

```http
PATCH /v3/connectors/{connectorID}/config HTTP/1.1

Content-Type: application/json

```

`PATCH /v3/connectors/{connectorID}/config`

Update connector config

> Body parameter

```json
{
  "apiKey": "string",
  "companyID": "string",
  "liveEndpointPrefix": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Adyen",
  "webhookPassword": "string",
  "webhookUsername": "string"
}
```

<h3 id="update-the-config-of-a-connector-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|connectorID|path|string|true|The connector ID|
|body|body|[V3ConnectorConfig](#schemav3connectorconfig)|false|none|

<h3 id="update-the-config-of-a-connector-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|none|None|
|default|Default|none|None|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Reset a connector. Be aware that this will delete all data and stop all existing tasks like payment initiations and bank account creations.

<a id="opIdv3ResetConnector"></a>

> Code samples

```http
POST /v3/connectors/{connectorID}/reset HTTP/1.1

Accept: application/json

```

`POST /v3/connectors/{connectorID}/reset`

<h3 id="reset-a-connector.-be-aware-that-this-will-delete-all-data-and-stop-all-existing-tasks-like-payment-initiations-and-bank-account-creations.-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|connectorID|path|string|true|The connector ID|

> Example responses

> 202 Response

```json
{
  "data": "string"
}
```

<h3 id="reset-a-connector.-be-aware-that-this-will-delete-all-data-and-stop-all-existing-tasks-like-payment-initiations-and-bank-account-creations.-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|202|[Accepted](https://tools.ietf.org/html/rfc7231#section-6.3.3)|Accepted|[V3ResetConnectorResponse](#schemav3resetconnectorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## List all connector schedules

<a id="opIdv3ListConnectorSchedules"></a>

> Code samples

```http
GET /v3/connectors/{connectorID}/schedules HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`GET /v3/connectors/{connectorID}/schedules`

> Body parameter

```json
{}
```

<h3 id="list-all-connector-schedules-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|connectorID|path|string|true|The connector ID|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|
|body|body|[V3QueryBuilder](#schemav3querybuilder)|false|none|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "connectorID": "string",
        "createdAt": "2019-08-24T14:15:22Z"
      }
    ]
  }
}
```

<h3 id="list-all-connector-schedules-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3ConnectorSchedulesCursorResponse](#schemav3connectorschedulescursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Get a connector schedule by ID

<a id="opIdv3GetConnectorSchedule"></a>

> Code samples

```http
GET /v3/connectors/{connectorID}/schedules/{scheduleID} HTTP/1.1

Accept: application/json

```

`GET /v3/connectors/{connectorID}/schedules/{scheduleID}`

<h3 id="get-a-connector-schedule-by-id-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|connectorID|path|string|true|The connector ID|
|scheduleID|path|string|true|The schedule ID|

> Example responses

> 200 Response

```json
{
  "data": {
    "id": "string",
    "connectorID": "string",
    "createdAt": "2019-08-24T14:15:22Z"
  }
}
```

<h3 id="get-a-connector-schedule-by-id-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3ConnectorScheduleResponse](#schemav3connectorscheduleresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## List all connector schedule instances

<a id="opIdv3ListConnectorScheduleInstances"></a>

> Code samples

```http
GET /v3/connectors/{connectorID}/schedules/{scheduleID}/instances HTTP/1.1

Accept: application/json

```

`GET /v3/connectors/{connectorID}/schedules/{scheduleID}/instances`

<h3 id="list-all-connector-schedule-instances-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|connectorID|path|string|true|The connector ID|
|scheduleID|path|string|true|The schedule ID|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "connectorID": "string",
        "scheduleID": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "updatedAt": "2019-08-24T14:15:22Z",
        "terminated": true,
        "terminatedAt": "2019-08-24T14:15:22Z",
        "error": "string"
      }
    ]
  }
}
```

<h3 id="list-all-connector-schedule-instances-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3ConnectorScheduleInstancesCursorResponse](#schemav3connectorscheduleinstancescursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Create a formance payment object. This object will not be forwarded to the connector. It is only used for internal purposes.

<a id="opIdv3CreatePayment"></a>

> Code samples

```http
POST /v3/payments HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`POST /v3/payments`

> Body parameter

```json
{
  "reference": "string",
  "connectorID": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "type": "UNKNOWN",
  "initialAmount": 0,
  "amount": 0,
  "asset": "string",
  "scheme": "string",
  "sourceAccountID": "string",
  "destinationAccountID": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  },
  "adjustments": [
    {
      "reference": "string",
      "createdAt": "2019-08-24T14:15:22Z",
      "status": "UNKNOWN",
      "amount": 0,
      "asset": "string",
      "metadata": {
        "property1": "string",
        "property2": "string"
      }
    }
  ]
}
```

<h3 id="create-a-formance-payment-object.-this-object-will-not-be-forwarded-to-the-connector.-it-is-only-used-for-internal-purposes.
-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[V3CreatePaymentRequest](#schemav3createpaymentrequest)|false|none|

> Example responses

> 201 Response

```json
{
  "data": {
    "id": "string",
    "connectorID": "string",
    "provider": "string",
    "reference": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "type": "UNKNOWN",
    "initialAmount": 0,
    "amount": 0,
    "asset": "string",
    "scheme": "string",
    "status": "UNKNOWN",
    "sourceAccountID": "string",
    "destinationAccountID": "string",
    "metadata": {
      "property1": "string",
      "property2": "string"
    },
    "adjustments": [
      {
        "id": "string",
        "reference": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "status": "UNKNOWN",
        "amount": 0,
        "asset": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        },
        "raw": {}
      }
    ]
  }
}
```

<h3 id="create-a-formance-payment-object.-this-object-will-not-be-forwarded-to-the-connector.-it-is-only-used-for-internal-purposes.
-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|201|[Created](https://tools.ietf.org/html/rfc7231#section-6.3.2)|Created|[V3CreatePaymentResponse](#schemav3createpaymentresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## List all payments

<a id="opIdv3ListPayments"></a>

> Code samples

```http
GET /v3/payments HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`GET /v3/payments`

> Body parameter

```json
{}
```

<h3 id="list-all-payments-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|
|body|body|[V3QueryBuilder](#schemav3querybuilder)|false|none|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "connectorID": "string",
        "provider": "string",
        "reference": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "type": "UNKNOWN",
        "initialAmount": 0,
        "amount": 0,
        "asset": "string",
        "scheme": "string",
        "status": "UNKNOWN",
        "sourceAccountID": "string",
        "destinationAccountID": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        },
        "adjustments": [
          {
            "id": "string",
            "reference": "string",
            "createdAt": "2019-08-24T14:15:22Z",
            "status": "UNKNOWN",
            "amount": 0,
            "asset": "string",
            "metadata": {
              "property1": "string",
              "property2": "string"
            },
            "raw": {}
          }
        ]
      }
    ]
  }
}
```

<h3 id="list-all-payments-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3PaymentsCursorResponse](#schemav3paymentscursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Get a payment by ID

<a id="opIdv3GetPayment"></a>

> Code samples

```http
GET /v3/payments/{paymentID} HTTP/1.1

Accept: application/json

```

`GET /v3/payments/{paymentID}`

<h3 id="get-a-payment-by-id-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentID|path|string|true|The payment ID|

> Example responses

> 200 Response

```json
{
  "data": {
    "id": "string",
    "connectorID": "string",
    "provider": "string",
    "reference": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "type": "UNKNOWN",
    "initialAmount": 0,
    "amount": 0,
    "asset": "string",
    "scheme": "string",
    "status": "UNKNOWN",
    "sourceAccountID": "string",
    "destinationAccountID": "string",
    "metadata": {
      "property1": "string",
      "property2": "string"
    },
    "adjustments": [
      {
        "id": "string",
        "reference": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "status": "UNKNOWN",
        "amount": 0,
        "asset": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        },
        "raw": {}
      }
    ]
  }
}
```

<h3 id="get-a-payment-by-id-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3GetPaymentResponse](#schemav3getpaymentresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Update a payment's metadata

<a id="opIdv3UpdatePaymentMetadata"></a>

> Code samples

```http
PATCH /v3/payments/{paymentID}/metadata HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`PATCH /v3/payments/{paymentID}/metadata`

> Body parameter

```json
{
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}
```

<h3 id="update-a-payment's-metadata-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentID|path|string|true|The payment ID|
|body|body|[V3UpdatePaymentMetadataRequest](#schemav3updatepaymentmetadatarequest)|false|none|

> Example responses

> default Response

```json
{
  "errorCode": "VALIDATION",
  "errorMessage": "[VALIDATION] missing required config field: pollingPeriod",
  "details": "string"
}
```

<h3 id="update-a-payment's-metadata-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|No Content|None|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Initiate a payment

<a id="opIdv3InitiatePayment"></a>

> Code samples

```http
POST /v3/payment-initiations HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`POST /v3/payment-initiations`

> Body parameter

```json
{
  "reference": "string",
  "scheduledAt": "2019-08-24T14:15:22Z",
  "connectorID": "string",
  "description": "string",
  "type": "UNKNOWN",
  "amount": 0,
  "asset": "string",
  "sourceAccountID": "string",
  "destinationAccountID": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}
```

<h3 id="initiate-a-payment-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|noValidation|query|boolean|false|If set to true, the request will not have to be validated. This is useful if we want to directly forward the request to the PSP.|
|body|body|[V3InitiatePaymentRequest](#schemav3initiatepaymentrequest)|false|none|

#### Detailed descriptions

**noValidation**: If set to true, the request will not have to be validated. This is useful if we want to directly forward the request to the PSP.

> Example responses

> 202 Response

```json
{
  "data": {
    "paymentInitiationID": "string",
    "taskID": "string"
  }
}
```

<h3 id="initiate-a-payment-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|202|[Accepted](https://tools.ietf.org/html/rfc7231#section-6.3.3)|Accepted|[V3InitiatePaymentResponse](#schemav3initiatepaymentresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## List all payment initiations

<a id="opIdv3ListPaymentInitiations"></a>

> Code samples

```http
GET /v3/payment-initiations HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`GET /v3/payment-initiations`

> Body parameter

```json
{}
```

<h3 id="list-all-payment-initiations-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|
|body|body|[V3QueryBuilder](#schemav3querybuilder)|false|none|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "connectorID": "string",
        "provider": "string",
        "reference": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "scheduledAt": "2019-08-24T14:15:22Z",
        "description": "string",
        "type": "UNKNOWN",
        "amount": 0,
        "asset": "string",
        "status": "UNKNOWN",
        "sourceAccountID": "string",
        "destinationAccountID": "string",
        "error": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        }
      }
    ]
  }
}
```

<h3 id="list-all-payment-initiations-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3PaymentInitiationsCursorResponse](#schemav3paymentinitiationscursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Delete a payment initiation by ID

<a id="opIdv3DeletePaymentInitiation"></a>

> Code samples

```http
DELETE /v3/payment-initiations/{paymentInitiationID} HTTP/1.1

Accept: application/json

```

`DELETE /v3/payment-initiations/{paymentInitiationID}`

<h3 id="delete-a-payment-initiation-by-id-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentInitiationID|path|string|true|The payment initiation ID|

> Example responses

> default Response

```json
{
  "errorCode": "VALIDATION",
  "errorMessage": "[VALIDATION] missing required config field: pollingPeriod",
  "details": "string"
}
```

<h3 id="delete-a-payment-initiation-by-id-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|No Content|None|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Get a payment initiation by ID

<a id="opIdv3GetPaymentInitiation"></a>

> Code samples

```http
GET /v3/payment-initiations/{paymentInitiationID} HTTP/1.1

Accept: application/json

```

`GET /v3/payment-initiations/{paymentInitiationID}`

<h3 id="get-a-payment-initiation-by-id-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentInitiationID|path|string|true|The payment initiation ID|

> Example responses

> 200 Response

```json
{
  "data": {
    "id": "string",
    "connectorID": "string",
    "provider": "string",
    "reference": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "scheduledAt": "2019-08-24T14:15:22Z",
    "description": "string",
    "type": "UNKNOWN",
    "amount": 0,
    "asset": "string",
    "status": "UNKNOWN",
    "sourceAccountID": "string",
    "destinationAccountID": "string",
    "error": "string",
    "metadata": {
      "property1": "string",
      "property2": "string"
    }
  }
}
```

<h3 id="get-a-payment-initiation-by-id-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3GetPaymentInitiationResponse](#schemav3getpaymentinitiationresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Retry a payment initiation

<a id="opIdv3RetryPaymentInitiation"></a>

> Code samples

```http
POST /v3/payment-initiations/{paymentInitiationID}/retry HTTP/1.1

Accept: application/json

```

`POST /v3/payment-initiations/{paymentInitiationID}/retry`

<h3 id="retry-a-payment-initiation-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentInitiationID|path|string|true|The payment initiation ID|

> Example responses

> 202 Response

```json
{
  "data": {
    "taskID": "string"
  }
}
```

<h3 id="retry-a-payment-initiation-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|202|[Accepted](https://tools.ietf.org/html/rfc7231#section-6.3.3)|Accepted|[V3RetryPaymentInitiationResponse](#schemav3retrypaymentinitiationresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Approve a payment initiation

<a id="opIdv3ApprovePaymentInitiation"></a>

> Code samples

```http
POST /v3/payment-initiations/{paymentInitiationID}/approve HTTP/1.1

Accept: application/json

```

`POST /v3/payment-initiations/{paymentInitiationID}/approve`

<h3 id="approve-a-payment-initiation-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentInitiationID|path|string|true|The payment initiation ID|

> Example responses

> 202 Response

```json
{
  "data": {
    "taskID": "string"
  }
}
```

<h3 id="approve-a-payment-initiation-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|202|[Accepted](https://tools.ietf.org/html/rfc7231#section-6.3.3)|Accepted|[V3ApprovePaymentInitiationResponse](#schemav3approvepaymentinitiationresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Reject a payment initiation

<a id="opIdv3RejectPaymentInitiation"></a>

> Code samples

```http
POST /v3/payment-initiations/{paymentInitiationID}/reject HTTP/1.1

Accept: application/json

```

`POST /v3/payment-initiations/{paymentInitiationID}/reject`

<h3 id="reject-a-payment-initiation-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentInitiationID|path|string|true|The payment initiation ID|

> Example responses

> default Response

```json
{
  "errorCode": "VALIDATION",
  "errorMessage": "[VALIDATION] missing required config field: pollingPeriod",
  "details": "string"
}
```

<h3 id="reject-a-payment-initiation-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|No Content|None|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Reverse a payment initiation

<a id="opIdv3ReversePaymentInitiation"></a>

> Code samples

```http
POST /v3/payment-initiations/{paymentInitiationID}/reverse HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`POST /v3/payment-initiations/{paymentInitiationID}/reverse`

> Body parameter

```json
{
  "reference": "string",
  "description": "string",
  "amount": 0,
  "asset": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}
```

<h3 id="reverse-a-payment-initiation-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentInitiationID|path|string|true|The payment initiation ID|
|body|body|[V3ReversePaymentInitiationRequest](#schemav3reversepaymentinitiationrequest)|false|none|

> Example responses

> 202 Response

```json
{
  "data": {
    "taskID": "string",
    "paymentInitiationReversalID": "string"
  }
}
```

<h3 id="reverse-a-payment-initiation-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|202|[Accepted](https://tools.ietf.org/html/rfc7231#section-6.3.3)|Accepted|[V3ReversePaymentInitiationResponse](#schemav3reversepaymentinitiationresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## List all payment initiation adjustments

<a id="opIdv3ListPaymentInitiationAdjustments"></a>

> Code samples

```http
GET /v3/payment-initiations/{paymentInitiationID}/adjustments HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`GET /v3/payment-initiations/{paymentInitiationID}/adjustments`

> Body parameter

```json
{}
```

<h3 id="list-all-payment-initiation-adjustments-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentInitiationID|path|string|true|The payment initiation ID|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|
|body|body|[V3QueryBuilder](#schemav3querybuilder)|false|none|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "status": "UNKNOWN",
        "amount": 0,
        "asset": "string",
        "error": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        }
      }
    ]
  }
}
```

<h3 id="list-all-payment-initiation-adjustments-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3PaymentInitiationAdjustmentsCursorResponse](#schemav3paymentinitiationadjustmentscursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## List all payments related to a payment initiation

<a id="opIdv3ListPaymentInitiationRelatedPayments"></a>

> Code samples

```http
GET /v3/payment-initiations/{paymentInitiationID}/payments HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`GET /v3/payment-initiations/{paymentInitiationID}/payments`

> Body parameter

```json
{}
```

<h3 id="list-all-payments-related-to-a-payment-initiation-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentInitiationID|path|string|true|The payment initiation ID|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|
|body|body|[V3QueryBuilder](#schemav3querybuilder)|false|none|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "connectorID": "string",
        "provider": "string",
        "reference": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "type": "UNKNOWN",
        "initialAmount": 0,
        "amount": 0,
        "asset": "string",
        "scheme": "string",
        "status": "UNKNOWN",
        "sourceAccountID": "string",
        "destinationAccountID": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        },
        "adjustments": [
          {
            "id": "string",
            "reference": "string",
            "createdAt": "2019-08-24T14:15:22Z",
            "status": "UNKNOWN",
            "amount": 0,
            "asset": "string",
            "metadata": {
              "property1": "string",
              "property2": "string"
            },
            "raw": {}
          }
        ]
      }
    ]
  }
}
```

<h3 id="list-all-payments-related-to-a-payment-initiation-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3PaymentInitiationRelatedPaymentsCursorResponse](#schemav3paymentinitiationrelatedpaymentscursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Create a formance payment service user object

<a id="opIdv3CreatePaymentServiceUser"></a>

> Code samples

```http
POST /v3/payment-service-users HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`POST /v3/payment-service-users`

> Body parameter

```json
{
  "name": "string",
  "contactDetails": {
    "email": "string",
    "phoneNumber": "string"
  },
  "address": {
    "streetNumber": "string",
    "streetName": "string",
    "city": "string",
    "region": "string",
    "postalCode": "string",
    "country": "string"
  },
  "bankAccountIDs": [
    "string"
  ],
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}
```

<h3 id="create-a-formance-payment-service-user-object-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[V3CreatePaymentServiceUserRequest](#schemav3createpaymentserviceuserrequest)|false|none|

> Example responses

> 201 Response

```json
{
  "data": "string"
}
```

<h3 id="create-a-formance-payment-service-user-object-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|201|[Created](https://tools.ietf.org/html/rfc7231#section-6.3.2)|Created|[V3CreatePaymentServiceUserResponse](#schemav3createpaymentserviceuserresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## List all payment service users

<a id="opIdv3ListPaymentServiceUsers"></a>

> Code samples

```http
GET /v3/payment-service-users HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`GET /v3/payment-service-users`

> Body parameter

```json
{}
```

<h3 id="list-all-payment-service-users-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|
|body|body|[V3QueryBuilder](#schemav3querybuilder)|false|none|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "data": [
      {
        "id": "string",
        "name": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "contactDetails": {
          "email": "string",
          "phoneNumber": "string"
        },
        "address": {
          "streetNumber": "string",
          "streetName": "string",
          "city": "string",
          "region": "string",
          "postalCode": "string",
          "country": "string"
        },
        "bankAccountIDs": [
          "string"
        ],
        "metadata": {
          "property1": "string",
          "property2": "string"
        }
      }
    ]
  }
}
```

<h3 id="list-all-payment-service-users-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3PaymentServiceUsersCursorResponse](#schemav3paymentserviceuserscursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Get a payment service user by ID

<a id="opIdv3GetPaymentServiceUser"></a>

> Code samples

```http
GET /v3/payment-service-users/{paymentServiceUserID} HTTP/1.1

Accept: application/json

```

`GET /v3/payment-service-users/{paymentServiceUserID}`

<h3 id="get-a-payment-service-user-by-id-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentServiceUserID|path|string|true|The payment service user ID|

> Example responses

> 200 Response

```json
{
  "data": {
    "id": "string",
    "name": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "contactDetails": {
      "email": "string",
      "phoneNumber": "string"
    },
    "address": {
      "streetNumber": "string",
      "streetName": "string",
      "city": "string",
      "region": "string",
      "postalCode": "string",
      "country": "string"
    },
    "bankAccountIDs": [
      "string"
    ],
    "metadata": {
      "property1": "string",
      "property2": "string"
    }
  }
}
```

<h3 id="get-a-payment-service-user-by-id-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3GetPaymentServiceUserResponse](#schemav3getpaymentserviceuserresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Delete a payment service user by ID

<a id="opIdv3DeletePaymentServiceUser"></a>

> Code samples

```http
DELETE /v3/payment-service-users/{paymentServiceUserID} HTTP/1.1

Accept: application/json

```

`DELETE /v3/payment-service-users/{paymentServiceUserID}`

<h3 id="delete-a-payment-service-user-by-id-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentServiceUserID|path|string|true|The payment service user ID|

> Example responses

> 202 Response

```json
{
  "data": {
    "taskID": "string"
  }
}
```

<h3 id="delete-a-payment-service-user-by-id-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|202|[Accepted](https://tools.ietf.org/html/rfc7231#section-6.3.3)|Accepted|[V3PaymentServiceUserDeleteResponse](#schemav3paymentserviceuserdeleteresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## List all connections for a payment service user

<a id="opIdv3ListPaymentServiceUserConnections"></a>

> Code samples

```http
GET /v3/payment-service-users/{paymentServiceUserID}/connections HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`GET /v3/payment-service-users/{paymentServiceUserID}/connections`

> Body parameter

```json
{}
```

<h3 id="list-all-connections-for-a-payment-service-user-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentServiceUserID|path|string|true|The payment service user ID|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|
|body|body|[V3QueryBuilder](#schemav3querybuilder)|false|none|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "data": [
      {
        "connectionID": "string",
        "connectorID": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "dataUpdatedAt": "2019-08-24T14:15:22Z",
        "status": "ACTIVE",
        "error": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        }
      }
    ]
  }
}
```

<h3 id="list-all-connections-for-a-payment-service-user-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3PaymentServiceUserConnectionsCursorResponse](#schemav3paymentserviceuserconnectionscursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Remove a payment service user from a connector, the PSU will still exist in Formance

<a id="opIdv3DeletePaymentServiceUserConnector"></a>

> Code samples

```http
DELETE /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID} HTTP/1.1

Accept: application/json

```

`DELETE /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}`

<h3 id="remove-a-payment-service-user-from-a-connector,-the-psu-will-still-exist-in-formance-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentServiceUserID|path|string|true|The payment service user ID|
|connectorID|path|string|true|The connector ID|

> Example responses

> 202 Response

```json
{
  "data": {
    "taskID": "string"
  }
}
```

<h3 id="remove-a-payment-service-user-from-a-connector,-the-psu-will-still-exist-in-formance-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|202|[Accepted](https://tools.ietf.org/html/rfc7231#section-6.3.3)|Accepted|[V3PaymentServiceUserDeleteConnectorResponse](#schemav3paymentserviceuserdeleteconnectorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Register/forward a payment service user on/to a connector

<a id="opIdv3ForwardPaymentServiceUserToProvider"></a>

> Code samples

```http
POST /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/forward HTTP/1.1

Accept: application/json

```

`POST /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/forward`

<h3 id="register/forward-a-payment-service-user-on/to-a-connector-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentServiceUserID|path|string|true|The payment service user ID|
|connectorID|path|string|true|The connector ID|

> Example responses

> default Response

```json
{
  "errorCode": "VALIDATION",
  "errorMessage": "[VALIDATION] missing required config field: pollingPeriod",
  "details": "string"
}
```

<h3 id="register/forward-a-payment-service-user-on/to-a-connector-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|No Content|None|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Create an authentication link for a payment service user on a connector, for oauth flow

<a id="opIdv3CreateLinkForPaymentServiceUser"></a>

> Code samples

```http
POST /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/create-link HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`POST /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/create-link`

> Body parameter

```json
{
  "applicationName": "string",
  "clientRedirectURL": "string"
}
```

<h3 id="create-an-authentication-link-for-a-payment-service-user-on-a-connector,-for-oauth-flow-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentServiceUserID|path|string|true|The payment service user ID|
|connectorID|path|string|true|The connector ID|
|body|body|[V3PaymentServiceUserCreateLinkRequest](#schemav3paymentserviceusercreatelinkrequest)|false|none|

> Example responses

> 201 Response

```json
{
  "attemptID": "string",
  "link": "string"
}
```

<h3 id="create-an-authentication-link-for-a-payment-service-user-on-a-connector,-for-oauth-flow-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|201|[Created](https://tools.ietf.org/html/rfc7231#section-6.3.2)|Created|[V3PaymentServiceUserCreateLinkResponse](#schemav3paymentserviceusercreatelinkresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## List enabled connections for a payment service user on a connector (i.e. the various banks PSUser has enabled on the connector)

<a id="opIdv3ListPaymentServiceUserConnectionsFromConnectorID"></a>

> Code samples

```http
GET /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/connections HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`GET /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/connections`

> Body parameter

```json
{}
```

<h3 id="list-enabled-connections-for-a-payment-service-user-on-a-connector-(i.e.-the-various-banks-psuser-has-enabled-on-the-connector)-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentServiceUserID|path|string|true|The payment service user ID|
|connectorID|path|string|true|The connector ID|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|
|body|body|[V3QueryBuilder](#schemav3querybuilder)|false|none|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "data": [
      {
        "connectionID": "string",
        "connectorID": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "dataUpdatedAt": "2019-08-24T14:15:22Z",
        "status": "ACTIVE",
        "error": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        }
      }
    ]
  }
}
```

<h3 id="list-enabled-connections-for-a-payment-service-user-on-a-connector-(i.e.-the-various-banks-psuser-has-enabled-on-the-connector)-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3PaymentServiceUserConnectionsCursorResponse](#schemav3paymentserviceuserconnectionscursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## List all link attempts for a payment service user on a connector.
Allows to check if users used the link and completed the oauth flow.

<a id="opIdv3ListPaymentServiceUserLinkAttemptsFromConnectorID"></a>

> Code samples

```http
GET /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/link-attempts HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`GET /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/link-attempts`

> Body parameter

```json
{}
```

<h3 id="list-all-link-attempts-for-a-payment-service-user-on-a-connector.
allows-to-check-if-users-used-the-link-and-completed-the-oauth-flow.
-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentServiceUserID|path|string|true|The payment service user ID|
|connectorID|path|string|true|The connector ID|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|
|body|body|[V3QueryBuilder](#schemav3querybuilder)|false|none|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "data": [
      {
        "id": "string",
        "psuID": "string",
        "connectorID": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "status": "pending",
        "clientRedirectURL": "string",
        "error": "string"
      }
    ]
  }
}
```

<h3 id="list-all-link-attempts-for-a-payment-service-user-on-a-connector.
allows-to-check-if-users-used-the-link-and-completed-the-oauth-flow.
-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3PaymentServiceUserLinkAttemptsCursorResponse](#schemav3paymentserviceuserlinkattemptscursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Get a link attempt for a payment service user on a connector

<a id="opIdv3GetPaymentServiceUserLinkAttemptFromConnectorID"></a>

> Code samples

```http
GET /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/link-attempts/{attemptID} HTTP/1.1

Accept: application/json

```

`GET /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/link-attempts/{attemptID}`

<h3 id="get-a-link-attempt-for-a-payment-service-user-on-a-connector-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentServiceUserID|path|string|true|The payment service user ID|
|connectorID|path|string|true|The connector ID|
|attemptID|path|string|true|The attempt ID|

> Example responses

> 200 Response

```json
{
  "id": "string",
  "psuID": "string",
  "connectorID": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "status": "pending",
  "clientRedirectURL": "string",
  "error": "string"
}
```

<h3 id="get-a-link-attempt-for-a-payment-service-user-on-a-connector-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3PaymentServiceUserLinkAttempt](#schemav3paymentserviceuserlinkattempt)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Delete a connection for a payment service user on a connector

<a id="opIdv3DeletePaymentServiceUserConnectionFromConnectorID"></a>

> Code samples

```http
DELETE /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/connections/{connectionID} HTTP/1.1

Accept: application/json

```

`DELETE /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/connections/{connectionID}`

<h3 id="delete-a-connection-for-a-payment-service-user-on-a-connector-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentServiceUserID|path|string|true|The payment service user ID|
|connectorID|path|string|true|The connector ID|
|connectionID|path|string|true|The connection ID|

> Example responses

> 202 Response

```json
{
  "data": {
    "taskID": "string"
  }
}
```

<h3 id="delete-a-connection-for-a-payment-service-user-on-a-connector-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|202|[Accepted](https://tools.ietf.org/html/rfc7231#section-6.3.3)|Accepted|[V3PaymentServiceUserDeleteConnectionResponse](#schemav3paymentserviceuserdeleteconnectionresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Update/Regenerate a link for a payment service user on a connector

<a id="opIdv3UpdateLinkForPaymentServiceUserOnConnector"></a>

> Code samples

```http
POST /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/connections/{connectionID}/update-link HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`POST /v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/connections/{connectionID}/update-link`

> Body parameter

```json
{
  "applicationName": "string",
  "clientRedirectURL": "string"
}
```

<h3 id="update/regenerate-a-link-for-a-payment-service-user-on-a-connector-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentServiceUserID|path|string|true|The payment service user ID|
|connectorID|path|string|true|The connector ID|
|connectionID|path|string|true|The connection ID|
|body|body|[V3PaymentServiceUserUpdateLinkRequest](#schemav3paymentserviceuserupdatelinkrequest)|false|none|

> Example responses

> 201 Response

```json
{
  "attemptID": "string",
  "link": "string"
}
```

<h3 id="update/regenerate-a-link-for-a-payment-service-user-on-a-connector-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|201|[Created](https://tools.ietf.org/html/rfc7231#section-6.3.2)|Created|[V3PaymentServiceUserUpdateLinkResponse](#schemav3paymentserviceuserupdatelinkresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Add a bank account to a payment service user

<a id="opIdv3AddBankAccountToPaymentServiceUser"></a>

> Code samples

```http
POST /v3/payment-service-users/{paymentServiceUserID}/bank-accounts/{bankAccountID} HTTP/1.1

Accept: application/json

```

`POST /v3/payment-service-users/{paymentServiceUserID}/bank-accounts/{bankAccountID}`

<h3 id="add-a-bank-account-to-a-payment-service-user-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentServiceUserID|path|string|true|The payment service user ID|
|bankAccountID|path|string|true|The bank account ID|

> Example responses

> default Response

```json
{
  "errorCode": "VALIDATION",
  "errorMessage": "[VALIDATION] missing required config field: pollingPeriod",
  "details": "string"
}
```

<h3 id="add-a-bank-account-to-a-payment-service-user-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|No Content|None|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Forward a payment service user's bank account to a connector

<a id="opIdv3ForwardPaymentServiceUserBankAccount"></a>

> Code samples

```http
POST /v3/payment-service-users/{paymentServiceUserID}/bank-accounts/{bankAccountID}/forward HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`POST /v3/payment-service-users/{paymentServiceUserID}/bank-accounts/{bankAccountID}/forward`

> Body parameter

```json
{
  "connectorID": "string"
}
```

<h3 id="forward-a-payment-service-user's-bank-account-to-a-connector-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|paymentServiceUserID|path|string|true|The payment service user ID|
|bankAccountID|path|string|true|The bank account ID|
|body|body|[V3ForwardPaymentServiceUserBankAccountRequest](#schemav3forwardpaymentserviceuserbankaccountrequest)|false|none|

> Example responses

> 202 Response

```json
{
  "data": {
    "taskID": "string"
  }
}
```

<h3 id="forward-a-payment-service-user's-bank-account-to-a-connector-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|202|[Accepted](https://tools.ietf.org/html/rfc7231#section-6.3.3)|Accepted|[V3ForwardPaymentServiceUserBankAccountResponse](#schemav3forwardpaymentserviceuserbankaccountresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Create a formance pool object

<a id="opIdv3CreatePool"></a>

> Code samples

```http
POST /v3/pools HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`POST /v3/pools`

> Body parameter

```json
{
  "name": "string",
  "accountIDs": [
    "string"
  ]
}
```

<h3 id="create-a-formance-pool-object-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[V3CreatePoolRequest](#schemav3createpoolrequest)|false|none|

> Example responses

> 201 Response

```json
{
  "data": "string"
}
```

<h3 id="create-a-formance-pool-object-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|201|[Created](https://tools.ietf.org/html/rfc7231#section-6.3.2)|Created|[V3CreatePoolResponse](#schemav3createpoolresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## List all pools

<a id="opIdv3ListPools"></a>

> Code samples

```http
GET /v3/pools HTTP/1.1

Content-Type: application/json
Accept: application/json

```

`GET /v3/pools`

> Body parameter

```json
{}
```

<h3 id="list-all-pools-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|pageSize|query|integer(int64)|false|The number of items to return|
|cursor|query|string|false|Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.|
|body|body|[V3QueryBuilder](#schemav3querybuilder)|false|none|

#### Detailed descriptions

**cursor**: Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.

> Example responses

> 200 Response

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "name": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "poolAccounts": [
          "string"
        ]
      }
    ]
  }
}
```

<h3 id="list-all-pools-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3PoolsCursorResponse](#schemav3poolscursorresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Get a pool by ID

<a id="opIdv3GetPool"></a>

> Code samples

```http
GET /v3/pools/{poolID} HTTP/1.1

Accept: application/json

```

`GET /v3/pools/{poolID}`

<h3 id="get-a-pool-by-id-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|poolID|path|string|true|The pool ID|

> Example responses

> 200 Response

```json
{
  "data": {
    "id": "string",
    "name": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "poolAccounts": [
      "string"
    ]
  }
}
```

<h3 id="get-a-pool-by-id-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3GetPoolResponse](#schemav3getpoolresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Delete a pool by ID

<a id="opIdv3DeletePool"></a>

> Code samples

```http
DELETE /v3/pools/{poolID} HTTP/1.1

Accept: application/json

```

`DELETE /v3/pools/{poolID}`

<h3 id="delete-a-pool-by-id-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|poolID|path|string|true|The pool ID|

> Example responses

> default Response

```json
{
  "errorCode": "VALIDATION",
  "errorMessage": "[VALIDATION] missing required config field: pollingPeriod",
  "details": "string"
}
```

<h3 id="delete-a-pool-by-id-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|No Content|None|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Get historical pool balances from a particular point in time

<a id="opIdv3GetPoolBalances"></a>

> Code samples

```http
GET /v3/pools/{poolID}/balances HTTP/1.1

Accept: application/json

```

`GET /v3/pools/{poolID}/balances`

<h3 id="get-historical-pool-balances-from-a-particular-point-in-time-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|poolID|path|string|true|The pool ID|
|at|query|string(date-time)|false|The time to filter by|

> Example responses

> 200 Response

```json
{
  "data": [
    {
      "asset": "string",
      "amount": 0
    }
  ]
}
```

<h3 id="get-historical-pool-balances-from-a-particular-point-in-time-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3PoolBalancesResponse](#schemav3poolbalancesresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Get latest pool balances

<a id="opIdv3GetPoolBalancesLatest"></a>

> Code samples

```http
GET /v3/pools/{poolID}/balances/latest HTTP/1.1

Accept: application/json

```

`GET /v3/pools/{poolID}/balances/latest`

<h3 id="get-latest-pool-balances-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|poolID|path|string|true|The pool ID|

> Example responses

> 200 Response

```json
{
  "data": [
    {
      "asset": "string",
      "amount": 0
    }
  ]
}
```

<h3 id="get-latest-pool-balances-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3PoolBalancesResponse](#schemav3poolbalancesresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

## Add an account to a pool

<a id="opIdv3AddAccountToPool"></a>

> Code samples

```http
POST /v3/pools/{poolID}/accounts/{accountID} HTTP/1.1

Accept: application/json

```

`POST /v3/pools/{poolID}/accounts/{accountID}`

<h3 id="add-an-account-to-a-pool-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|poolID|path|string|true|The pool ID|
|accountID|path|string|true|The account ID|

> Example responses

> default Response

```json
{
  "errorCode": "VALIDATION",
  "errorMessage": "[VALIDATION] missing required config field: pollingPeriod",
  "details": "string"
}
```

<h3 id="add-an-account-to-a-pool-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|No Content|None|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Remove an account from a pool

<a id="opIdv3RemoveAccountFromPool"></a>

> Code samples

```http
DELETE /v3/pools/{poolID}/accounts/{accountID} HTTP/1.1

Accept: application/json

```

`DELETE /v3/pools/{poolID}/accounts/{accountID}`

<h3 id="remove-an-account-from-a-pool-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|poolID|path|string|true|The pool ID|
|accountID|path|string|true|The account ID|

> Example responses

> default Response

```json
{
  "errorCode": "VALIDATION",
  "errorMessage": "[VALIDATION] missing required config field: pollingPeriod",
  "details": "string"
}
```

<h3 id="remove-an-account-from-a-pool-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|No Content|None|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:write )
</aside>

## Get a task and its result by ID

<a id="opIdv3GetTask"></a>

> Code samples

```http
GET /v3/tasks/{taskID} HTTP/1.1

Accept: application/json

```

`GET /v3/tasks/{taskID}`

<h3 id="get-a-task-and-its-result-by-id-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|taskID|path|string|true|The task ID|

> Example responses

> 200 Response

```json
{
  "data": {
    "id": "string",
    "status": "PROCESSING",
    "createdAt": "2019-08-24T14:15:22Z",
    "updatedAt": "2019-08-24T14:15:22Z",
    "connectorID": "string",
    "createdObjectID": "string",
    "error": "string"
  }
}
```

<h3 id="get-a-task-and-its-result-by-id-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[V3GetTaskResponse](#schemav3gettaskresponse)|
|default|Default|Error|[V3ErrorResponse](#schemav3errorresponse)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None ( Scopes: payments:read )
</aside>

# Schemas

<h2 id="tocS_V3AccountID">V3AccountID</h2>
<!-- backwards compatibility -->
<a id="schemav3accountid"></a>
<a id="schema_V3AccountID"></a>
<a id="tocSv3accountid"></a>
<a id="tocsv3accountid"></a>

```json
"string"

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|string|false|none|none|

<h2 id="tocS_V3AccountsCursorResponse">V3AccountsCursorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3accountscursorresponse"></a>
<a id="schema_V3AccountsCursorResponse"></a>
<a id="tocSv3accountscursorresponse"></a>
<a id="tocsv3accountscursorresponse"></a>

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "connectorID": "string",
        "provider": "string",
        "reference": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "type": "UNKNOWN",
        "name": "string",
        "defaultAsset": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        },
        "raw": {}
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|cursor|object|true|none|none|
|» pageSize|integer(int64)|true|none|none|
|» hasMore|boolean|true|none|none|
|» previous|string|false|none|none|
|» next|string|false|none|none|
|» data|[[V3Account](#schemav3account)]|true|none|none|

<h2 id="tocS_V3GetAccountResponse">V3GetAccountResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3getaccountresponse"></a>
<a id="schema_V3GetAccountResponse"></a>
<a id="tocSv3getaccountresponse"></a>
<a id="tocsv3getaccountresponse"></a>

```json
{
  "data": {
    "id": "string",
    "connectorID": "string",
    "provider": "string",
    "reference": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "type": "UNKNOWN",
    "name": "string",
    "defaultAsset": "string",
    "metadata": {
      "property1": "string",
      "property2": "string"
    },
    "raw": {}
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|[V3Account](#schemav3account)|true|none|none|

<h2 id="tocS_V3CreateAccountRequest">V3CreateAccountRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3createaccountrequest"></a>
<a id="schema_V3CreateAccountRequest"></a>
<a id="tocSv3createaccountrequest"></a>
<a id="tocsv3createaccountrequest"></a>

```json
{
  "reference": "string",
  "connectorID": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "accountName": "string",
  "type": "UNKNOWN",
  "defaultAsset": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|reference|string|true|none|none|
|connectorID|string(byte)|true|none|none|
|createdAt|string(date-time)|true|none|none|
|accountName|string|true|none|none|
|type|[V3AccountTypeEnum](#schemav3accounttypeenum)|true|none|none|
|defaultAsset|string¦null|false|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|

<h2 id="tocS_V3CreateAccountResponse">V3CreateAccountResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3createaccountresponse"></a>
<a id="schema_V3CreateAccountResponse"></a>
<a id="tocSv3createaccountresponse"></a>
<a id="tocsv3createaccountresponse"></a>

```json
{
  "data": {
    "id": "string",
    "connectorID": "string",
    "provider": "string",
    "reference": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "type": "UNKNOWN",
    "name": "string",
    "defaultAsset": "string",
    "metadata": {
      "property1": "string",
      "property2": "string"
    },
    "raw": {}
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|[V3Account](#schemav3account)|true|none|none|

<h2 id="tocS_V3Account">V3Account</h2>
<!-- backwards compatibility -->
<a id="schemav3account"></a>
<a id="schema_V3Account"></a>
<a id="tocSv3account"></a>
<a id="tocsv3account"></a>

```json
{
  "id": "string",
  "connectorID": "string",
  "provider": "string",
  "reference": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "type": "UNKNOWN",
  "name": "string",
  "defaultAsset": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  },
  "raw": {}
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|none|
|connectorID|string(byte)|true|none|none|
|provider|string|true|none|none|
|reference|string|true|none|none|
|createdAt|string(date-time)|true|none|none|
|type|[V3AccountTypeEnum](#schemav3accounttypeenum)|true|none|none|
|name|string¦null|false|none|none|
|defaultAsset|string¦null|false|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|
|raw|object|true|none|none|

<h2 id="tocS_V3AccountTypeEnum">V3AccountTypeEnum</h2>
<!-- backwards compatibility -->
<a id="schemav3accounttypeenum"></a>
<a id="schema_V3AccountTypeEnum"></a>
<a id="tocSv3accounttypeenum"></a>
<a id="tocsv3accounttypeenum"></a>

```json
"UNKNOWN"

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|string|false|none|none|

#### Enumerated Values

|Property|Value|
|---|---|
|*anonymous*|UNKNOWN|
|*anonymous*|INTERNAL|
|*anonymous*|EXTERNAL|

<h2 id="tocS_V3BalancesCursorResponse">V3BalancesCursorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3balancescursorresponse"></a>
<a id="schema_V3BalancesCursorResponse"></a>
<a id="tocSv3balancescursorresponse"></a>
<a id="tocsv3balancescursorresponse"></a>

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "accountID": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "lastUpdatedAt": "2019-08-24T14:15:22Z",
        "asset": "string",
        "balance": 0
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|cursor|object|true|none|none|
|» pageSize|integer(int64)|true|none|none|
|» hasMore|boolean|true|none|none|
|» previous|string|false|none|none|
|» next|string|false|none|none|
|» data|[[V3Balance](#schemav3balance)]|true|none|none|

<h2 id="tocS_V3Balance">V3Balance</h2>
<!-- backwards compatibility -->
<a id="schemav3balance"></a>
<a id="schema_V3Balance"></a>
<a id="tocSv3balance"></a>
<a id="tocsv3balance"></a>

```json
{
  "accountID": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "lastUpdatedAt": "2019-08-24T14:15:22Z",
  "asset": "string",
  "balance": 0
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|accountID|string|true|none|none|
|createdAt|string(date-time)|true|none|none|
|lastUpdatedAt|string(date-time)|true|none|none|
|asset|string|true|none|none|
|balance|integer(bigint)|true|none|none|

<h2 id="tocS_V3CreateBankAccountRequest">V3CreateBankAccountRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3createbankaccountrequest"></a>
<a id="schema_V3CreateBankAccountRequest"></a>
<a id="tocSv3createbankaccountrequest"></a>
<a id="tocsv3createbankaccountrequest"></a>

```json
{
  "name": "string",
  "accountNumber": "string",
  "iban": "string",
  "swiftBicCode": "string",
  "country": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|name|string|true|none|none|
|accountNumber|string|false|none|none|
|iban|string|false|none|none|
|swiftBicCode|string|false|none|none|
|country|string|false|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|

<h2 id="tocS_V3CreateBankAccountResponse">V3CreateBankAccountResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3createbankaccountresponse"></a>
<a id="schema_V3CreateBankAccountResponse"></a>
<a id="tocSv3createbankaccountresponse"></a>
<a id="tocsv3createbankaccountresponse"></a>

```json
{
  "data": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|string|true|none|The ID of the created bank account|

<h2 id="tocS_V3UpdateBankAccountMetadataRequest">V3UpdateBankAccountMetadataRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3updatebankaccountmetadatarequest"></a>
<a id="schema_V3UpdateBankAccountMetadataRequest"></a>
<a id="tocSv3updatebankaccountmetadatarequest"></a>
<a id="tocsv3updatebankaccountmetadatarequest"></a>

```json
{
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|metadata|[V3Metadata](#schemav3metadata)|true|none|none|

<h2 id="tocS_V3ForwardBankAccountRequest">V3ForwardBankAccountRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3forwardbankaccountrequest"></a>
<a id="schema_V3ForwardBankAccountRequest"></a>
<a id="tocSv3forwardbankaccountrequest"></a>
<a id="tocsv3forwardbankaccountrequest"></a>

```json
{
  "connectorID": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|connectorID|string(byte)|true|none|none|

<h2 id="tocS_V3ForwardBankAccountResponse">V3ForwardBankAccountResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3forwardbankaccountresponse"></a>
<a id="schema_V3ForwardBankAccountResponse"></a>
<a id="tocSv3forwardbankaccountresponse"></a>
<a id="tocsv3forwardbankaccountresponse"></a>

```json
{
  "data": {
    "taskID": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|object|true|none|none|
|» taskID|string|true|none|Since this call is asynchronous, the response will contain the ID of the task that was created to forward the bank account to the PSP. You can use the task API to check the status of the task and get the resulting bank account ID.|

<h2 id="tocS_V3BankAccountsCursorResponse">V3BankAccountsCursorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3bankaccountscursorresponse"></a>
<a id="schema_V3BankAccountsCursorResponse"></a>
<a id="tocSv3bankaccountscursorresponse"></a>
<a id="tocsv3bankaccountscursorresponse"></a>

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "name": "string",
        "accountNumber": "string",
        "iban": "string",
        "swiftBicCode": "string",
        "country": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        },
        "relatedAccounts": [
          {
            "accountID": "string",
            "createdAt": "2019-08-24T14:15:22Z"
          }
        ]
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|cursor|object|true|none|none|
|» pageSize|integer(int64)|true|none|none|
|» hasMore|boolean|true|none|none|
|» previous|string|false|none|none|
|» next|string|false|none|none|
|» data|[[V3BankAccount](#schemav3bankaccount)]|true|none|none|

<h2 id="tocS_V3GetBankAccountResponse">V3GetBankAccountResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3getbankaccountresponse"></a>
<a id="schema_V3GetBankAccountResponse"></a>
<a id="tocSv3getbankaccountresponse"></a>
<a id="tocsv3getbankaccountresponse"></a>

```json
{
  "data": {
    "id": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "name": "string",
    "accountNumber": "string",
    "iban": "string",
    "swiftBicCode": "string",
    "country": "string",
    "metadata": {
      "property1": "string",
      "property2": "string"
    },
    "relatedAccounts": [
      {
        "accountID": "string",
        "createdAt": "2019-08-24T14:15:22Z"
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|[V3BankAccount](#schemav3bankaccount)|true|none|none|

<h2 id="tocS_V3BankAccount">V3BankAccount</h2>
<!-- backwards compatibility -->
<a id="schemav3bankaccount"></a>
<a id="schema_V3BankAccount"></a>
<a id="tocSv3bankaccount"></a>
<a id="tocsv3bankaccount"></a>

```json
{
  "id": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "name": "string",
  "accountNumber": "string",
  "iban": "string",
  "swiftBicCode": "string",
  "country": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  },
  "relatedAccounts": [
    {
      "accountID": "string",
      "createdAt": "2019-08-24T14:15:22Z"
    }
  ]
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|none|
|createdAt|string(date-time)|true|none|none|
|name|string|true|none|none|
|accountNumber|string¦null|false|none|none|
|iban|string¦null|false|none|none|
|swiftBicCode|string¦null|false|none|none|
|country|string¦null|false|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|
|relatedAccounts|[[V3BankAccountRelatedAccount](#schemav3bankaccountrelatedaccount)]|false|none|none|

<h2 id="tocS_V3BankAccountRelatedAccount">V3BankAccountRelatedAccount</h2>
<!-- backwards compatibility -->
<a id="schemav3bankaccountrelatedaccount"></a>
<a id="schema_V3BankAccountRelatedAccount"></a>
<a id="tocSv3bankaccountrelatedaccount"></a>
<a id="tocsv3bankaccountrelatedaccount"></a>

```json
{
  "accountID": "string",
  "createdAt": "2019-08-24T14:15:22Z"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|accountID|string|true|none|none|
|createdAt|string(date-time)|true|none|none|

<h2 id="tocS_V3InstallConnectorRequest">V3InstallConnectorRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3installconnectorrequest"></a>
<a id="schema_V3InstallConnectorRequest"></a>
<a id="tocSv3installconnectorrequest"></a>
<a id="tocsv3installconnectorrequest"></a>

```json
{
  "apiKey": "string",
  "companyID": "string",
  "liveEndpointPrefix": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Adyen",
  "webhookPassword": "string",
  "webhookUsername": "string"
}

```

### Properties

*None*

<h2 id="tocS_V3InstallConnectorResponse">V3InstallConnectorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3installconnectorresponse"></a>
<a id="schema_V3InstallConnectorResponse"></a>
<a id="tocSv3installconnectorresponse"></a>
<a id="tocsv3installconnectorresponse"></a>

```json
{
  "data": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|string|true|none|The ID of the created connector|

<h2 id="tocS_V3UninstallConnectorResponse">V3UninstallConnectorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3uninstallconnectorresponse"></a>
<a id="schema_V3UninstallConnectorResponse"></a>
<a id="tocSv3uninstallconnectorresponse"></a>
<a id="tocsv3uninstallconnectorresponse"></a>

```json
{
  "data": {
    "taskID": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|object|true|none|none|
|» taskID|string|true|none|Since this call is asynchronous, the response will contain the ID of the task that was created to uninstall the connector. You can use the task API to check the status of the task and get the results.|

<h2 id="tocS_V3ResetConnectorResponse">V3ResetConnectorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3resetconnectorresponse"></a>
<a id="schema_V3ResetConnectorResponse"></a>
<a id="tocSv3resetconnectorresponse"></a>
<a id="tocsv3resetconnectorresponse"></a>

```json
{
  "data": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|string|true|none|Since this call is asynchronous, the response will contain the ID of the task that was created to reset the connector. You can use the task API to check the status of the task and get the results.|

<h2 id="tocS_V3ConnectorConfigsResponse">V3ConnectorConfigsResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3connectorconfigsresponse"></a>
<a id="schema_V3ConnectorConfigsResponse"></a>
<a id="tocSv3connectorconfigsresponse"></a>
<a id="tocsv3connectorconfigsresponse"></a>

```json
{
  "data": {
    "property1": {
      "property1": {
        "dataType": "string",
        "required": true,
        "defaultValue": "string"
      },
      "property2": {
        "dataType": "string",
        "required": true,
        "defaultValue": "string"
      }
    },
    "property2": {
      "property1": {
        "dataType": "string",
        "required": true,
        "defaultValue": "string"
      },
      "property2": {
        "dataType": "string",
        "required": true,
        "defaultValue": "string"
      }
    }
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|object|true|none|none|
|» **additionalProperties**|object|false|none|none|
|»» **additionalProperties**|object|false|none|none|
|»»» dataType|string|true|none|none|
|»»» required|boolean|true|none|none|
|»»» defaultValue|string|false|none|none|

<h2 id="tocS_V3GetConnectorConfigResponse">V3GetConnectorConfigResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3getconnectorconfigresponse"></a>
<a id="schema_V3GetConnectorConfigResponse"></a>
<a id="tocSv3getconnectorconfigresponse"></a>
<a id="tocsv3getconnectorconfigresponse"></a>

```json
{
  "data": {
    "apiKey": "string",
    "companyID": "string",
    "liveEndpointPrefix": "string",
    "name": "string",
    "pageSize": 25,
    "pollingPeriod": "2m",
    "provider": "Adyen",
    "webhookPassword": "string",
    "webhookUsername": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|[V3ConnectorConfig](#schemav3connectorconfig)|true|none|none|

<h2 id="tocS_V3UpdateConnectorRequest">V3UpdateConnectorRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3updateconnectorrequest"></a>
<a id="schema_V3UpdateConnectorRequest"></a>
<a id="tocSv3updateconnectorrequest"></a>
<a id="tocsv3updateconnectorrequest"></a>

```json
{
  "apiKey": "string",
  "companyID": "string",
  "liveEndpointPrefix": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Adyen",
  "webhookPassword": "string",
  "webhookUsername": "string"
}

```

### Properties

*None*

<h2 id="tocS_V3ConnectorsCursorResponse">V3ConnectorsCursorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3connectorscursorresponse"></a>
<a id="schema_V3ConnectorsCursorResponse"></a>
<a id="tocSv3connectorscursorresponse"></a>
<a id="tocsv3connectorscursorresponse"></a>

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "reference": "string",
        "name": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "provider": "string",
        "scheduledForDeletion": true,
        "config": {}
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|cursor|object|true|none|none|
|» pageSize|integer(int64)|true|none|none|
|» hasMore|boolean|true|none|none|
|» previous|string|false|none|none|
|» next|string|false|none|none|
|» data|[[V3Connector](#schemav3connector)]|true|none|none|

<h2 id="tocS_V3ConnectorSchedulesCursorResponse">V3ConnectorSchedulesCursorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3connectorschedulescursorresponse"></a>
<a id="schema_V3ConnectorSchedulesCursorResponse"></a>
<a id="tocSv3connectorschedulescursorresponse"></a>
<a id="tocsv3connectorschedulescursorresponse"></a>

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "connectorID": "string",
        "createdAt": "2019-08-24T14:15:22Z"
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|cursor|object|true|none|none|
|» pageSize|integer(int64)|true|none|none|
|» hasMore|boolean|true|none|none|
|» previous|string|false|none|none|
|» next|string|false|none|none|
|» data|[[V3Schedule](#schemav3schedule)]|true|none|none|

<h2 id="tocS_V3ConnectorScheduleResponse">V3ConnectorScheduleResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3connectorscheduleresponse"></a>
<a id="schema_V3ConnectorScheduleResponse"></a>
<a id="tocSv3connectorscheduleresponse"></a>
<a id="tocsv3connectorscheduleresponse"></a>

```json
{
  "data": {
    "id": "string",
    "connectorID": "string",
    "createdAt": "2019-08-24T14:15:22Z"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|[V3Schedule](#schemav3schedule)|true|none|none|

<h2 id="tocS_V3ConnectorScheduleInstancesCursorResponse">V3ConnectorScheduleInstancesCursorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3connectorscheduleinstancescursorresponse"></a>
<a id="schema_V3ConnectorScheduleInstancesCursorResponse"></a>
<a id="tocSv3connectorscheduleinstancescursorresponse"></a>
<a id="tocsv3connectorscheduleinstancescursorresponse"></a>

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "connectorID": "string",
        "scheduleID": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "updatedAt": "2019-08-24T14:15:22Z",
        "terminated": true,
        "terminatedAt": "2019-08-24T14:15:22Z",
        "error": "string"
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|cursor|object|true|none|none|
|» pageSize|integer(int64)|true|none|none|
|» hasMore|boolean|true|none|none|
|» previous|string|false|none|none|
|» next|string|false|none|none|
|» data|[[V3Instance](#schemav3instance)]|true|none|none|

<h2 id="tocS_V3Connector">V3Connector</h2>
<!-- backwards compatibility -->
<a id="schemav3connector"></a>
<a id="schema_V3Connector"></a>
<a id="tocSv3connector"></a>
<a id="tocsv3connector"></a>

```json
{
  "id": "string",
  "reference": "string",
  "name": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "provider": "string",
  "scheduledForDeletion": true,
  "config": {}
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|none|
|reference|string|true|none|none|
|name|string|true|none|none|
|createdAt|string(date-time)|true|none|none|
|provider|string|true|none|none|
|scheduledForDeletion|boolean|true|none|none|
|config|object|true|none|none|

<h2 id="tocS_V3Schedule">V3Schedule</h2>
<!-- backwards compatibility -->
<a id="schemav3schedule"></a>
<a id="schema_V3Schedule"></a>
<a id="tocSv3schedule"></a>
<a id="tocsv3schedule"></a>

```json
{
  "id": "string",
  "connectorID": "string",
  "createdAt": "2019-08-24T14:15:22Z"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|none|
|connectorID|string(byte)|true|none|none|
|createdAt|string(date-time)|true|none|none|

<h2 id="tocS_V3Instance">V3Instance</h2>
<!-- backwards compatibility -->
<a id="schemav3instance"></a>
<a id="schema_V3Instance"></a>
<a id="tocSv3instance"></a>
<a id="tocsv3instance"></a>

```json
{
  "id": "string",
  "connectorID": "string",
  "scheduleID": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "updatedAt": "2019-08-24T14:15:22Z",
  "terminated": true,
  "terminatedAt": "2019-08-24T14:15:22Z",
  "error": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|none|
|connectorID|string(byte)|true|none|none|
|scheduleID|string|true|none|none|
|createdAt|string(date-time)|true|none|none|
|updatedAt|string(date-time)|true|none|none|
|terminated|boolean|true|none|none|
|terminatedAt|string(date-time)|false|none|none|
|error|string¦null|false|none|none|

<h2 id="tocS_V3CreatePaymentRequest">V3CreatePaymentRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3createpaymentrequest"></a>
<a id="schema_V3CreatePaymentRequest"></a>
<a id="tocSv3createpaymentrequest"></a>
<a id="tocsv3createpaymentrequest"></a>

```json
{
  "reference": "string",
  "connectorID": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "type": "UNKNOWN",
  "initialAmount": 0,
  "amount": 0,
  "asset": "string",
  "scheme": "string",
  "sourceAccountID": "string",
  "destinationAccountID": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  },
  "adjustments": [
    {
      "reference": "string",
      "createdAt": "2019-08-24T14:15:22Z",
      "status": "UNKNOWN",
      "amount": 0,
      "asset": "string",
      "metadata": {
        "property1": "string",
        "property2": "string"
      }
    }
  ]
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|reference|string|true|none|none|
|connectorID|string(byte)|true|none|none|
|createdAt|string(date-time)|true|none|none|
|type|[V3PaymentTypeEnum](#schemav3paymenttypeenum)|true|none|none|
|initialAmount|integer(bigint)|true|none|none|
|amount|integer(bigint)|true|none|none|
|asset|string|true|none|none|
|scheme|string|true|none|none|
|sourceAccountID|string(byte)|false|none|none|
|destinationAccountID|string(byte)|false|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|
|adjustments|[[V3CreatePaymentAdjustmentRequest](#schemav3createpaymentadjustmentrequest)]|false|none|none|

<h2 id="tocS_V3CreatePaymentAdjustmentRequest">V3CreatePaymentAdjustmentRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3createpaymentadjustmentrequest"></a>
<a id="schema_V3CreatePaymentAdjustmentRequest"></a>
<a id="tocSv3createpaymentadjustmentrequest"></a>
<a id="tocsv3createpaymentadjustmentrequest"></a>

```json
{
  "reference": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "status": "UNKNOWN",
  "amount": 0,
  "asset": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|reference|string|true|none|none|
|createdAt|string(date-time)|true|none|none|
|status|[V3PaymentStatusEnum](#schemav3paymentstatusenum)|true|none|none|
|amount|integer(bigint)|false|none|none|
|asset|string|false|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|

<h2 id="tocS_V3CreatePaymentResponse">V3CreatePaymentResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3createpaymentresponse"></a>
<a id="schema_V3CreatePaymentResponse"></a>
<a id="tocSv3createpaymentresponse"></a>
<a id="tocsv3createpaymentresponse"></a>

```json
{
  "data": {
    "id": "string",
    "connectorID": "string",
    "provider": "string",
    "reference": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "type": "UNKNOWN",
    "initialAmount": 0,
    "amount": 0,
    "asset": "string",
    "scheme": "string",
    "status": "UNKNOWN",
    "sourceAccountID": "string",
    "destinationAccountID": "string",
    "metadata": {
      "property1": "string",
      "property2": "string"
    },
    "adjustments": [
      {
        "id": "string",
        "reference": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "status": "UNKNOWN",
        "amount": 0,
        "asset": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        },
        "raw": {}
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|[V3Payment](#schemav3payment)|true|none|none|

<h2 id="tocS_V3UpdatePaymentMetadataRequest">V3UpdatePaymentMetadataRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3updatepaymentmetadatarequest"></a>
<a id="schema_V3UpdatePaymentMetadataRequest"></a>
<a id="tocSv3updatepaymentmetadatarequest"></a>
<a id="tocsv3updatepaymentmetadatarequest"></a>

```json
{
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|metadata|[V3Metadata](#schemav3metadata)|true|none|none|

<h2 id="tocS_V3PaymentsCursorResponse">V3PaymentsCursorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentscursorresponse"></a>
<a id="schema_V3PaymentsCursorResponse"></a>
<a id="tocSv3paymentscursorresponse"></a>
<a id="tocsv3paymentscursorresponse"></a>

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "connectorID": "string",
        "provider": "string",
        "reference": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "type": "UNKNOWN",
        "initialAmount": 0,
        "amount": 0,
        "asset": "string",
        "scheme": "string",
        "status": "UNKNOWN",
        "sourceAccountID": "string",
        "destinationAccountID": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        },
        "adjustments": [
          {
            "id": "string",
            "reference": "string",
            "createdAt": "2019-08-24T14:15:22Z",
            "status": "UNKNOWN",
            "amount": 0,
            "asset": "string",
            "metadata": {
              "property1": "string",
              "property2": "string"
            },
            "raw": {}
          }
        ]
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|cursor|object|true|none|none|
|» pageSize|integer(int64)|true|none|none|
|» hasMore|boolean|true|none|none|
|» previous|string|false|none|none|
|» next|string|false|none|none|
|» data|[[V3Payment](#schemav3payment)]|true|none|none|

<h2 id="tocS_V3GetPaymentResponse">V3GetPaymentResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3getpaymentresponse"></a>
<a id="schema_V3GetPaymentResponse"></a>
<a id="tocSv3getpaymentresponse"></a>
<a id="tocsv3getpaymentresponse"></a>

```json
{
  "data": {
    "id": "string",
    "connectorID": "string",
    "provider": "string",
    "reference": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "type": "UNKNOWN",
    "initialAmount": 0,
    "amount": 0,
    "asset": "string",
    "scheme": "string",
    "status": "UNKNOWN",
    "sourceAccountID": "string",
    "destinationAccountID": "string",
    "metadata": {
      "property1": "string",
      "property2": "string"
    },
    "adjustments": [
      {
        "id": "string",
        "reference": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "status": "UNKNOWN",
        "amount": 0,
        "asset": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        },
        "raw": {}
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|[V3Payment](#schemav3payment)|true|none|none|

<h2 id="tocS_V3Payment">V3Payment</h2>
<!-- backwards compatibility -->
<a id="schemav3payment"></a>
<a id="schema_V3Payment"></a>
<a id="tocSv3payment"></a>
<a id="tocsv3payment"></a>

```json
{
  "id": "string",
  "connectorID": "string",
  "provider": "string",
  "reference": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "type": "UNKNOWN",
  "initialAmount": 0,
  "amount": 0,
  "asset": "string",
  "scheme": "string",
  "status": "UNKNOWN",
  "sourceAccountID": "string",
  "destinationAccountID": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  },
  "adjustments": [
    {
      "id": "string",
      "reference": "string",
      "createdAt": "2019-08-24T14:15:22Z",
      "status": "UNKNOWN",
      "amount": 0,
      "asset": "string",
      "metadata": {
        "property1": "string",
        "property2": "string"
      },
      "raw": {}
    }
  ]
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|none|
|connectorID|string(byte)|true|none|none|
|provider|string|true|none|none|
|reference|string|true|none|none|
|createdAt|string(date-time)|true|none|none|
|type|[V3PaymentTypeEnum](#schemav3paymenttypeenum)|true|none|none|
|initialAmount|integer(bigint)|true|none|none|
|amount|integer(bigint)|true|none|none|
|asset|string|true|none|none|
|scheme|string|true|none|none|
|status|[V3PaymentStatusEnum](#schemav3paymentstatusenum)|true|none|none|
|sourceAccountID|string(byte)¦null|false|none|none|
|destinationAccountID|string(byte)¦null|false|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|
|adjustments|[[V3PaymentAdjustment](#schemav3paymentadjustment)]¦null|false|none|none|

<h2 id="tocS_V3PaymentAdjustment">V3PaymentAdjustment</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentadjustment"></a>
<a id="schema_V3PaymentAdjustment"></a>
<a id="tocSv3paymentadjustment"></a>
<a id="tocsv3paymentadjustment"></a>

```json
{
  "id": "string",
  "reference": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "status": "UNKNOWN",
  "amount": 0,
  "asset": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  },
  "raw": {}
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|none|
|reference|string|true|none|none|
|createdAt|string(date-time)|true|none|none|
|status|[V3PaymentStatusEnum](#schemav3paymentstatusenum)|true|none|none|
|amount|integer(bigint)|false|none|none|
|asset|string|false|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|
|raw|object|true|none|none|

<h2 id="tocS_V3PaymentTypeEnum">V3PaymentTypeEnum</h2>
<!-- backwards compatibility -->
<a id="schemav3paymenttypeenum"></a>
<a id="schema_V3PaymentTypeEnum"></a>
<a id="tocSv3paymenttypeenum"></a>
<a id="tocsv3paymenttypeenum"></a>

```json
"UNKNOWN"

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|string|false|none|none|

#### Enumerated Values

|Property|Value|
|---|---|
|*anonymous*|UNKNOWN|
|*anonymous*|PAY-IN|
|*anonymous*|PAYOUT|
|*anonymous*|TRANSFER|
|*anonymous*|OTHER|

<h2 id="tocS_V3PaymentStatusEnum">V3PaymentStatusEnum</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentstatusenum"></a>
<a id="schema_V3PaymentStatusEnum"></a>
<a id="tocSv3paymentstatusenum"></a>
<a id="tocsv3paymentstatusenum"></a>

```json
"UNKNOWN"

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|string|false|none|none|

#### Enumerated Values

|Property|Value|
|---|---|
|*anonymous*|UNKNOWN|
|*anonymous*|PENDING|
|*anonymous*|SUCCEEDED|
|*anonymous*|CANCELLED|
|*anonymous*|FAILED|
|*anonymous*|EXPIRED|
|*anonymous*|REFUNDED|
|*anonymous*|REFUNDED_FAILURE|
|*anonymous*|REFUND_REVERSED|
|*anonymous*|DISPUTE|
|*anonymous*|DISPUTE_WON|
|*anonymous*|DISPUTE_LOST|
|*anonymous*|AMOUNT_ADJUSTEMENT|
|*anonymous*|AUTHORISATION|
|*anonymous*|CAPTURE|
|*anonymous*|CAPTURE_FAILED|
|*anonymous*|OTHER|

<h2 id="tocS_V3InitiatePaymentRequest">V3InitiatePaymentRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3initiatepaymentrequest"></a>
<a id="schema_V3InitiatePaymentRequest"></a>
<a id="tocSv3initiatepaymentrequest"></a>
<a id="tocsv3initiatepaymentrequest"></a>

```json
{
  "reference": "string",
  "scheduledAt": "2019-08-24T14:15:22Z",
  "connectorID": "string",
  "description": "string",
  "type": "UNKNOWN",
  "amount": 0,
  "asset": "string",
  "sourceAccountID": "string",
  "destinationAccountID": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|reference|string|true|none|none|
|scheduledAt|string(date-time)|true|none|none|
|connectorID|string(byte)|true|none|none|
|description|string|true|none|none|
|type|[V3PaymentInitiationTypeEnum](#schemav3paymentinitiationtypeenum)|true|none|none|
|amount|integer(bigint)|true|none|none|
|asset|string|true|none|none|
|sourceAccountID|string(byte)¦null|false|none|none|
|destinationAccountID|string(byte)|false|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|

<h2 id="tocS_V3InitiatePaymentResponse">V3InitiatePaymentResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3initiatepaymentresponse"></a>
<a id="schema_V3InitiatePaymentResponse"></a>
<a id="tocSv3initiatepaymentresponse"></a>
<a id="tocsv3initiatepaymentresponse"></a>

```json
{
  "data": {
    "paymentInitiationID": "string",
    "taskID": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|object|true|none|none|
|» paymentInitiationID|string|false|none|Related payment initiation object ID created.|
|» taskID|string|false|none|Will be filled if the noValidation query parameter is set to true. Since this call is asynchronous, the response will contain the ID of the task that was created to create the payment on the PSP. You can use the task API to check the status of the task and get the resulting payment ID|

<h2 id="tocS_V3RetryPaymentInitiationResponse">V3RetryPaymentInitiationResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3retrypaymentinitiationresponse"></a>
<a id="schema_V3RetryPaymentInitiationResponse"></a>
<a id="tocSv3retrypaymentinitiationresponse"></a>
<a id="tocsv3retrypaymentinitiationresponse"></a>

```json
{
  "data": {
    "taskID": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|object|true|none|none|
|» taskID|string|true|none|Since this call is asynchronous, the response will contain the ID of the task that was created to retry the payment initiation to the PSP. You can use the task API to check the status of the task and get the resulting payment ID.|

<h2 id="tocS_V3ApprovePaymentInitiationResponse">V3ApprovePaymentInitiationResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3approvepaymentinitiationresponse"></a>
<a id="schema_V3ApprovePaymentInitiationResponse"></a>
<a id="tocSv3approvepaymentinitiationresponse"></a>
<a id="tocsv3approvepaymentinitiationresponse"></a>

```json
{
  "data": {
    "taskID": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|object|true|none|none|
|» taskID|string|true|none|Since this call is asynchronous, the response will contain the ID of the task that was created to approve the payment initiation. You can use the task API to check the status of the task and get the resulting payment ID.|

<h2 id="tocS_V3ReversePaymentInitiationRequest">V3ReversePaymentInitiationRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3reversepaymentinitiationrequest"></a>
<a id="schema_V3ReversePaymentInitiationRequest"></a>
<a id="tocSv3reversepaymentinitiationrequest"></a>
<a id="tocsv3reversepaymentinitiationrequest"></a>

```json
{
  "reference": "string",
  "description": "string",
  "amount": 0,
  "asset": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|reference|string|true|none|none|
|description|string|true|none|none|
|amount|integer(bigint)|true|none|none|
|asset|string|true|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|

<h2 id="tocS_V3ReversePaymentInitiationResponse">V3ReversePaymentInitiationResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3reversepaymentinitiationresponse"></a>
<a id="schema_V3ReversePaymentInitiationResponse"></a>
<a id="tocSv3reversepaymentinitiationresponse"></a>
<a id="tocsv3reversepaymentinitiationresponse"></a>

```json
{
  "data": {
    "taskID": "string",
    "paymentInitiationReversalID": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|object|true|none|none|
|» taskID|string|false|none|Since this call is asynchronous, the response will contain the ID of the task that was created to reverse the payment initiation. You can use the task API to check the status of the task and get the resulting payment ID.|
|» paymentInitiationReversalID|string|false|none|Related payment initiation reversal object ID created.|

<h2 id="tocS_V3PaymentInitiationsCursorResponse">V3PaymentInitiationsCursorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentinitiationscursorresponse"></a>
<a id="schema_V3PaymentInitiationsCursorResponse"></a>
<a id="tocSv3paymentinitiationscursorresponse"></a>
<a id="tocsv3paymentinitiationscursorresponse"></a>

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "connectorID": "string",
        "provider": "string",
        "reference": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "scheduledAt": "2019-08-24T14:15:22Z",
        "description": "string",
        "type": "UNKNOWN",
        "amount": 0,
        "asset": "string",
        "status": "UNKNOWN",
        "sourceAccountID": "string",
        "destinationAccountID": "string",
        "error": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        }
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|cursor|object|true|none|none|
|» pageSize|integer(int64)|true|none|none|
|» hasMore|boolean|true|none|none|
|» previous|string|false|none|none|
|» next|string|false|none|none|
|» data|[[V3PaymentInitiation](#schemav3paymentinitiation)]|true|none|none|

<h2 id="tocS_V3PaymentInitiationAdjustmentsCursorResponse">V3PaymentInitiationAdjustmentsCursorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentinitiationadjustmentscursorresponse"></a>
<a id="schema_V3PaymentInitiationAdjustmentsCursorResponse"></a>
<a id="tocSv3paymentinitiationadjustmentscursorresponse"></a>
<a id="tocsv3paymentinitiationadjustmentscursorresponse"></a>

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "status": "UNKNOWN",
        "amount": 0,
        "asset": "string",
        "error": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        }
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|cursor|object|true|none|none|
|» pageSize|integer(int64)|true|none|none|
|» hasMore|boolean|true|none|none|
|» previous|string|false|none|none|
|» next|string|false|none|none|
|» data|[[V3PaymentInitiationAdjustment](#schemav3paymentinitiationadjustment)]|true|none|none|

<h2 id="tocS_V3PaymentInitiationRelatedPaymentsCursorResponse">V3PaymentInitiationRelatedPaymentsCursorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentinitiationrelatedpaymentscursorresponse"></a>
<a id="schema_V3PaymentInitiationRelatedPaymentsCursorResponse"></a>
<a id="tocSv3paymentinitiationrelatedpaymentscursorresponse"></a>
<a id="tocsv3paymentinitiationrelatedpaymentscursorresponse"></a>

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "connectorID": "string",
        "provider": "string",
        "reference": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "type": "UNKNOWN",
        "initialAmount": 0,
        "amount": 0,
        "asset": "string",
        "scheme": "string",
        "status": "UNKNOWN",
        "sourceAccountID": "string",
        "destinationAccountID": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        },
        "adjustments": [
          {
            "id": "string",
            "reference": "string",
            "createdAt": "2019-08-24T14:15:22Z",
            "status": "UNKNOWN",
            "amount": 0,
            "asset": "string",
            "metadata": {
              "property1": "string",
              "property2": "string"
            },
            "raw": {}
          }
        ]
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|cursor|object|true|none|none|
|» pageSize|integer(int64)|true|none|none|
|» hasMore|boolean|true|none|none|
|» previous|string|false|none|none|
|» next|string|false|none|none|
|» data|[[V3Payment](#schemav3payment)]|true|none|none|

<h2 id="tocS_V3PaymentInitiation">V3PaymentInitiation</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentinitiation"></a>
<a id="schema_V3PaymentInitiation"></a>
<a id="tocSv3paymentinitiation"></a>
<a id="tocsv3paymentinitiation"></a>

```json
{
  "id": "string",
  "connectorID": "string",
  "provider": "string",
  "reference": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "scheduledAt": "2019-08-24T14:15:22Z",
  "description": "string",
  "type": "UNKNOWN",
  "amount": 0,
  "asset": "string",
  "status": "UNKNOWN",
  "sourceAccountID": "string",
  "destinationAccountID": "string",
  "error": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|none|
|connectorID|string(byte)|true|none|none|
|provider|string|true|none|none|
|reference|string|true|none|none|
|createdAt|string(date-time)|true|none|none|
|scheduledAt|string(date-time)|true|none|none|
|description|string|true|none|none|
|type|[V3PaymentInitiationTypeEnum](#schemav3paymentinitiationtypeenum)|true|none|none|
|amount|integer(bigint)|true|none|none|
|asset|string|true|none|none|
|status|[V3PaymentInitiationStatusEnum](#schemav3paymentinitiationstatusenum)|true|none|none|
|sourceAccountID|string(byte)|false|none|none|
|destinationAccountID|string(byte)|false|none|none|
|error|string¦null|false|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|

<h2 id="tocS_V3PaymentInitiationAdjustment">V3PaymentInitiationAdjustment</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentinitiationadjustment"></a>
<a id="schema_V3PaymentInitiationAdjustment"></a>
<a id="tocSv3paymentinitiationadjustment"></a>
<a id="tocsv3paymentinitiationadjustment"></a>

```json
{
  "id": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "status": "UNKNOWN",
  "amount": 0,
  "asset": "string",
  "error": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|none|
|createdAt|string(date-time)|true|none|none|
|status|[V3PaymentInitiationStatusEnum](#schemav3paymentinitiationstatusenum)|true|none|none|
|amount|integer(bigint)|false|none|none|
|asset|string|false|none|none|
|error|string¦null|false|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|

<h2 id="tocS_V3GetPaymentInitiationResponse">V3GetPaymentInitiationResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3getpaymentinitiationresponse"></a>
<a id="schema_V3GetPaymentInitiationResponse"></a>
<a id="tocSv3getpaymentinitiationresponse"></a>
<a id="tocsv3getpaymentinitiationresponse"></a>

```json
{
  "data": {
    "id": "string",
    "connectorID": "string",
    "provider": "string",
    "reference": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "scheduledAt": "2019-08-24T14:15:22Z",
    "description": "string",
    "type": "UNKNOWN",
    "amount": 0,
    "asset": "string",
    "status": "UNKNOWN",
    "sourceAccountID": "string",
    "destinationAccountID": "string",
    "error": "string",
    "metadata": {
      "property1": "string",
      "property2": "string"
    }
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|[V3PaymentInitiation](#schemav3paymentinitiation)|true|none|none|

<h2 id="tocS_V3PaymentInitiationStatusEnum">V3PaymentInitiationStatusEnum</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentinitiationstatusenum"></a>
<a id="schema_V3PaymentInitiationStatusEnum"></a>
<a id="tocSv3paymentinitiationstatusenum"></a>
<a id="tocsv3paymentinitiationstatusenum"></a>

```json
"UNKNOWN"

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|string|false|none|none|

#### Enumerated Values

|Property|Value|
|---|---|
|*anonymous*|UNKNOWN|
|*anonymous*|WAITING_FOR_VALIDATION|
|*anonymous*|SCHEDULED_FOR_PROCESSING|
|*anonymous*|PROCESSING|
|*anonymous*|PROCESSED|
|*anonymous*|FAILED|
|*anonymous*|REJECTED|
|*anonymous*|REVERSE_PROCESSING|
|*anonymous*|REVERSE_FAILED|
|*anonymous*|REVERSED|

<h2 id="tocS_V3PaymentInitiationTypeEnum">V3PaymentInitiationTypeEnum</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentinitiationtypeenum"></a>
<a id="schema_V3PaymentInitiationTypeEnum"></a>
<a id="tocSv3paymentinitiationtypeenum"></a>
<a id="tocsv3paymentinitiationtypeenum"></a>

```json
"UNKNOWN"

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|string|false|none|none|

#### Enumerated Values

|Property|Value|
|---|---|
|*anonymous*|UNKNOWN|
|*anonymous*|TRANSFER|
|*anonymous*|PAYOUT|

<h2 id="tocS_V3CreatePaymentServiceUserRequest">V3CreatePaymentServiceUserRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3createpaymentserviceuserrequest"></a>
<a id="schema_V3CreatePaymentServiceUserRequest"></a>
<a id="tocSv3createpaymentserviceuserrequest"></a>
<a id="tocsv3createpaymentserviceuserrequest"></a>

```json
{
  "name": "string",
  "contactDetails": {
    "email": "string",
    "phoneNumber": "string"
  },
  "address": {
    "streetNumber": "string",
    "streetName": "string",
    "city": "string",
    "region": "string",
    "postalCode": "string",
    "country": "string"
  },
  "bankAccountIDs": [
    "string"
  ],
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|name|string|true|none|none|
|contactDetails|[V3ContactDetailsRequest](#schemav3contactdetailsrequest)|false|none|none|
|address|[V3AddressRequest](#schemav3addressrequest)|false|none|none|
|bankAccountIDs|[string]¦null|false|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|

<h2 id="tocS_V3AddressRequest">V3AddressRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3addressrequest"></a>
<a id="schema_V3AddressRequest"></a>
<a id="tocSv3addressrequest"></a>
<a id="tocsv3addressrequest"></a>

```json
{
  "streetNumber": "string",
  "streetName": "string",
  "city": "string",
  "region": "string",
  "postalCode": "string",
  "country": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|streetNumber|string|false|none|none|
|streetName|string|false|none|none|
|city|string|false|none|none|
|region|string|false|none|none|
|postalCode|string|false|none|none|
|country|string|false|none|none|

<h2 id="tocS_V3ContactDetailsRequest">V3ContactDetailsRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3contactdetailsrequest"></a>
<a id="schema_V3ContactDetailsRequest"></a>
<a id="tocSv3contactdetailsrequest"></a>
<a id="tocsv3contactdetailsrequest"></a>

```json
{
  "email": "string",
  "phoneNumber": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|email|string|false|none|none|
|phoneNumber|string|false|none|none|

<h2 id="tocS_V3CreatePaymentServiceUserResponse">V3CreatePaymentServiceUserResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3createpaymentserviceuserresponse"></a>
<a id="schema_V3CreatePaymentServiceUserResponse"></a>
<a id="tocSv3createpaymentserviceuserresponse"></a>
<a id="tocsv3createpaymentserviceuserresponse"></a>

```json
{
  "data": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|string|true|none|The ID of the created payment service user|

<h2 id="tocS_V3PaymentServiceUserDeleteResponse">V3PaymentServiceUserDeleteResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentserviceuserdeleteresponse"></a>
<a id="schema_V3PaymentServiceUserDeleteResponse"></a>
<a id="tocSv3paymentserviceuserdeleteresponse"></a>
<a id="tocsv3paymentserviceuserdeleteresponse"></a>

```json
{
  "data": {
    "taskID": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|object|true|none|none|
|» taskID|string|true|none|Since this call is asynchronous, the response will contain the ID of the task that was created to delete the payment service user. You can use the task API to check the status of the task.|

<h2 id="tocS_V3PaymentServiceUserDeleteConnectorResponse">V3PaymentServiceUserDeleteConnectorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentserviceuserdeleteconnectorresponse"></a>
<a id="schema_V3PaymentServiceUserDeleteConnectorResponse"></a>
<a id="tocSv3paymentserviceuserdeleteconnectorresponse"></a>
<a id="tocsv3paymentserviceuserdeleteconnectorresponse"></a>

```json
{
  "data": {
    "taskID": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|object|true|none|none|
|» taskID|string|true|none|Since this call is asynchronous, the response will contain the ID of the task that was created to delete the payment service user on the connector. You can use the task API to check the status of the task.|

<h2 id="tocS_V3PaymentServiceUserDeleteConnectionResponse">V3PaymentServiceUserDeleteConnectionResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentserviceuserdeleteconnectionresponse"></a>
<a id="schema_V3PaymentServiceUserDeleteConnectionResponse"></a>
<a id="tocSv3paymentserviceuserdeleteconnectionresponse"></a>
<a id="tocsv3paymentserviceuserdeleteconnectionresponse"></a>

```json
{
  "data": {
    "taskID": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|object|true|none|none|
|» taskID|string|true|none|Since this call is asynchronous, the response will contain the ID of the task that was created to delete the connection. You can use the task API to check the status of the task.|

<h2 id="tocS_V3PaymentServiceUsersCursorResponse">V3PaymentServiceUsersCursorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentserviceuserscursorresponse"></a>
<a id="schema_V3PaymentServiceUsersCursorResponse"></a>
<a id="tocSv3paymentserviceuserscursorresponse"></a>
<a id="tocsv3paymentserviceuserscursorresponse"></a>

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "data": [
      {
        "id": "string",
        "name": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "contactDetails": {
          "email": "string",
          "phoneNumber": "string"
        },
        "address": {
          "streetNumber": "string",
          "streetName": "string",
          "city": "string",
          "region": "string",
          "postalCode": "string",
          "country": "string"
        },
        "bankAccountIDs": [
          "string"
        ],
        "metadata": {
          "property1": "string",
          "property2": "string"
        }
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|cursor|object|true|none|none|
|» pageSize|integer(int64)|true|none|none|
|» hasMore|boolean|true|none|none|
|» previous|string|false|none|none|
|» next|string|false|none|none|
|» data|[[V3PaymentServiceUser](#schemav3paymentserviceuser)]|true|none|none|

<h2 id="tocS_V3PaymentServiceUserConnectionsCursorResponse">V3PaymentServiceUserConnectionsCursorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentserviceuserconnectionscursorresponse"></a>
<a id="schema_V3PaymentServiceUserConnectionsCursorResponse"></a>
<a id="tocSv3paymentserviceuserconnectionscursorresponse"></a>
<a id="tocsv3paymentserviceuserconnectionscursorresponse"></a>

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "data": [
      {
        "connectionID": "string",
        "connectorID": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "dataUpdatedAt": "2019-08-24T14:15:22Z",
        "status": "ACTIVE",
        "error": "string",
        "metadata": {
          "property1": "string",
          "property2": "string"
        }
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|cursor|object|true|none|none|
|» pageSize|integer(int64)|true|none|none|
|» hasMore|boolean|true|none|none|
|» previous|string|false|none|none|
|» next|string|false|none|none|
|» data|[[V3PaymentServiceUserConnection](#schemav3paymentserviceuserconnection)]|true|none|none|

<h2 id="tocS_V3PaymentServiceUserLinkAttemptsCursorResponse">V3PaymentServiceUserLinkAttemptsCursorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentserviceuserlinkattemptscursorresponse"></a>
<a id="schema_V3PaymentServiceUserLinkAttemptsCursorResponse"></a>
<a id="tocSv3paymentserviceuserlinkattemptscursorresponse"></a>
<a id="tocsv3paymentserviceuserlinkattemptscursorresponse"></a>

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "data": [
      {
        "id": "string",
        "psuID": "string",
        "connectorID": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "status": "pending",
        "clientRedirectURL": "string",
        "error": "string"
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|cursor|object|true|none|none|
|» pageSize|integer(int64)|true|none|none|
|» hasMore|boolean|true|none|none|
|» previous|string|false|none|none|
|» next|string|false|none|none|
|» data|[[V3PaymentServiceUserLinkAttempt](#schemav3paymentserviceuserlinkattempt)]|true|none|none|

<h2 id="tocS_V3PaymentServiceUser">V3PaymentServiceUser</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentserviceuser"></a>
<a id="schema_V3PaymentServiceUser"></a>
<a id="tocSv3paymentserviceuser"></a>
<a id="tocsv3paymentserviceuser"></a>

```json
{
  "id": "string",
  "name": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "contactDetails": {
    "email": "string",
    "phoneNumber": "string"
  },
  "address": {
    "streetNumber": "string",
    "streetName": "string",
    "city": "string",
    "region": "string",
    "postalCode": "string",
    "country": "string"
  },
  "bankAccountIDs": [
    "string"
  ],
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|none|
|name|string|true|none|none|
|createdAt|string(date-time)|true|none|none|
|contactDetails|[V3ContactDetails](#schemav3contactdetails)|false|none|none|
|address|[V3Address](#schemav3address)|false|none|none|
|bankAccountIDs|[string]¦null|false|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|

<h2 id="tocS_V3PaymentServiceUserConnection">V3PaymentServiceUserConnection</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentserviceuserconnection"></a>
<a id="schema_V3PaymentServiceUserConnection"></a>
<a id="tocSv3paymentserviceuserconnection"></a>
<a id="tocsv3paymentserviceuserconnection"></a>

```json
{
  "connectionID": "string",
  "connectorID": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "dataUpdatedAt": "2019-08-24T14:15:22Z",
  "status": "ACTIVE",
  "error": "string",
  "metadata": {
    "property1": "string",
    "property2": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|connectionID|string|true|none|none|
|connectorID|string|true|none|none|
|createdAt|string(date-time)|true|none|none|
|dataUpdatedAt|string(date-time)|true|none|none|
|status|[V3ConnectionStatusEnum](#schemav3connectionstatusenum)|true|none|none|
|error|string¦null|false|none|none|
|metadata|[V3Metadata](#schemav3metadata)|false|none|none|

<h2 id="tocS_V3PaymentServiceUserLinkAttempt">V3PaymentServiceUserLinkAttempt</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentserviceuserlinkattempt"></a>
<a id="schema_V3PaymentServiceUserLinkAttempt"></a>
<a id="tocSv3paymentserviceuserlinkattempt"></a>
<a id="tocsv3paymentserviceuserlinkattempt"></a>

```json
{
  "id": "string",
  "psuID": "string",
  "connectorID": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "status": "pending",
  "clientRedirectURL": "string",
  "error": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|none|
|psuID|string|true|none|none|
|connectorID|string|true|none|none|
|createdAt|string(date-time)|true|none|none|
|status|[V3PSUOpenBankingConnectionAttemptStatusEnum](#schemav3psuopenbankingconnectionattemptstatusenum)|true|none|none|
|clientRedirectURL|string(url)|true|none|none|
|error|string¦null|false|none|none|

<h2 id="tocS_V3ContactDetails">V3ContactDetails</h2>
<!-- backwards compatibility -->
<a id="schemav3contactdetails"></a>
<a id="schema_V3ContactDetails"></a>
<a id="tocSv3contactdetails"></a>
<a id="tocsv3contactdetails"></a>

```json
{
  "email": "string",
  "phoneNumber": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|email|string|false|none|none|
|phoneNumber|string|false|none|none|

<h2 id="tocS_V3Address">V3Address</h2>
<!-- backwards compatibility -->
<a id="schemav3address"></a>
<a id="schema_V3Address"></a>
<a id="tocSv3address"></a>
<a id="tocsv3address"></a>

```json
{
  "streetNumber": "string",
  "streetName": "string",
  "city": "string",
  "region": "string",
  "postalCode": "string",
  "country": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|streetNumber|string|false|none|none|
|streetName|string|false|none|none|
|city|string|false|none|none|
|region|string|false|none|none|
|postalCode|string|false|none|none|
|country|string|false|none|none|

<h2 id="tocS_V3GetPaymentServiceUserResponse">V3GetPaymentServiceUserResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3getpaymentserviceuserresponse"></a>
<a id="schema_V3GetPaymentServiceUserResponse"></a>
<a id="tocSv3getpaymentserviceuserresponse"></a>
<a id="tocsv3getpaymentserviceuserresponse"></a>

```json
{
  "data": {
    "id": "string",
    "name": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "contactDetails": {
      "email": "string",
      "phoneNumber": "string"
    },
    "address": {
      "streetNumber": "string",
      "streetName": "string",
      "city": "string",
      "region": "string",
      "postalCode": "string",
      "country": "string"
    },
    "bankAccountIDs": [
      "string"
    ],
    "metadata": {
      "property1": "string",
      "property2": "string"
    }
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|[V3PaymentServiceUser](#schemav3paymentserviceuser)|true|none|none|

<h2 id="tocS_V3ForwardPaymentServiceUserBankAccountRequest">V3ForwardPaymentServiceUserBankAccountRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3forwardpaymentserviceuserbankaccountrequest"></a>
<a id="schema_V3ForwardPaymentServiceUserBankAccountRequest"></a>
<a id="tocSv3forwardpaymentserviceuserbankaccountrequest"></a>
<a id="tocsv3forwardpaymentserviceuserbankaccountrequest"></a>

```json
{
  "connectorID": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|connectorID|string(byte)|true|none|none|

<h2 id="tocS_V3ForwardPaymentServiceUserBankAccountResponse">V3ForwardPaymentServiceUserBankAccountResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3forwardpaymentserviceuserbankaccountresponse"></a>
<a id="schema_V3ForwardPaymentServiceUserBankAccountResponse"></a>
<a id="tocSv3forwardpaymentserviceuserbankaccountresponse"></a>
<a id="tocsv3forwardpaymentserviceuserbankaccountresponse"></a>

```json
{
  "data": {
    "taskID": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|object|true|none|none|
|» taskID|string|true|none|Since this call is asynchronous, the response will contain the ID of the task that was created to forward the bank account to the PSP. You can use the task API to check the status of the task and get the resulting bank account ID.|

<h2 id="tocS_V3ConnectionStatusEnum">V3ConnectionStatusEnum</h2>
<!-- backwards compatibility -->
<a id="schemav3connectionstatusenum"></a>
<a id="schema_V3ConnectionStatusEnum"></a>
<a id="tocSv3connectionstatusenum"></a>
<a id="tocsv3connectionstatusenum"></a>

```json
"ACTIVE"

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|string|false|none|none|

#### Enumerated Values

|Property|Value|
|---|---|
|*anonymous*|ACTIVE|
|*anonymous*|ERROR|

<h2 id="tocS_V3PSUOpenBankingConnectionAttemptStatusEnum">V3PSUOpenBankingConnectionAttemptStatusEnum</h2>
<!-- backwards compatibility -->
<a id="schemav3psuopenbankingconnectionattemptstatusenum"></a>
<a id="schema_V3PSUOpenBankingConnectionAttemptStatusEnum"></a>
<a id="tocSv3psuopenbankingconnectionattemptstatusenum"></a>
<a id="tocsv3psuopenbankingconnectionattemptstatusenum"></a>

```json
"pending"

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|string|false|none|none|

#### Enumerated Values

|Property|Value|
|---|---|
|*anonymous*|pending|
|*anonymous*|completed|
|*anonymous*|exited|

<h2 id="tocS_V3PaymentServiceUserCreateLinkRequest">V3PaymentServiceUserCreateLinkRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentserviceusercreatelinkrequest"></a>
<a id="schema_V3PaymentServiceUserCreateLinkRequest"></a>
<a id="tocSv3paymentserviceusercreatelinkrequest"></a>
<a id="tocsv3paymentserviceusercreatelinkrequest"></a>

```json
{
  "applicationName": "string",
  "clientRedirectURL": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|applicationName|string|true|none|The name of the application to be displayed to the user when they click the link (depending on the open banking provider).|
|clientRedirectURL|string(url)|true|none|The URL to redirect the user to after the link flow is completed.|

<h2 id="tocS_V3PaymentServiceUserCreateLinkResponse">V3PaymentServiceUserCreateLinkResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentserviceusercreatelinkresponse"></a>
<a id="schema_V3PaymentServiceUserCreateLinkResponse"></a>
<a id="tocSv3paymentserviceusercreatelinkresponse"></a>
<a id="tocsv3paymentserviceusercreatelinkresponse"></a>

```json
{
  "attemptID": "string",
  "link": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|attemptID|string|true|none|none|
|link|string(url)|true|none|none|

<h2 id="tocS_V3PaymentServiceUserUpdateLinkRequest">V3PaymentServiceUserUpdateLinkRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentserviceuserupdatelinkrequest"></a>
<a id="schema_V3PaymentServiceUserUpdateLinkRequest"></a>
<a id="tocSv3paymentserviceuserupdatelinkrequest"></a>
<a id="tocsv3paymentserviceuserupdatelinkrequest"></a>

```json
{
  "applicationName": "string",
  "clientRedirectURL": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|applicationName|string|true|none|none|
|clientRedirectURL|string(url)|true|none|none|

<h2 id="tocS_V3PaymentServiceUserUpdateLinkResponse">V3PaymentServiceUserUpdateLinkResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3paymentserviceuserupdatelinkresponse"></a>
<a id="schema_V3PaymentServiceUserUpdateLinkResponse"></a>
<a id="tocSv3paymentserviceuserupdatelinkresponse"></a>
<a id="tocsv3paymentserviceuserupdatelinkresponse"></a>

```json
{
  "attemptID": "string",
  "link": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|attemptID|string|true|none|none|
|link|string(url)|true|none|none|

<h2 id="tocS_V3CreatePoolRequest">V3CreatePoolRequest</h2>
<!-- backwards compatibility -->
<a id="schemav3createpoolrequest"></a>
<a id="schema_V3CreatePoolRequest"></a>
<a id="tocSv3createpoolrequest"></a>
<a id="tocsv3createpoolrequest"></a>

```json
{
  "name": "string",
  "accountIDs": [
    "string"
  ]
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|name|string|true|none|none|
|accountIDs|[string]|true|none|none|

<h2 id="tocS_V3CreatePoolResponse">V3CreatePoolResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3createpoolresponse"></a>
<a id="schema_V3CreatePoolResponse"></a>
<a id="tocSv3createpoolresponse"></a>
<a id="tocsv3createpoolresponse"></a>

```json
{
  "data": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|string|true|none|The ID of the created pool|

<h2 id="tocS_V3PoolsCursorResponse">V3PoolsCursorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3poolscursorresponse"></a>
<a id="schema_V3PoolsCursorResponse"></a>
<a id="tocSv3poolscursorresponse"></a>
<a id="tocsv3poolscursorresponse"></a>

```json
{
  "cursor": {
    "pageSize": 15,
    "hasMore": false,
    "previous": "YXVsdCBhbmQgYSBtYXhpbXVtIG1heF9yZXN1bHRzLol=",
    "next": "",
    "data": [
      {
        "id": "string",
        "name": "string",
        "createdAt": "2019-08-24T14:15:22Z",
        "poolAccounts": [
          "string"
        ]
      }
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|cursor|object|true|none|none|
|» pageSize|integer(int64)|true|none|none|
|» hasMore|boolean|true|none|none|
|» previous|string|false|none|none|
|» next|string|false|none|none|
|» data|[[V3Pool](#schemav3pool)]|true|none|none|

<h2 id="tocS_V3GetPoolResponse">V3GetPoolResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3getpoolresponse"></a>
<a id="schema_V3GetPoolResponse"></a>
<a id="tocSv3getpoolresponse"></a>
<a id="tocsv3getpoolresponse"></a>

```json
{
  "data": {
    "id": "string",
    "name": "string",
    "createdAt": "2019-08-24T14:15:22Z",
    "poolAccounts": [
      "string"
    ]
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|[V3Pool](#schemav3pool)|true|none|none|

<h2 id="tocS_V3PoolBalancesResponse">V3PoolBalancesResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3poolbalancesresponse"></a>
<a id="schema_V3PoolBalancesResponse"></a>
<a id="tocSv3poolbalancesresponse"></a>
<a id="tocsv3poolbalancesresponse"></a>

```json
{
  "data": [
    {
      "asset": "string",
      "amount": 0
    }
  ]
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|[V3PoolBalances](#schemav3poolbalances)|true|none|none|

<h2 id="tocS_V3Pool">V3Pool</h2>
<!-- backwards compatibility -->
<a id="schemav3pool"></a>
<a id="schema_V3Pool"></a>
<a id="tocSv3pool"></a>
<a id="tocsv3pool"></a>

```json
{
  "id": "string",
  "name": "string",
  "createdAt": "2019-08-24T14:15:22Z",
  "poolAccounts": [
    "string"
  ]
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|none|
|name|string|true|none|none|
|createdAt|string(date-time)|true|none|none|
|poolAccounts|[[V3AccountID](#schemav3accountid)]|true|none|none|

<h2 id="tocS_V3PoolBalances">V3PoolBalances</h2>
<!-- backwards compatibility -->
<a id="schemav3poolbalances"></a>
<a id="schema_V3PoolBalances"></a>
<a id="tocSv3poolbalances"></a>
<a id="tocsv3poolbalances"></a>

```json
[
  {
    "asset": "string",
    "amount": 0
  }
]

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[[V3PoolBalance](#schemav3poolbalance)]|false|none|none|

<h2 id="tocS_V3PoolBalance">V3PoolBalance</h2>
<!-- backwards compatibility -->
<a id="schemav3poolbalance"></a>
<a id="schema_V3PoolBalance"></a>
<a id="tocSv3poolbalance"></a>
<a id="tocsv3poolbalance"></a>

```json
{
  "asset": "string",
  "amount": 0
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|asset|string|true|none|none|
|amount|integer(bigint)|true|none|none|

<h2 id="tocS_V3GetTaskResponse">V3GetTaskResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3gettaskresponse"></a>
<a id="schema_V3GetTaskResponse"></a>
<a id="tocSv3gettaskresponse"></a>
<a id="tocsv3gettaskresponse"></a>

```json
{
  "data": {
    "id": "string",
    "status": "PROCESSING",
    "createdAt": "2019-08-24T14:15:22Z",
    "updatedAt": "2019-08-24T14:15:22Z",
    "connectorID": "string",
    "createdObjectID": "string",
    "error": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|data|[V3Task](#schemav3task)|true|none|none|

<h2 id="tocS_V3Task">V3Task</h2>
<!-- backwards compatibility -->
<a id="schemav3task"></a>
<a id="schema_V3Task"></a>
<a id="tocSv3task"></a>
<a id="tocsv3task"></a>

```json
{
  "id": "string",
  "status": "PROCESSING",
  "createdAt": "2019-08-24T14:15:22Z",
  "updatedAt": "2019-08-24T14:15:22Z",
  "connectorID": "string",
  "createdObjectID": "string",
  "error": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|none|
|status|[V3TaskStatusEnum](#schemav3taskstatusenum)|true|none|none|
|createdAt|string(date-time)|true|none|none|
|updatedAt|string(date-time)|true|none|none|
|connectorID|string(byte)|false|none|none|
|createdObjectID|string|false|none|none|
|error|string¦null|false|none|none|

<h2 id="tocS_V3TaskStatusEnum">V3TaskStatusEnum</h2>
<!-- backwards compatibility -->
<a id="schemav3taskstatusenum"></a>
<a id="schema_V3TaskStatusEnum"></a>
<a id="tocSv3taskstatusenum"></a>
<a id="tocsv3taskstatusenum"></a>

```json
"PROCESSING"

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|string|false|none|none|

#### Enumerated Values

|Property|Value|
|---|---|
|*anonymous*|PROCESSING|
|*anonymous*|SUCCEEDED|
|*anonymous*|FAILED|

<h2 id="tocS_V3QueryBuilder">V3QueryBuilder</h2>
<!-- backwards compatibility -->
<a id="schemav3querybuilder"></a>
<a id="schema_V3QueryBuilder"></a>
<a id="tocSv3querybuilder"></a>
<a id="tocsv3querybuilder"></a>

```json
{}

```

### Properties

*None*

<h2 id="tocS_V3Metadata">V3Metadata</h2>
<!-- backwards compatibility -->
<a id="schemav3metadata"></a>
<a id="schema_V3Metadata"></a>
<a id="tocSv3metadata"></a>
<a id="tocsv3metadata"></a>

```json
{
  "property1": "string",
  "property2": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|**additionalProperties**|string|false|none|none|

<h2 id="tocS_V3ErrorResponse">V3ErrorResponse</h2>
<!-- backwards compatibility -->
<a id="schemav3errorresponse"></a>
<a id="schema_V3ErrorResponse"></a>
<a id="tocSv3errorresponse"></a>
<a id="tocsv3errorresponse"></a>

```json
{
  "errorCode": "VALIDATION",
  "errorMessage": "[VALIDATION] missing required config field: pollingPeriod",
  "details": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|errorCode|[V3ErrorsEnum](#schemav3errorsenum)|true|none|none|
|errorMessage|string|true|none|none|
|details|string|false|none|none|

<h2 id="tocS_V3ErrorsEnum">V3ErrorsEnum</h2>
<!-- backwards compatibility -->
<a id="schemav3errorsenum"></a>
<a id="schema_V3ErrorsEnum"></a>
<a id="tocSv3errorsenum"></a>
<a id="tocsv3errorsenum"></a>

```json
"VALIDATION"

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|string|false|none|none|

#### Enumerated Values

|Property|Value|
|---|---|
|*anonymous*|INTERNAL|
|*anonymous*|VALIDATION|
|*anonymous*|INVALID_ID|
|*anonymous*|MISSING_OR_INVALID_BODY|
|*anonymous*|CONFLICT|
|*anonymous*|NOT_FOUND|

<h2 id="tocS_V3ConnectorConfig">V3ConnectorConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3connectorconfig"></a>
<a id="schema_V3ConnectorConfig"></a>
<a id="tocSv3connectorconfig"></a>
<a id="tocsv3connectorconfig"></a>

```json
{
  "apiKey": "string",
  "companyID": "string",
  "liveEndpointPrefix": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Adyen",
  "webhookPassword": "string",
  "webhookUsername": "string"
}

```

### Properties

oneOf

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3AdyenConfig](#schemav3adyenconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3AtlarConfig](#schemav3atlarconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3BankingcircleConfig](#schemav3bankingcircleconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3ColumnConfig](#schemav3columnconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3CurrencycloudConfig](#schemav3currencycloudconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3DummypayConfig](#schemav3dummypayconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3GenericConfig](#schemav3genericconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3IncreaseConfig](#schemav3increaseconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3MangopayConfig](#schemav3mangopayconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3ModulrConfig](#schemav3modulrconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3MoneycorpConfig](#schemav3moneycorpconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3PlaidConfig](#schemav3plaidconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3PowensConfig](#schemav3powensconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3QontoConfig](#schemav3qontoconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3StripeConfig](#schemav3stripeconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3TinkConfig](#schemav3tinkconfig)|false|none|none|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[V3WiseConfig](#schemav3wiseconfig)|false|none|none|

<h2 id="tocS_V3AdyenConfig">V3AdyenConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3adyenconfig"></a>
<a id="schema_V3AdyenConfig"></a>
<a id="tocSv3adyenconfig"></a>
<a id="tocsv3adyenconfig"></a>

```json
{
  "apiKey": "string",
  "companyID": "string",
  "liveEndpointPrefix": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Adyen",
  "webhookPassword": "string",
  "webhookUsername": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|apiKey|string|true|none|none|
|companyID|string|true|none|none|
|liveEndpointPrefix|string|false|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|
|webhookPassword|string|false|none|none|
|webhookUsername|string|false|none|none|

<h2 id="tocS_V3AtlarConfig">V3AtlarConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3atlarconfig"></a>
<a id="schema_V3AtlarConfig"></a>
<a id="tocSv3atlarconfig"></a>
<a id="tocsv3atlarconfig"></a>

```json
{
  "accessKey": "string",
  "baseUrl": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Atlar",
  "secret": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|accessKey|string|true|none|none|
|baseUrl|string|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|
|secret|string|true|none|none|

<h2 id="tocS_V3BankingcircleConfig">V3BankingcircleConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3bankingcircleconfig"></a>
<a id="schema_V3BankingcircleConfig"></a>
<a id="tocSv3bankingcircleconfig"></a>
<a id="tocsv3bankingcircleconfig"></a>

```json
{
  "authorizationEndpoint": "string",
  "endpoint": "string",
  "name": "string",
  "pageSize": 25,
  "password": "string",
  "pollingPeriod": "2m",
  "provider": "Bankingcircle",
  "userCertificate": "string",
  "userCertificateKey": "string",
  "username": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|authorizationEndpoint|string|true|none|none|
|endpoint|string|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|password|string|true|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|
|userCertificate|string|true|none|none|
|userCertificateKey|string|true|none|none|
|username|string|true|none|none|

<h2 id="tocS_V3ColumnConfig">V3ColumnConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3columnconfig"></a>
<a id="schema_V3ColumnConfig"></a>
<a id="tocSv3columnconfig"></a>
<a id="tocsv3columnconfig"></a>

```json
{
  "apiKey": "string",
  "endpoint": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Column"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|apiKey|string|true|none|none|
|endpoint|string|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|

<h2 id="tocS_V3CurrencycloudConfig">V3CurrencycloudConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3currencycloudconfig"></a>
<a id="schema_V3CurrencycloudConfig"></a>
<a id="tocSv3currencycloudconfig"></a>
<a id="tocsv3currencycloudconfig"></a>

```json
{
  "apiKey": "string",
  "endpoint": "string",
  "loginID": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Currencycloud"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|apiKey|string|true|none|none|
|endpoint|string|true|none|none|
|loginID|string|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|

<h2 id="tocS_V3DummypayConfig">V3DummypayConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3dummypayconfig"></a>
<a id="schema_V3DummypayConfig"></a>
<a id="tocSv3dummypayconfig"></a>
<a id="tocsv3dummypayconfig"></a>

```json
{
  "directory": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Dummypay"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|directory|string|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|

<h2 id="tocS_V3GenericConfig">V3GenericConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3genericconfig"></a>
<a id="schema_V3GenericConfig"></a>
<a id="tocSv3genericconfig"></a>
<a id="tocsv3genericconfig"></a>

```json
{
  "apiKey": "string",
  "endpoint": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Generic"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|apiKey|string|true|none|none|
|endpoint|string|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|

<h2 id="tocS_V3IncreaseConfig">V3IncreaseConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3increaseconfig"></a>
<a id="schema_V3IncreaseConfig"></a>
<a id="tocSv3increaseconfig"></a>
<a id="tocsv3increaseconfig"></a>

```json
{
  "apiKey": "string",
  "endpoint": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Increase",
  "webhookSharedSecret": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|apiKey|string|true|none|none|
|endpoint|string|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|
|webhookSharedSecret|string|true|none|none|

<h2 id="tocS_V3MangopayConfig">V3MangopayConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3mangopayconfig"></a>
<a id="schema_V3MangopayConfig"></a>
<a id="tocSv3mangopayconfig"></a>
<a id="tocsv3mangopayconfig"></a>

```json
{
  "apiKey": "string",
  "clientID": "string",
  "endpoint": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Mangopay"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|apiKey|string|true|none|none|
|clientID|string|true|none|none|
|endpoint|string|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|

<h2 id="tocS_V3ModulrConfig">V3ModulrConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3modulrconfig"></a>
<a id="schema_V3ModulrConfig"></a>
<a id="tocSv3modulrconfig"></a>
<a id="tocsv3modulrconfig"></a>

```json
{
  "apiKey": "string",
  "apiSecret": "string",
  "endpoint": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Modulr"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|apiKey|string|true|none|none|
|apiSecret|string|true|none|none|
|endpoint|string|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|

<h2 id="tocS_V3MoneycorpConfig">V3MoneycorpConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3moneycorpconfig"></a>
<a id="schema_V3MoneycorpConfig"></a>
<a id="tocSv3moneycorpconfig"></a>
<a id="tocsv3moneycorpconfig"></a>

```json
{
  "apiKey": "string",
  "clientID": "string",
  "endpoint": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Moneycorp"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|apiKey|string|true|none|none|
|clientID|string|true|none|none|
|endpoint|string|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|

<h2 id="tocS_V3PlaidConfig">V3PlaidConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3plaidconfig"></a>
<a id="schema_V3PlaidConfig"></a>
<a id="tocSv3plaidconfig"></a>
<a id="tocsv3plaidconfig"></a>

```json
{
  "clientID": "string",
  "clientSecret": "string",
  "isSandbox": true,
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Plaid"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|clientID|string|true|none|none|
|clientSecret|string|true|none|none|
|isSandbox|boolean|false|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|

<h2 id="tocS_V3PowensConfig">V3PowensConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3powensconfig"></a>
<a id="schema_V3PowensConfig"></a>
<a id="tocSv3powensconfig"></a>
<a id="tocsv3powensconfig"></a>

```json
{
  "clientID": "string",
  "clientSecret": "string",
  "configurationToken": "string",
  "domain": "string",
  "endpoint": "string",
  "maxConnectionsPerLink": 0,
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Powens"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|clientID|string|true|none|none|
|clientSecret|string|true|none|none|
|configurationToken|string|true|none|none|
|domain|string|true|none|none|
|endpoint|string|true|none|none|
|maxConnectionsPerLink|integer|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|

<h2 id="tocS_V3QontoConfig">V3QontoConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3qontoconfig"></a>
<a id="schema_V3QontoConfig"></a>
<a id="tocSv3qontoconfig"></a>
<a id="tocsv3qontoconfig"></a>

```json
{
  "apiKey": "string",
  "clientID": "string",
  "endpoint": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Qonto",
  "stagingToken": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|apiKey|string|true|none|none|
|clientID|string|true|none|none|
|endpoint|string|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|
|stagingToken|string|false|none|none|

<h2 id="tocS_V3StripeConfig">V3StripeConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3stripeconfig"></a>
<a id="schema_V3StripeConfig"></a>
<a id="tocSv3stripeconfig"></a>
<a id="tocsv3stripeconfig"></a>

```json
{
  "apiKey": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Stripe"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|apiKey|string|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|

<h2 id="tocS_V3TinkConfig">V3TinkConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3tinkconfig"></a>
<a id="schema_V3TinkConfig"></a>
<a id="tocSv3tinkconfig"></a>
<a id="tocsv3tinkconfig"></a>

```json
{
  "clientID": "string",
  "clientSecret": "string",
  "endpoint": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Tink"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|clientID|string|true|none|none|
|clientSecret|string|true|none|none|
|endpoint|string|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|

<h2 id="tocS_V3WiseConfig">V3WiseConfig</h2>
<!-- backwards compatibility -->
<a id="schemav3wiseconfig"></a>
<a id="schema_V3WiseConfig"></a>
<a id="tocSv3wiseconfig"></a>
<a id="tocsv3wiseconfig"></a>

```json
{
  "apiKey": "string",
  "name": "string",
  "pageSize": 25,
  "pollingPeriod": "2m",
  "provider": "Wise",
  "webhookPublicKey": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|apiKey|string|true|none|none|
|name|string|true|none|none|
|pageSize|integer|false|none|none|
|pollingPeriod|string|false|none|none|
|provider|string|false|none|none|
|webhookPublicKey|string|true|none|none|

