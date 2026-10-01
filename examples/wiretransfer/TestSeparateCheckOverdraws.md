# TestSeparateCheckOverdraws

## invariant "the balance never goes negative" violated

Explored 9 states.

```mermaid
flowchart TD
    s0["Balance:10 Stage:[ready ready]"]
    s1["Balance:10 Stage:[checked ready]"]
    s2["Balance:10 Stage:[checked checked]"]
    s3["Balance:4 Stage:[done checked]"]
    s4["Balance:-2 Stage:[done done]"]
    s0 -->|"transfer 0 checks the balance"| s1
    s1 -->|"transfer 1 checks the balance"| s2
    s2 -->|"transfer 0 withdraws"| s3
    s3 -->|"transfer 1 withdraws"| s4
    style s4 stroke:#d33,stroke-width:2px
```

| # | action | state |
|--:|---|---|
| 0 | (init) | `Balance:10 Stage:[ready ready]` |
| 1 | transfer 0 checks the balance | `Balance:10 Stage:[checked ready]` |
| 2 | transfer 1 checks the balance | `Balance:10 Stage:[checked checked]` |
| 3 | transfer 0 withdraws | `Balance:4 Stage:[done checked]` |
| 4 | transfer 1 withdraws | `Balance:-2 Stage:[done done]` |
