package engine

// LivenessMode selects the temporal property shape.
type LivenessMode int

const (
	// ModeEventuallyAlways checks that on every fair behavior the predicate
	// eventually holds forever.
	ModeEventuallyAlways LivenessMode = iota
	// ModeAlwaysEventually checks that on every fair behavior the predicate
	// holds infinitely often.
	ModeAlwaysEventually
)

type Liveness[S any] struct {
	Name  string
	Mode  LivenessMode
	Holds func(S) bool
}

// Lasso is a liveness counterexample: a finite prefix into a cycle that can
// repeat forever under the declared fairness.
type Lasso[S any] struct {
	Property string
	Prefix   []Step[S]
	Cycle    []Step[S]
}

// CheckLiveness looks for a cycle violating one of the properties that is
// consistent with the fairness of the rules. It adds stuttering self-loops to
// terminal states so that every behavior is infinite.
//
// A strongly connected component admits a fair violating cycle exactly when a
// closed walk covering it does. Weak fairness of a rule needs the rule taken
// on some internal edge or disabled somewhere in the component; failing that,
// every subcomponent fails too, so the component is rejected. Strong fairness
// needs the rule taken or enabled nowhere; failing that, a subcomponent
// avoiding the enabling states may still qualify, so those states are pruned
// and the search recurses.
func CheckLiveness[S any](g *Graph[S], props []Liveness[S]) *Lasso[S] {
	for id := range g.States {
		if len(g.Edges[id]) == 0 {
			g.Edges[id] = append(g.Edges[id], Edge{To: id, Rule: -1})
		}
	}
	var weak, strong []int
	for i, r := range g.Rules {
		if r.Fair {
			weak = append(weak, i)
		}
		if r.StrongFair {
			strong = append(strong, i)
		}
	}

	for _, p := range props {
		holds := make([]bool, len(g.States))
		for id, s := range g.States {
			holds[id] = p.Holds(s)
		}
		nodes := make(map[int]bool, len(g.States))
		for id := range g.States {
			if p.Mode == ModeAlwaysEventually && holds[id] {
				// A violating cycle avoids the predicate entirely.
				continue
			}
			nodes[id] = true
		}
		accept := func([]int) bool { return true }
		if p.Mode == ModeEventuallyAlways {
			accept = func(comp []int) bool { return !allHold(comp, holds) }
		}
		if comp := searchFair(g, nodes, weak, strong, accept); comp != nil {
			return buildLasso(g, p, comp, weak, strong, holds)
		}
	}
	return nil
}

// searchFair returns a component of the subgraph induced by nodes that has an
// internal edge, satisfies accept, and admits a fair covering walk.
func searchFair[S any](g *Graph[S], nodes map[int]bool, weak, strong []int, accept func([]int) bool) []int {
	member := func(id int) bool { return nodes[id] }
	for _, comp := range sccs(g, member) {
		in := make(map[int]bool, len(comp))
		for _, id := range comp {
			in[id] = true
		}
		if !hasInternalEdge(g, comp, in) {
			continue
		}
		if !accept(comp) {
			continue
		}
		if !weaklyFairComponent(g, comp, in, weak) {
			// Weak fairness failure is monotone under pruning: the rule stays
			// enabled everywhere and untaken in every subcomponent.
			continue
		}
		if bad := unsatisfiedStrong(g, comp, in, strong); bad >= 0 {
			pruned := make(map[int]bool, len(comp))
			for _, id := range comp {
				if len(g.Rules[bad].Next(g.States[id])) == 0 {
					pruned[id] = true
				}
			}
			if sub := searchFair(g, pruned, weak, strong, accept); sub != nil {
				return sub
			}
			continue
		}
		return comp
	}
	return nil
}

func hasInternalEdge[S any](g *Graph[S], comp []int, in map[int]bool) bool {
	for _, u := range comp {
		for _, e := range g.Edges[u] {
			if in[e.To] {
				return true
			}
		}
	}
	return false
}

func allHold(comp []int, holds []bool) bool {
	for _, id := range comp {
		if !holds[id] {
			return false
		}
	}
	return true
}

func takenInside[S any](g *Graph[S], comp []int, in map[int]bool, rule int) bool {
	for _, u := range comp {
		for _, e := range g.Edges[u] {
			if in[e.To] && e.Rule == rule {
				return true
			}
		}
	}
	return false
}

func weaklyFairComponent[S any](g *Graph[S], comp []int, in map[int]bool, weak []int) bool {
	for _, f := range weak {
		if takenInside(g, comp, in, f) {
			continue
		}
		disabled := false
		for _, u := range comp {
			if len(g.Rules[f].Next(g.States[u])) == 0 {
				disabled = true
				break
			}
		}
		if !disabled {
			return false
		}
	}
	return true
}

// unsatisfiedStrong returns a strongly fair rule that is enabled somewhere in
// the component yet taken on no internal edge, or -1.
func unsatisfiedStrong[S any](g *Graph[S], comp []int, in map[int]bool, strong []int) int {
	for _, f := range strong {
		if takenInside(g, comp, in, f) {
			continue
		}
		for _, u := range comp {
			if len(g.Rules[f].Next(g.States[u])) > 0 {
				return f
			}
		}
	}
	return -1
}

func ruleName[S any](g *Graph[S], idx int) string {
	if idx < 0 {
		return "(stutter)"
	}
	return g.Rules[idx].Name
}

// buildLasso constructs a concrete counterexample: a shortest prefix to the
// component and a closed walk through it that visits every witness needed by
// the property and by fairness.
func buildLasso[S any](g *Graph[S], p Liveness[S], comp []int, weak, strong []int, holds []bool) *Lasso[S] {
	in := make(map[int]bool, len(comp))
	for _, id := range comp {
		in[id] = true
	}
	type edgeAt struct {
		from int
		e    Edge
	}
	internal := func(id int) []Edge {
		var out []Edge
		for _, e := range g.Edges[id] {
			if in[e.To] {
				out = append(out, e)
			}
		}
		return out
	}
	edgeWitness := func(rule int) (edgeAt, bool) {
		for _, u := range comp {
			for _, e := range internal(u) {
				if e.Rule == rule {
					return edgeAt{u, e}, true
				}
			}
		}
		return edgeAt{}, false
	}

	var stateTargets []int
	var edgeTargets []edgeAt
	if p.Mode == ModeEventuallyAlways {
		for _, id := range comp {
			if !holds[id] {
				stateTargets = append(stateTargets, id)
				break
			}
		}
	}
	for _, f := range weak {
		if w, ok := edgeWitness(f); ok {
			edgeTargets = append(edgeTargets, w)
			continue
		}
		for _, u := range comp {
			if len(g.Rules[f].Next(g.States[u])) == 0 {
				stateTargets = append(stateTargets, u)
				break
			}
		}
	}
	for _, f := range strong {
		// Either the rule fires inside the cycle or it is enabled nowhere in
		// the component, which needs no witness.
		if w, ok := edgeWitness(f); ok {
			edgeTargets = append(edgeTargets, w)
		}
	}

	c0 := comp[0]
	cur := c0
	var cycle []Step[S]
	walkTo := func(to int) {
		if cur == to {
			return
		}
		prev := make(map[int]edgeAt)
		seen := map[int]bool{cur: true}
		queue := []int{cur}
		for len(queue) > 0 {
			u := queue[0]
			queue = queue[1:]
			for _, e := range internal(u) {
				if seen[e.To] {
					continue
				}
				seen[e.To] = true
				prev[e.To] = edgeAt{u, e}
				if e.To == to {
					queue = nil
					break
				}
				queue = append(queue, e.To)
			}
		}
		if !seen[to] {
			panic("engine: lasso target unreachable inside its component")
		}
		var rev []edgeAt
		for at := to; at != cur; {
			pe := prev[at]
			rev = append(rev, pe)
			at = pe.from
		}
		for i := len(rev) - 1; i >= 0; i-- {
			e := rev[i]
			cycle = append(cycle, Step[S]{Rule: ruleName(g, e.e.Rule), State: g.States[e.e.To]})
		}
		cur = to
	}

	for _, t := range stateTargets {
		walkTo(t)
	}
	for _, et := range edgeTargets {
		walkTo(et.from)
		cycle = append(cycle, Step[S]{Rule: ruleName(g, et.e.Rule), State: g.States[et.e.To]})
		cur = et.e.To
	}
	walkTo(c0)
	if len(cycle) == 0 {
		e := internal(c0)[0]
		cycle = append(cycle, Step[S]{Rule: ruleName(g, e.Rule), State: g.States[e.To]})
		cur = e.To
		walkTo(c0)
	}

	return &Lasso[S]{Property: p.Name, Prefix: g.path(c0), Cycle: cycle}
}

// sccs computes strongly connected components of the subgraph induced by
// member, using an iterative Tarjan traversal.
func sccs[S any](g *Graph[S], member func(int) bool) [][]int {
	n := len(g.States)
	index := make([]int, n)
	low := make([]int, n)
	onstack := make([]bool, n)
	for i := range index {
		index[i] = -1
	}
	var stack []int
	var result [][]int
	next := 1

	type frame struct {
		v, ei int
	}
	for root := 0; root < n; root++ {
		if !member(root) || index[root] != -1 {
			continue
		}
		index[root], low[root] = next, next
		next++
		stack = append(stack, root)
		onstack[root] = true
		call := []frame{{root, 0}}
		for len(call) > 0 {
			f := &call[len(call)-1]
			v := f.v
			descended := false
			for f.ei < len(g.Edges[v]) {
				e := g.Edges[v][f.ei]
				f.ei++
				w := e.To
				if !member(w) {
					continue
				}
				if index[w] == -1 {
					index[w], low[w] = next, next
					next++
					stack = append(stack, w)
					onstack[w] = true
					call = append(call, frame{w, 0})
					descended = true
					break
				}
				if onstack[w] && index[w] < low[v] {
					low[v] = index[w]
				}
			}
			if descended {
				continue
			}
			call = call[:len(call)-1]
			if len(call) > 0 {
				parent := call[len(call)-1].v
				if low[v] < low[parent] {
					low[parent] = low[v]
				}
			}
			if low[v] == index[v] {
				var comp []int
				for {
					w := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					onstack[w] = false
					comp = append(comp, w)
					if w == v {
						break
					}
				}
				result = append(result, comp)
			}
		}
	}
	return result
}
