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
// repeat forever under weak fairness.
type Lasso[S any] struct {
	Property string
	Prefix   []Step[S]
	Cycle    []Step[S]
}

// CheckLiveness looks for a cycle violating one of the properties that is
// consistent with weak fairness of the fair rules. It adds stuttering
// self-loops to terminal states so that every behavior is infinite.
//
// A strongly connected component admits a fair violating cycle exactly when a
// closed walk covering the component does: every fair rule must either be
// taken on some internal edge or be disabled somewhere in the component.
func CheckLiveness[S any](g *Graph[S], props []Liveness[S]) *Lasso[S] {
	for id := range g.States {
		if len(g.Edges[id]) == 0 {
			g.Edges[id] = append(g.Edges[id], Edge{To: id, Rule: -1})
		}
	}
	var fair []int
	for i, r := range g.Rules {
		if r.Fair {
			fair = append(fair, i)
		}
	}

	for _, p := range props {
		holds := make([]bool, len(g.States))
		for id, s := range g.States {
			holds[id] = p.Holds(s)
		}
		member := func(int) bool { return true }
		if p.Mode == ModeAlwaysEventually {
			// A violating cycle avoids the predicate entirely.
			member = func(id int) bool { return !holds[id] }
		}
		for _, comp := range sccs(g, member) {
			in := make(map[int]bool, len(comp))
			for _, id := range comp {
				in[id] = true
			}
			if !hasInternalEdge(g, comp, in) {
				continue
			}
			if p.Mode == ModeEventuallyAlways && allHold(comp, holds) {
				continue
			}
			if !fairComponent(g, comp, in, fair) {
				continue
			}
			return buildLasso(g, p, comp, in, fair, holds)
		}
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

// fairComponent reports whether a closed walk covering the component
// satisfies weak fairness for every fair rule.
func fairComponent[S any](g *Graph[S], comp []int, in map[int]bool, fair []int) bool {
	for _, f := range fair {
		ok := false
		for _, u := range comp {
			for _, e := range g.Edges[u] {
				if in[e.To] && e.Rule == f {
					ok = true
					break
				}
			}
			if ok {
				break
			}
		}
		if ok {
			continue
		}
		for _, u := range comp {
			if len(g.Rules[f].Next(g.States[u])) == 0 {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
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
func buildLasso[S any](g *Graph[S], p Liveness[S], comp []int, in map[int]bool, fair []int, holds []bool) *Lasso[S] {
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
	for _, f := range fair {
		var viaEdge *edgeAt
		for _, u := range comp {
			for _, e := range internal(u) {
				if e.Rule == f {
					viaEdge = &edgeAt{u, e}
					break
				}
			}
			if viaEdge != nil {
				break
			}
		}
		if viaEdge != nil {
			edgeTargets = append(edgeTargets, *viaEdge)
			continue
		}
		for _, u := range comp {
			if len(g.Rules[f].Next(g.States[u])) == 0 {
				stateTargets = append(stateTargets, u)
				break
			}
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
