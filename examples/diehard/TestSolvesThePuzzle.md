# TestSolvesThePuzzle

## invariant "big jug is never at 4" violated

Explored 13 states.

```mermaid
flowchart TD
    s0["Big:0 Small:0"]
    s1["Big:5 Small:0"]
    s2["Big:2 Small:3"]
    s3["Big:2 Small:0"]
    s4["Big:0 Small:2"]
    s5["Big:5 Small:2"]
    s6["Big:4 Small:3"]
    s0 -->|"fill big"| s1
    s1 -->|"pour big into small"| s2
    s2 -->|"empty small"| s3
    s3 -->|"pour big into small"| s4
    s4 -->|"fill big"| s5
    s5 -->|"pour big into small"| s6
    style s6 stroke:#d33,stroke-width:2px
```

| # | action | state |
|--:|---|---|
| 0 | (init) | `Big:0 Small:0` |
| 1 | fill big | `Big:5 Small:0` |
| 2 | pour big into small | `Big:2 Small:3` |
| 3 | empty small | `Big:2 Small:0` |
| 4 | pour big into small | `Big:0 Small:2` |
| 5 | fill big | `Big:5 Small:2` |
| 6 | pour big into small | `Big:4 Small:3` |
