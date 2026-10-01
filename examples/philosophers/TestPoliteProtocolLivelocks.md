# TestPoliteProtocolLivelocks

## liveness "everyone has eaten" violated

Explored 27 states.

```mermaid
flowchart TD
    s0["Taken:[false false false] Phase:[thinking thinking thinking]"]
    s1["Taken:[true false false] Phase:[one fork thinking thinking]"]
    s2["Taken:[false false false] Phase:[done thinking thinking]"]
    s3["Taken:[false false true] Phase:[done thinking one fork]"]
    s4["Taken:[false true true] Phase:[done one fork one fork]"]
    s0 -->|"P0 takes a first fork"| s1
    s1 -->|"P0 takes the second fork and eats"| s2
    s2 -->|"P2 takes a first fork"| s3
    s3 ==>|"P1 takes a first fork"| s4
    s4 ==>|"P1 puts the fork back"| s3
```

| # | action | state |
|--:|---|---|
| 0 | (init) | `Taken:[false false false] Phase:[thinking thinking thinking]` |
| 1 | P0 takes a first fork | `Taken:[true false false] Phase:[one fork thinking thinking]` |
| 2 | P0 takes the second fork and eats | `Taken:[false false false] Phase:[done thinking thinking]` |
| 3 | P2 takes a first fork | `Taken:[false false true] Phase:[done thinking one fork]` |
| 4 ↺ | P1 takes a first fork | `Taken:[false true true] Phase:[done one fork one fork]` |
| 5 ↺ | P1 puts the fork back | `Taken:[false false true] Phase:[done thinking one fork]` |

The ↺ steps repeat forever.
