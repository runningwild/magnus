package ability

import (
	"encoding/gob"
	"github.com/runningwild/cgf"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/game"
	"math"
	"math/rand"
)

func makeMine(params map[string]int) game.Ability {
	var b mine
	b.id = NextAbilityId()
	b.health = float64(params["health"])
	b.damage = float64(params["damage"])
	b.trigger = float64(params["trigger"])
	b.mass = float64(params["mass"])
	return &b
}

func init() {
	game.RegisterAbility("mine", makeMine)
}

type mine struct {
	NonResponder
	NonThinker
	NonRendering

	id      int
	health  float64
	damage  float64
	trigger float64
	mass    float64
}

func (p *mine) Activate(gid game.Gid, keyPress bool) ([]cgf.Event, bool) {
	if !keyPress {
		return nil, false
	}
	ret := []cgf.Event{
		addMineEvent{
			PlayerGid: gid,
			Id:        p.id,
			Health:    p.health,
			Damage:    p.damage,
			Trigger:   p.trigger,
			Mass:      p.mass,
		},
	}
	return ret, false
}

func (p *mine) Deactivate(gid game.Gid) []cgf.Event {
	return nil
}

type addMineEvent struct {
	PlayerGid game.Gid
	Id        int
	Health    float64
	Damage    float64
	Trigger   float64
	Mass      float64
}

func init() {
	gob.Register(addMineEvent{})
}

func (e addMineEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player, ok := g.Ents[e.PlayerGid].(*game.Player)
	if !ok {
		return
	}
	var angle float64
	if player.Velocity.Mag() < 10 {
		angle = player.Velocity.Angle()
	} else {
		angle = player.Angle
	}
	pos := player.Position.Add((linear.Vec2{50, 0}).Rotate(angle + math.Pi))
	rng := rand.New(g.Rng)
	pos = pos.Add((linear.Vec2{rng.NormFloat64() * 15, 0}).Rotate(rng.Float64() * math.Pi * 2))
	g.MakeMine(pos, player.Velocity.Scale(0.5), e.Health, e.Mass, e.Damage, e.Trigger)
}
