# V3Schedule

A recurring job a connector runs to fetch data from its provider


## Fields

| Field                                                    | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ID`                                                     | *string*                                                 | :heavy_check_mark:                                       | Unique identifier of the schedule                        |
| `ConnectorID`                                            | *string*                                                 | :heavy_check_mark:                                       | Identifier of the connector this schedule belongs to     |
| `CreatedAt`                                              | [time.Time](https://pkg.go.dev/time#Time)                | :heavy_check_mark:                                       | When the schedule was created                            |
| `PausedAt`                                               | [*time.Time](https://pkg.go.dev/time#Time)               | :heavy_minus_sign:                                       | When the schedule was paused, absent while it is running |
| `PausedReason`                                           | **string*                                                | :heavy_minus_sign:                                       | Why the schedule was paused                              |