# PRD: Exchange Connectivity

## Introduction

Formance clients increasingly need to manage crypto assets alongside their fiat currencies. Today, they must manually manage their crypto exchanges outside of Formance, creating operational complexity and fragmented visibility.

This feature integrates crypto exchanges (Kraken, Coinbase Prime, Bitstamp, Binance) as new PSPs in the Payment Service, enabling unified fiat + crypto management. It introduces a new **Order** primitive for trading operations and a **Conversion** primitive for stablecoin exchanges, while ensuring MiCA compliance for European clients.

## Goals

- Connect crypto exchanges (Coinbase Prime, Kraken, Bitstamp, Binance) as payment connectors
- Sync balances, accounts/wallets, and transactions from exchanges
- Enable trading via a new `Order` primitive supporting MARKET and LIMIT order types
- Support stablecoin conversions via a separate `Conversion` primitive
- Provide order lifecycle management (create, list, get, cancel, fetch/sync)
- Deliver market data capabilities (order book, quotes, ticker info, tradable assets, OHLC)
- Ensure MiCA compliance for European clients
- Start with REST API integration, with WebSocket support as a later enhancement

## User Stories

### Phase 1: Core Infrastructure & Coinbase Connector

#### US-001: Define Order primitive data model
**Description:** As a developer, I need to define the Order primitive schema so that orders can be stored and tracked consistently across all connectors.

**Acceptance Criteria:**
- [x] Order model includes: id, reference, connectorId, direction (BUY/SELL), sourceAsset, targetAsset, status, type, baseQuantityOrdered, baseQuantityFilled, timeInForce, fee, createdAt, updatedAt
- [x] Status enum includes: PENDING, OPEN, PARTIALLY_FILLED, FILLED, CANCELLED, FAILED, EXPIRED
- [x] Type enum includes: MARKET, LIMIT
- [x] TimeInForce enum includes: GOOD_UNTIL_CANCELLED, GOOD_UNTIL_DATE_TIME, IMMEDIATE_OR_CANCEL, FILL_OR_KILL
- [x] Quantities use integer representation (UMN) to avoid decimal precision issues
- [x] Database migration created and tested
- [x] Typecheck/lint passes

#### US-002: Define Conversion primitive data model
**Description:** As a developer, I need to define the Conversion primitive schema so that stablecoin conversions (USD↔USDC, USD↔PYUSD) can be tracked separately from trading orders.

**Acceptance Criteria:**
- [x] Conversion model includes: id, reference, connectorId, sourceAsset, targetAsset, sourceAmount, targetAmount, status, walletId, createdAt, updatedAt
- [x] Status enum includes: PENDING, COMPLETED, FAILED
- [x] Database migration created and tested
- [x] Typecheck/lint passes

#### US-003: Implement Coinbase Prime connector configuration
**Description:** As an operator, I want to configure a Coinbase Prime connector so that I can connect my Coinbase Prime portfolio to Formance.

**Acceptance Criteria:**
- [x] Connector accepts credentials JSON containing: accessKey, passphrase, signingKey, portfolioId, svcAccountId, entityId
- [x] Credentials are securely stored and encrypted at rest
- [x] Connector validates credentials on installation by making a test API call
- [x] One connector per portfolio (respects Coinbase's segregation model)
- [x] Typecheck/lint passes

#### US-004: Fetch accounts/wallets from Coinbase Prime
**Description:** As a user, I want to sync my Coinbase Prime wallets so that I can see all my crypto accounts in Formance.

**Acceptance Criteria:**
- [x] Connector implements CAPABILITY_FETCH_ACCOUNTS
- [x] Wallets are mapped to Formance accounts with reference = wallet.id
- [x] Wallet types (VAULT, TRADING, WALLET_TYPE_OTHER, QC, ONCHAIN) are preserved as metadata
- [x] Wallet visibility status is preserved
- [x] Network information (bitcoin, ethereum, solana, etc.) is preserved as metadata
- [x] Periodic sync runs on configured polling interval
- [x] Typecheck/lint passes

#### US-005: Fetch balances from Coinbase Prime
**Description:** As a user, I want to sync my Coinbase Prime balances so that I can see my crypto holdings in Formance.

**Acceptance Criteria:**
- [x] Connector implements CAPABILITY_FETCH_BALANCES
- [x] Balance mapping includes: symbol, amount, holds, withdrawable_amount
- [x] Bonded/staking amounts are tracked as metadata
- [x] Periodic sync runs on configured polling interval
- [x] Typecheck/lint passes

#### US-006: Fetch orders from Coinbase Prime
**Description:** As a user, I want to sync my Coinbase Prime orders so that orders placed outside Formance are visible.

**Acceptance Criteria:**
- [x] Connector implements CAPABILITY_FETCH_ORDERS
- [x] Uses ListPortfolioOrders for closed orders and ListOpenOrders for open orders
- [x] Handles pagination (limit 3000 for closed, 1000 for open)
- [x] Uses sort_direction=ASCENDING with start_date filter for incremental sync
- [x] Maps Coinbase status to Formance status (OPEN→OPEN, FILLED→FILLED, CANCELLED→CANCELLED, EXPIRED→EXPIRED, PENDING→PENDING)
- [x] Detects PARTIALLY_FILLED when filled_quantity < base_quantity
- [x] Periodic sync runs on configured polling interval
- [x] Typecheck/lint passes

#### US-007: Get order by ID API endpoint
**Description:** As a user, I want to retrieve a specific order by its ID so that I can check its current status.

**Acceptance Criteria:**
- [x] Endpoint: GET /payments/v3/orders/:orderID
- [x] Returns full order DTO with all fields
- [x] Returns 404 if order not found
- [x] Typecheck/lint passes

#### US-008: List orders API endpoint
**Description:** As a user, I want to list and filter orders so that I can find specific orders or view order history.

**Acceptance Criteria:**
- [x] Endpoint: GET /payments/v3/orders
- [x] Supports filtering by: reference, id, connectorID, status, type, sourceAsset, targetAsset, metadata
- [x] Returns cursored list of order DTOs
- [x] Supports pagination
- [x] Typecheck/lint passes

### Phase 2: Kraken Connector & LIMIT Orders

#### US-009: Implement Kraken connector configuration
**Description:** As an operator, I want to configure a Kraken connector so that I can connect my Kraken account to Formance.

**Acceptance Criteria:**
- [x] Connector accepts: endpoint (api.kraken.com or UAT), publicKey, privateKey
- [x] Uses official Kraken Go SDK (github.com/krakenfx/api-go/v2) which handles API-Sign authentication internally
- [x] Credentials are securely stored and encrypted at rest
- [x] Connector validates credentials on installation
- [x] Typecheck/lint passes

#### US-010: Fetch accounts/balances from Kraken
**Description:** As a user, I want to sync my Kraken accounts and balances so that I can see my holdings in Formance.

**Acceptance Criteria:**
- [x] Connector implements CAPABILITY_FETCH_ACCOUNTS and CAPABILITY_FETCH_BALANCES
- [x] Uses /0/private/Balance endpoint
- [x] Respects Kraken rate limits
- [x] Periodic sync runs on configured polling interval
- [x] Typecheck/lint passes

#### US-011: Fetch orders from Kraken
**Description:** As a user, I want to sync my Kraken orders so that orders placed outside Formance are visible.

**Acceptance Criteria:**
- [x] Connector implements CAPABILITY_FETCH_ORDERS
- [x] Uses /0/private/ClosedOrders endpoint
- [x] Maps Kraken fields: refid→reference, cl_ord_id→id, vol→baseQuantityOrdered, vol_exec→baseQuantityFilled, fee→fee
- [x] Maps Kraken status: open→OPEN, closed→FILLED, canceled→CANCELLED, expired→EXPIRED, pending→PENDING
- [x] Respects Kraken rate limits
- [x] Typecheck/lint passes

#### US-012: Create LIMIT order API endpoint
**Description:** As a user, I want to create a LIMIT order so that I can buy or sell crypto at a specified price.

**Acceptance Criteria:**
- [x] Endpoint: POST /payments/v3/orders
- [x] Request includes: connectorId, direction (BUY/SELL), sourceAsset, targetAsset, type=LIMIT, quantity (sourceQuantity or targetQuantity), limitPrice
- [x] Supports quantitySide (SOURCE/TARGET) to specify which quantity is fixed
- [x] Quantities use integer representation with asset precision (e.g., BTC/8 for 8 decimals)
- [x] Order is forwarded to connector via OrderHook
- [x] Returns created order with PENDING status
- [x] Typecheck/lint passes

#### US-013: Implement LIMIT order for Coinbase Prime
**Description:** As a developer, I need to implement LIMIT order creation for Coinbase Prime so that users can place limit orders.

**Acceptance Criteria:**
- [x] Connector implements OrderHook for LIMIT orders
- [x] Maps Formance order fields to Coinbase Prime CreateOrder request
- [x] Handles order response and updates order status
- [x] Typecheck/lint passes

#### US-014: Implement LIMIT order for Kraken
**Description:** As a developer, I need to implement LIMIT order creation for Kraken so that users can place limit orders.

**Acceptance Criteria:**
- [x] Connector implements OrderHook for LIMIT orders
- [x] Uses /0/private/AddOrder endpoint
- [x] Maps Formance order fields to Kraken order format (ordertype=limit, pair, price, type, volume)
- [x] Handles order response and updates order status
- [x] Typecheck/lint passes

### Phase 3: Order Cancellation

#### US-015: Cancel order API endpoint
**Description:** As a user, I want to cancel an open order so that I can stop a pending trade.

**Acceptance Criteria:**
- [x] Endpoint: POST /payments/v3/orders/:orderID/cancel
- [x] Only allows cancellation of orders in PENDING or OPEN status
- [x] Returns error if order is already FILLED, CANCELLED, or EXPIRED
- [x] Updates order status to CANCELLED on success
- [x] Typecheck/lint passes

#### US-016: Implement order cancellation for Coinbase Prime
**Description:** As a developer, I need to implement order cancellation for Coinbase Prime.

**Acceptance Criteria:**
- [x] Connector implements CancelOrderHook
- [x] Calls appropriate Coinbase Prime cancel endpoint
- [x] Handles cancellation response
- [x] Typecheck/lint passes

#### US-017: Implement order cancellation for Kraken
**Description:** As a developer, I need to implement order cancellation for Kraken.

**Acceptance Criteria:**
- [x] Connector implements CancelOrderHook
- [x] Uses /0/private/CancelOrder endpoint
- [x] Handles cancellation response
- [x] Typecheck/lint passes

### Phase 4: Market Data - Order Book & Quotes

#### US-018: Get order book API endpoint
**Description:** As a user, I want to view the order book for a trading pair so that I can see current market depth.

**Acceptance Criteria:**
- [x] Endpoint: GET /payments/v3/connectors/:connectorID/orderbook?pair=BTC/EUR
- [x] Returns bids and asks with price and quantity
- [x] Supports depth parameter to limit results
- [x] Typecheck/lint passes

#### US-019: Implement order book for Coinbase Prime
**Description:** As a developer, I need to implement order book retrieval for Coinbase Prime.

**Acceptance Criteria:**
- [x] Connector implements GetOrderBook capability
- [x] Maps response to standardized order book format
- [x] Typecheck/lint passes

#### US-020: Implement order book for Kraken
**Description:** As a developer, I need to implement order book retrieval for Kraken.

**Acceptance Criteria:**
- [x] Connector implements GetOrderBook capability
- [x] Maps response to standardized order book format
- [x] Typecheck/lint passes

#### US-021: Request quote API endpoint
**Description:** As a user, I want to request a price quote so that I can see the estimated cost before placing an order.

**Acceptance Criteria:**
- [x] Endpoint: POST /payments/v3/connectors/:connectorID/quotes
- [x] Request includes: sourceAsset, targetAsset, quantity, direction
- [x] Returns estimated price, fees, and expiration time
- [x] Typecheck/lint passes

#### US-022: Implement quotes for connectors
**Description:** As a developer, I need to implement quote requests for each connector.

**Acceptance Criteria:**
- [x] Connectors implement GetQuote capability where supported
- [x] Falls back to order book price estimation if quote endpoint unavailable
- [x] Typecheck/lint passes

### Phase 5: Time in Force & Tradable Assets

#### US-023: Support time in force on orders
**Description:** As a user, I want to specify time in force on my orders so that I can control order expiration behavior.

**Acceptance Criteria:**
- [x] Order creation accepts optional timeInForce parameter
- [x] Supports: GOOD_UNTIL_CANCELLED (GTC), GOOD_UNTIL_DATE_TIME (GTD), IMMEDIATE_OR_CANCEL (IOC), FILL_OR_KILL (FOK)
- [x] GTD requires additional expiresAt datetime parameter
- [x] Defaults to GTC if not specified
- [x] Connector maps to exchange-specific time in force values
- [x] Typecheck/lint passes

#### US-024: Get tradable assets API endpoint
**Description:** As a user, I want to see which trading pairs are available on a connector so that I know what I can trade.

**Acceptance Criteria:**
- [x] Endpoint: GET /payments/v3/connectors/:connectorID/assets
- [x] Returns list of available trading pairs with: pair name, base asset, quote asset, minimum order size, price precision
- [x] Typecheck/lint passes

#### US-025: Implement tradable assets for connectors
**Description:** As a developer, I need to implement tradable asset listing for each connector.

**Acceptance Criteria:**
- [x] Coinbase Prime connector returns available trading pairs
- [x] Kraken connector returns available trading pairs
- [x] Typecheck/lint passes

### Phase 6: MARKET Orders

#### US-026: Create MARKET order API endpoint
**Description:** As a user, I want to create a MARKET order so that I can buy or sell crypto immediately at the best available price.

**Acceptance Criteria:**
- [x] Endpoint: POST /payments/v3/orders with type=MARKET
- [x] Request includes: connectorId, direction, sourceAsset, targetAsset, quantity
- [x] No limitPrice required for MARKET orders
- [x] Order executes immediately or fails
- [x] Typecheck/lint passes

#### US-027: Implement MARKET order for Coinbase Prime
**Description:** As a developer, I need to implement MARKET order creation for Coinbase Prime.

**Acceptance Criteria:**
- [x] Connector implements OrderHook for MARKET orders
- [x] Maps to Coinbase Prime market order request
- [x] Typecheck/lint passes

#### US-028: Implement MARKET order for Kraken
**Description:** As a developer, I need to implement MARKET order creation for Kraken.

**Acceptance Criteria:**
- [x] Connector implements OrderHook for MARKET orders
- [x] Uses ordertype=market in Kraken request
- [x] Typecheck/lint passes

### Phase 7: Stablecoin Conversions

#### US-029: Create conversion API endpoint
**Description:** As a user, I want to convert between USD and stablecoins (USDC, PYUSD) so that I can move between fiat and crypto easily.

**Acceptance Criteria:**
- [x] Endpoint: POST /payments/v3/conversions
- [x] Request includes: connectorId, sourceAsset, targetAsset, amount, walletId
- [x] Supports USD↔USDC and USD↔PYUSD conversions
- [x] Returns conversion with PENDING status
- [x] Typecheck/lint passes

#### US-030: Implement conversions for Coinbase Prime
**Description:** As a developer, I need to implement stablecoin conversions for Coinbase Prime.

**Acceptance Criteria:**
- [x] Uses CreateConversion endpoint (not CreateOrder)
- [x] Requires walletId in request
- [x] Tracks conversion status via GetTransactions endpoint
- [x] Typecheck/lint passes

#### US-031: List conversions API endpoint
**Description:** As a user, I want to list my conversions so that I can track conversion history.

**Acceptance Criteria:**
- [x] Endpoint: GET /payments/v3/conversions
- [x] Supports filtering by connectorId, status, sourceAsset, targetAsset
- [x] Returns cursored list
- [x] Typecheck/lint passes

### Phase 8: Additional Market Data

#### US-032: Get ticker info API endpoint
**Description:** As a user, I want to see current ticker information so that I can monitor price movements.

**Acceptance Criteria:**
- [x] Endpoint: GET /payments/v3/connectors/:connectorID/ticker?pair=BTC/EUR
- [x] Returns: last price, bid, ask, volume, 24h high/low, price change
- [x] Typecheck/lint passes

#### US-033: Get OHLC data API endpoint
**Description:** As a user, I want to retrieve OHLC (candlestick) data so that I can analyze price history.

**Acceptance Criteria:**
- [x] Endpoint: GET /payments/v3/connectors/:connectorID/ohlc?pair=BTC/EUR&interval=1h
- [x] Supports intervals: 1m, 5m, 15m, 1h, 4h, 1d
- [x] Returns: timestamp, open, high, low, close, volume
- [x] Supports time range filtering
- [x] Typecheck/lint passes

### Phase 9: Additional Connectors

#### US-034: Implement Bitstamp connector
**Description:** As an operator, I want to configure a Bitstamp connector so that I can connect my Bitstamp account to Formance.

**Acceptance Criteria:**
- [x] Connector accepts Bitstamp API credentials
- [x] Implements all required capabilities (accounts, balances, orders)
- [x] Supports LIMIT and MARKET orders
- [x] Typecheck/lint passes

#### US-035: Implement Binance connector
**Description:** As an operator, I want to configure a Binance connector so that I can connect my Binance account to Formance.

**Acceptance Criteria:**
- [x] Connector accepts Binance API credentials
- [x] Uses official Binance Go SDK (github.com/binance/binance-connector-go)
- [x] Implements all required capabilities
- [x] Includes appropriate warnings about EU regulatory status
- [x] Typecheck/lint passes

### Phase 10: WebSocket Support & Optimizations

#### US-036: Add WebSocket support for real-time order updates
**Description:** As a user, I want real-time order status updates so that I don't have to poll for changes.

**Acceptance Criteria:**
- [x] Connectors support WebSocket connections where available
- [x] Order status changes are pushed in real-time
- [x] Falls back to REST polling if WebSocket unavailable
- [x] Handles reconnection gracefully
- [x] Typecheck/lint passes

#### US-037: Implement order validation
**Description:** As a system, I need to validate orders before submission so that obviously invalid orders are rejected early.

**Acceptance Criteria:**
- [x] Validates that trading pair is supported by connector
- [x] Validates minimum order size requirements
- [x] Optionally warns if limit price is >X% away from current market price
- [x] Returns clear error messages for validation failures
- [x] Typecheck/lint passes

### Phase 11: Advanced Order Types (Future)

#### US-038: Create STOP_LIMIT order
**Description:** As a user, I want to create a STOP_LIMIT order so that my order triggers when a price threshold is reached.

**Acceptance Criteria:**
- [ ] Supports type=STOP_LIMIT with stopPrice and limitPrice
- [ ] Order becomes LIMIT order when stopPrice is reached
- [ ] Implemented for connectors that support it
- [ ] Typecheck/lint passes

## Functional Requirements

### Order Primitive
- FR-1: The system must store orders with fields: id, reference, connectorId, direction, sourceAsset, targetAsset, status, type, baseQuantityOrdered, baseQuantityFilled, timeInForce, fee, createdAt, updatedAt
- FR-2: Order status must follow lifecycle: PENDING → OPEN → PARTIALLY_FILLED → FILLED (or CANCELLED/FAILED/EXPIRED from any state)
- FR-3: Order types must include MARKET and LIMIT (STOP_LIMIT in future)
- FR-4: All quantities must use integer representation (UMN) to avoid floating-point precision issues
- FR-5: The system must support specifying quantity on either source or target side via quantitySide parameter

### Conversion Primitive
- FR-6: The system must store conversions separately from orders for stablecoin exchanges
- FR-7: Conversions must support USD↔USDC and USD↔PYUSD pairs
- FR-8: Conversions must require a walletId for execution

### Connector Framework
- FR-9: Each connector must implement OrderHook to handle order creation
- FR-10: Each connector must implement CancelOrderHook to handle order cancellation
- FR-11: Connectors must implement CAPABILITY_FETCH_ORDERS for periodic order sync
- FR-12: Connectors must implement CAPABILITY_FETCH_ACCOUNTS for account/wallet sync
- FR-13: Connectors must implement CAPABILITY_FETCH_BALANCES for balance sync
- FR-14: Connectors must map exchange-specific statuses to Formance standard statuses

### API Endpoints
- FR-15: GET /payments/v3/orders/:orderID must return a single order by ID
- FR-16: GET /payments/v3/orders must return a filterable, paginated list of orders
- FR-17: POST /payments/v3/orders must create a new order
- FR-18: POST /payments/v3/orders/:orderID/cancel must cancel an open order
- FR-19: POST /payments/v3/conversions must create a new stablecoin conversion
- FR-20: GET /payments/v3/conversions must return a filterable list of conversions
- FR-21: GET /payments/v3/connectors/:connectorID/orderbook must return order book data
- FR-22: POST /payments/v3/connectors/:connectorID/quotes must return price quotes
- FR-23: GET /payments/v3/connectors/:connectorID/assets must return tradable pairs
- FR-24: GET /payments/v3/connectors/:connectorID/ticker must return ticker info
- FR-25: GET /payments/v3/connectors/:connectorID/ohlc must return OHLC data

### Connector-Specific
- FR-26: Coinbase Prime connector must use the official Go SDK (github.com/coinbase-samples/prime-sdk-go)
- FR-27: Coinbase Prime connector must be scoped to a single portfolio
- FR-28: Kraken connector must use the official Go SDK (github.com/krakenfx/api-go/v2) which handles API-Sign authentication internally
- FR-29: Binance connector must use the official Go SDK (github.com/binance/binance-connector-go)
- FR-30: Bitstamp connector uses direct REST API integration (no official SDK available)
- FR-31: All connectors must respect exchange rate limits

## Non-Goals (Out of Scope)

- **Futures trading** - Only spot trading is supported
- **Batch orders** - Orders must be created individually
- **Deposits/Withdrawals** - Fiat and crypto transfers to/from exchanges
- **Advanced order types (TWAP, VWAP, RFQ)** - Only MARKET, LIMIT, and STOP_LIMIT
- **Automated trading / bots** - No algorithmic trading features
- **Price alerts / notifications** - No notification system for price movements
- **Tax reporting** - No capital gains calculations or tax document generation
- **Multi-portfolio management** - Each Coinbase connector is scoped to one portfolio
- **On-chain transactions** - Only exchange-internal operations
- **Margin trading / leverage** - Only spot trading with available balance

## Technical Considerations

### Architecture
- Order primitive extends the existing Payment Service data model
- Connectors follow the existing PSP connector pattern
- OrderHook mechanism similar to existing payment hooks
- REST-first approach with optional WebSocket enhancement later

### Authentication & Security
- Connector credentials stored encrypted at rest
- Coinbase Prime: SDK handles authentication (accessKey, passphrase, signingKey)
- Kraken: SDK handles API-Sign authentication (HMAC-SHA512 with nonce) internally
- Binance: SDK handles HMAC-SHA256 authentication internally
- Bitstamp: Custom implementation of HMAC-SHA256 authentication
- API keys should have minimum required permissions

### Data Representation
- Use integer quantities (UMN) with asset precision metadata (e.g., BTC/8 = 8 decimal places)
- Store prices with sufficient precision for crypto assets
- Timestamps in UTC

### Rate Limiting
- Implement per-connector rate limiting
- SDKs handle basic rate limiting internally where supported
- Additional application-level rate limiting may be needed for high-volume usage
- Implement exponential backoff on rate limit errors

### Error Handling
- Map exchange-specific errors to standard error codes
- Provide clear error messages for order rejections
- Handle network timeouts and retries gracefully

### Dependencies
- Coinbase Prime Go SDK: github.com/coinbase-samples/prime-sdk-go
- Kraken Go SDK: github.com/krakenfx/api-go/v2
- Binance Go SDK: github.com/binance/binance-connector-go
- Bitstamp: Direct REST API integration (no official SDK available)

## Success Metrics

- Users can create and manage orders across multiple exchanges from a single API
- Order sync latency < 30 seconds for REST polling
- 99.9% success rate for order submission (excluding user errors)
- All exchange-specific order statuses correctly mapped to Formance statuses
- Zero precision loss in quantity/price conversions

## Open Questions

1. **UAT Environment:** How do we obtain Kraken UAT access for testing? Need to contact Kraken with company details and use case.
2. **Binance EU Compliance:** Should Binance connector include explicit compliance warnings given EU regulatory concerns?
3. **Order Validation Threshold:** What percentage deviation from market price should trigger a validation warning?
4. **Partial Fill Handling:** How should the system handle orders that remain partially filled for extended periods?
5. **Conversion Tracking:** Should conversions appear in the main order list or remain completely separate?
6. **WebSocket Priority:** Which connectors should get WebSocket support first?
