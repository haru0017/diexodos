# TestAStoppedManagerBlocksThePrepared

## liveness "every resource manager decides" violated

Explored 31 states.

```mermaid
flowchart TD
    s0["TM:deciding RM:[working working]"]
    s1["TM:deciding RM:[prepared working]"]
    s2["TM:deciding RM:[prepared prepared]"]
    s3["TM:stopped RM:[prepared prepared]"]
    s0 -->|"RM0 prepares"| s1
    s1 -->|"RM1 prepares"| s2
    s2 -->|"the TM stops"| s3
    s3 ==>|"(stutter)"| s3
```

| # | action | state |
|--:|---|---|
| 0 | (init) | `TM:deciding RM:[working working]` |
| 1 | RM0 prepares | `TM:deciding RM:[prepared working]` |
| 2 | RM1 prepares | `TM:deciding RM:[prepared prepared]` |
| 3 | the TM stops | `TM:stopped RM:[prepared prepared]` |
| 4 ↺ | (stutter) | `TM:stopped RM:[prepared prepared]` |

The ↺ steps repeat forever.
