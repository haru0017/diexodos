# TestNaiveProtocolDeadlocks

## liveness "everyone has eaten" violated

Explored 27 states.

```mermaid
flowchart TD
    s0["Taken:[false false false] Phase:[thinking thinking thinking]"]
    s1["Taken:[true false false] Phase:[one fork thinking thinking]"]
    s2["Taken:[true true false] Phase:[one fork one fork thinking]"]
    s3["Taken:[true true true] Phase:[one fork one fork one fork]"]
    s0 -->|"P0 takes a first fork"| s1
    s1 -->|"P1 takes a first fork"| s2
    s2 -->|"P2 takes a first fork"| s3
    s3 ==>|"(stutter)"| s3
```

| # | action | state |
|--:|---|---|
| 0 | (init) | `Taken:[false false false] Phase:[thinking thinking thinking]` |
| 1 | P0 takes a first fork | `Taken:[true false false] Phase:[one fork thinking thinking]` |
| 2 | P1 takes a first fork | `Taken:[true true false] Phase:[one fork one fork thinking]` |
| 3 | P2 takes a first fork | `Taken:[true true true] Phase:[one fork one fork one fork]` |
| 4 ↺ | (stutter) | `Taken:[true true true] Phase:[one fork one fork one fork]` |

The ↺ steps repeat forever.
