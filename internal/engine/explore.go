// Package engine implements exhaustive breadth-first exploration of a state
// space defined by transition rules. State identity is delegated to a key
// function mapping states to a comparable key, so equality is exact and
// hash-collision unsoundness is ruled out by construction.
package engine

// Rule produces the successor states reachable from a state in one step.
// A rule that is not enabled in a state returns no successors. Fair marks
// the rule as weakly fair and StrongFair as strongly fair for liveness
// checking.
type Rule[S any] struct {
	Name       string
	Fair       bool
	StrongFair bool
	Next       func(S) []S
}

// Invariant is a predicate that must hold in every reachable state.
type Invariant[S any] struct {
	Name  string
	Holds func(S) bool
}

// Step is one entry of a counterexample path. Rule is empty for an initial
// state.
type Step[S any] struct {
	Rule  string
	State S
}

// Violation is an invariant failure together with a shortest path from an
// initial state to the failing state.
type Violation[S any] struct {
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
	// DetectDeadlocks reports a state with no successors as a violation.
	DetectDeadlocks bool
	// BuildGraph records the full edge list for liveness checking.
	BuildGraph bool
}

type Result[S any] struct {
	States    int
	Violation *Violation[S]
	Truncated bool
}

// Graph is the explored state graph, produced when Options.BuildGraph is set
// and exploration completes without a violation.
type Graph[S any] struct {
	States []S
	Origin []Origin
	Edges  [][]Edge
	Rules  []Rule[S]
}

// Origin records how a state was first reached. Prev is -1 for an initial
// state.
type Origin struct {
	Prev int
	Rule int
}

// Edge is a labeled transition between state ids. Rule is -1 for the
// stuttering self-loop added to terminal states.
type Edge struct {
	To   int
	Rule int
}

// Explore walks every state reachable from init through rules, checking invs
// on each state. It stops at the first violation, which BFS makes a shortest
// counterexample.
func Explore[S any, K comparable](init []S, rules []Rule[S], invs []Invariant[S], key func(S) K, opts Options) (Result[S], *Graph[S]) {
	progressEvery := opts.ProgressEvery
	if progressEvery <= 0 {
		progressEvery = 1 << 15
	}

	g := &Graph[S]{Rules: rules}
	index := make(map[K]int)
	var queue []int
	var res Result[S]

	check := func(id int) bool {
		s := g.States[id]
		for _, inv := range invs {
			if !inv.Holds(s) {
				res.Violation = &Violation[S]{
					Invariant: inv.Name,
					Path:      g.path(id),
				}
				return true
			}
		}
		return false
	}

	admit := func(k K, s S, prev, rule int) int {
		id := len(g.States)
		index[k] = id
		g.States = append(g.States, s)
		g.Origin = append(g.Origin, Origin{Prev: prev, Rule: rule})
		if opts.BuildGraph {
			g.Edges = append(g.Edges, nil)
		}
		res.States++
		if opts.Progress != nil && res.States%progressEvery == 0 {
			opts.Progress(res.States)
		}
		return id
	}

	for _, s := range init {
		k := key(s)
		if _, ok := index[k]; ok {
			continue
		}
		id := admit(k, s, -1, -1)
		if check(id) {
			return res, nil
		}
		queue = append(queue, id)
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		successors := 0
		for ri, r := range rules {
			for _, next := range r.Next(g.States[cur]) {
				successors++
				k := key(next)
				id, seen := index[k]
				if !seen {
					if opts.MaxStates > 0 && res.States >= opts.MaxStates {
						res.Truncated = true
						return res, nil
					}
					id = admit(k, next, cur, ri)
				}
				if opts.BuildGraph {
					g.Edges[cur] = append(g.Edges[cur], Edge{To: id, Rule: ri})
				}
				if seen {
					continue
				}
				if check(id) {
					return res, nil
				}
				queue = append(queue, id)
			}
		}
		if successors == 0 && opts.DetectDeadlocks {
			res.Violation = &Violation[S]{Invariant: "deadlock", Path: g.path(cur)}
			return res, nil
		}
	}

	if !opts.BuildGraph {
		return res, nil
	}
	return res, g
}

func (g *Graph[S]) path(id int) []Step[S] {
	var rev []Step[S]
	for cur := id; cur >= 0; {
		o := g.Origin[cur]
		name := ""
		if o.Rule >= 0 {
			name = g.Rules[o.Rule].Name
		}
		rev = append(rev, Step[S]{Rule: name, State: g.States[cur]})
		cur = o.Prev
	}
	path := make([]Step[S], len(rev))
	for i, s := range rev {
		path[len(rev)-1-i] = s
	}
	return path
}
