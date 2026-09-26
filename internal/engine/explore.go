// Package engine implements exhaustive breadth-first exploration of a state
// space defined by transition rules. States must be comparable so that Go
// equality is the state identity, which rules out hash-collision unsoundness
// by construction.
package engine

// Rule produces the successor states reachable from a state in one step.
// A rule that is not enabled in a state returns no successors.
type Rule[S comparable] struct {
	Name string
	Next func(S) []S
}

// Invariant is a predicate that must hold in every reachable state.
type Invariant[S comparable] struct {
	Name  string
	Holds func(S) bool
}

// Step is one entry of a counterexample path. Rule is empty for an initial
// state.
type Step[S comparable] struct {
	Rule  string
	State S
}

// Violation is an invariant failure together with a shortest path from an
// initial state to the failing state.
type Violation[S comparable] struct {
	Invariant string
	Path      []Step[S]
}

type Options struct {
	// MaxStates aborts exploration after this many states. Zero means no
	// limit.
	MaxStates int
	// Progress, if set, is called every ProgressEvery discovered states.
	Progress      func(states int)
	ProgressEvery int
}

type Result[S comparable] struct {
	States    int
	Violation *Violation[S]
	Truncated bool
}

type origin[S comparable] struct {
	prev    S
	rule    string
	initial bool
}

// Explore walks every state reachable from init through rules, checking invs
// on each state. It stops at the first violation, which BFS makes a shortest
// counterexample.
func Explore[S comparable](init []S, rules []Rule[S], invs []Invariant[S], opts Options) Result[S] {
	progressEvery := opts.ProgressEvery
	if progressEvery <= 0 {
		progressEvery = 1 << 15
	}

	visited := make(map[S]origin[S])
	var queue []S
	var res Result[S]

	check := func(s S) bool {
		for _, inv := range invs {
			if !inv.Holds(s) {
				res.Violation = &Violation[S]{
					Invariant: inv.Name,
					Path:      buildPath(visited, s),
				}
				return true
			}
		}
		return false
	}

	for _, s := range init {
		if _, ok := visited[s]; ok {
			continue
		}
		visited[s] = origin[S]{initial: true}
		res.States++
		if check(s) {
			return res
		}
		queue = append(queue, s)
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, r := range rules {
			for _, next := range r.Next(cur) {
				if _, ok := visited[next]; ok {
					continue
				}
				if opts.MaxStates > 0 && res.States >= opts.MaxStates {
					res.Truncated = true
					return res
				}
				visited[next] = origin[S]{prev: cur, rule: r.Name}
				res.States++
				if opts.Progress != nil && res.States%progressEvery == 0 {
					opts.Progress(res.States)
				}
				if check(next) {
					return res
				}
				queue = append(queue, next)
			}
		}
	}
	return res
}

func buildPath[S comparable](visited map[S]origin[S], last S) []Step[S] {
	var rev []Step[S]
	cur := last
	for {
		o := visited[cur]
		rev = append(rev, Step[S]{Rule: o.rule, State: cur})
		if o.initial {
			break
		}
		cur = o.prev
	}
	path := make([]Step[S], len(rev))
	for i, s := range rev {
		path[len(rev)-1-i] = s
	}
	return path
}
