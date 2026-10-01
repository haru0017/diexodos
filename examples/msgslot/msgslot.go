// Package msgslot models two runners racing for one slot held by a third
// machine, with every interaction going through mailboxes. Asking whether the
// slot is free and taking it are separate messages, so both runners can be
// told the slot is free. The try-take variant decides inside one handler,
// which is atomic, and mutual exclusion holds.
package msgslot

import "github.com/haru0017/diexodos/actor"

type Slot struct{ Holder int }
type Runner struct {
	ID      int
	Holding bool
}

type IsFree struct{ From string }
type IsFreeReply struct{ Free bool }
type Take struct{ ID int }

type TryTake struct {
	From string
	ID   int
}
type Granted struct{}

func System(atomicTake bool) *actor.System {
	s := actor.NewSystem()
	actor.Spawn(s, "slot", Slot{})
	actor.Spawn(s, "r1", Runner{ID: 1})
	actor.Spawn(s, "r2", Runner{ID: 2})

	if atomicTake {
		actor.On(s, "slot", func(m Slot, msg TryTake, ctx *actor.Ctx) Slot {
			if m.Holder == 0 {
				m.Holder = msg.ID
				ctx.Send(msg.From, Granted{})
			}
			return m
		})
		for id, r := range []string{"r1", "r2"} {
			actor.On(s, r, func(m Runner, _ Granted, _ *actor.Ctx) Runner {
				m.Holding = true
				return m
			})
			actor.Post(s, "slot", TryTake{From: r, ID: id + 1})
		}
	} else {
		actor.On(s, "slot", func(m Slot, msg IsFree, ctx *actor.Ctx) Slot {
			ctx.Send(msg.From, IsFreeReply{Free: m.Holder == 0})
			return m
		})
		actor.On(s, "slot", func(m Slot, msg Take, _ *actor.Ctx) Slot {
			m.Holder = msg.ID
			return m
		})
		for _, r := range []string{"r1", "r2"} {
			actor.On(s, r, func(m Runner, msg IsFreeReply, ctx *actor.Ctx) Runner {
				if msg.Free {
					ctx.Send("slot", Take{ID: m.ID})
					m.Holding = true
				}
				return m
			})
			actor.Post(s, "slot", IsFree{From: r})
		}
	}

	s.Inv("mutual exclusion", func(w actor.Snapshot) bool {
		return !(actor.Machine[Runner](w, "r1").Holding && actor.Machine[Runner](w, "r2").Holding)
	})
	return s
}
