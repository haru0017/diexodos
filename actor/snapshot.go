package actor

import (
	"fmt"
	"reflect"
	"strings"
)

// Snapshot is the state of the whole system at one point: every machine and
// every mailbox.
type Snapshot struct {
	names    []string
	machines map[string]any
	queues   map[string][]any
}

// Machine returns the state of a machine in the snapshot.
func Machine[M any](s Snapshot, name string) M {
	m, ok := s.machines[name]
	if !ok {
		panic(fmt.Sprintf("actor: machine %q not in snapshot", name))
	}
	return m.(M)
}

// QueueLen is the number of messages pending for a machine.
func (s Snapshot) QueueLen(name string) int { return len(s.queues[name]) }

// Quiet reports whether no message is pending anywhere.
func (s Snapshot) Quiet() bool {
	for _, q := range s.queues {
		if len(q) > 0 {
			return false
		}
	}
	return true
}

func (s Snapshot) clone() Snapshot {
	machines := make(map[string]any, len(s.machines))
	for k, v := range s.machines {
		machines[k] = v
	}
	queues := make(map[string][]any, len(s.queues))
	for k, v := range s.queues {
		queues[k] = v
	}
	return Snapshot{names: s.names, machines: machines, queues: queues}
}

// without returns the queue with the message at idx removed, always as a
// fresh slice so that branches never alias each other.
func without(q []any, idx int) []any {
	out := make([]any, 0, len(q)-1)
	out = append(out, q[:idx]...)
	return append(out, q[idx+1:]...)
}

func appended(q []any, msg any) []any {
	out := make([]any, 0, len(q)+1)
	out = append(out, q...)
	return append(out, msg)
}

// key builds a canonical string identity. Machine states and messages are
// plain values, so their printed form is deterministic.
func (s Snapshot) key() string {
	var b strings.Builder
	for _, name := range s.names {
		fmt.Fprintf(&b, "%s=%#v;", name, s.machines[name])
		for _, msg := range s.queues[name] {
			fmt.Fprintf(&b, "%#v|", msg)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// mustBePlain rejects types whose values could alias mutable memory or make
// the canonical key nondeterministic.
func mustBePlain(t reflect.Type) {
	if t == nil {
		panic("actor: nil is not a valid state or message")
	}
	if err := plainErr(t); err != nil {
		panic(fmt.Sprintf("actor: %s", err))
	}
}

func plainErr(t reflect.Type) error {
	switch t.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return nil
	case reflect.Array:
		return plainErr(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if err := plainErr(t.Field(i).Type); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("type %s contains %s; states and messages must be plain values", t, t.Kind())
	}
}
