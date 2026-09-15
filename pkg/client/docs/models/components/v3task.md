# V3Task

An asynchronous unit of work, tracking an operation that completes in the background


## Fields

| Field                                                                      | Type                                                                       | Required                                                                   | Description                                                                |
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `ID`                                                                       | *string*                                                                   | :heavy_check_mark:                                                         | Unique identifier of the task                                              |
| `Status`                                                                   | [components.V3TaskStatusEnum](../../models/components/v3taskstatusenum.md) | :heavy_check_mark:                                                         | Where a task stands, from processing through to succeeded or failed        |
| `CreatedAt`                                                                | [time.Time](https://pkg.go.dev/time#Time)                                  | :heavy_check_mark:                                                         | When the task was created                                                  |
| `UpdatedAt`                                                                | [time.Time](https://pkg.go.dev/time#Time)                                  | :heavy_check_mark:                                                         | When the task was last updated                                             |
| `ConnectorID`                                                              | **string*                                                                  | :heavy_minus_sign:                                                         | Identifier of the connector the task runs against                          |
| `CreatedObjectID`                                                          | **string*                                                                  | :heavy_minus_sign:                                                         | Identifier of the object the task created, once it has succeeded           |
| `Error`                                                                    | **string*                                                                  | :heavy_minus_sign:                                                         | Why the task failed, absent when it succeeded                              |