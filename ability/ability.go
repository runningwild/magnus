package ability

import (
	"github.com/runningwild/cgf"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/game"
	"github.com/runningwild/magnus/stats"
	"sync"
)

var abilityId int
var abilityIdMutex sync.Mutex

func nextAbilityId() int {
	abilityIdMutex.Lock()
	defer abilityIdMutex.Unlock()
	abilityId++
	return abilityId
}

type nonResponder struct{}

func (nonResponder) Respond(gid game.Gid, group gin.EventGroup) bool { return false }

type neverActive struct {
	nonResponder
}

func (neverActive) Deactivate(gid game.Gid) []cgf.Event { return nil }

type nonThinker struct{}

func (nonThinker) Think(game.Gid, *game.Game, linear.Vec2) ([]cgf.Event, bool) { return nil, false }

type nonRendering struct{}

func (nonRendering) Draw(gid game.Gid, game *game.Game, side int) {}

type BasicPhases struct {
	The_phase game.Phase
}

func (bp *BasicPhases) Kill(g *game.Game) {
	bp.The_phase = game.PhaseComplete
}

func (bp *BasicPhases) Terminated() bool {
	return bp.The_phase == game.PhaseComplete
}

func (bp *BasicPhases) Phase() game.Phase {
	return bp.The_phase
}

type NullCondition struct{}

func (NullCondition) ModifyBase(base stats.Base) stats.Base {
	return base
}
func (NullCondition) ModifyDamage(damage stats.Damage) stats.Damage {
	return damage
}
func (NullCondition) CauseDamage() stats.Damage {
	return stats.Damage{}
}
