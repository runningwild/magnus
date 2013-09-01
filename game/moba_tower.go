package game

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/stats"
	"github.com/runningwild/magnus/texture"
)

type ControlPoint struct {
	BaseEnt

	// 0.0 - 1.0, measured controlledness of the point
	Control float64

	// The side that controls the point
	Controller int

	// Whether or not the point is currently controlled.  This is always true when
	// Control is 1.0, but may be true or false otherwise.
	Controlled bool

	// If an enemy comes into LOS of the CP when the AttackTimer is at zero then
	// an attack process will begin and the AttackTimer will be set.  It will
	// count down on every think until it reaches zero again.
	AttackTimer int
}

func (g *Game) MakeControlPoints() {
	data := g.Levels[GidInvadersStart].Room.Moba.SideData
	neutralData := data[len(data)-1]
	for _, towerPos := range neutralData.Towers {
		cp := ControlPoint{
			BaseEnt: BaseEnt{
				Side_:        -1,
				CurrentLevel: GidInvadersStart,
				Position:     towerPos,
				StatsInst: stats.Make(stats.Base{
					Health: 100000,
					Mass:   1000000,
					Rate:   1,
					Size:   50,
					Vision: 900,
				}),
			},
		}
		g.AddEnt(&cp)
	}
}

func (cp *ControlPoint) Side() int {
	if cp.Controlled {
		return cp.Controller
	}
	return cp.BaseEnt.Side()
}

func (cp *ControlPoint) Think(g *Game) {
	cp.BaseEnt.Think(g)

	// All of this is basic logic for capturing control points
	sides := make(map[int]int)
	var theSide int
	for _, ent := range g.temp.AllEnts {
		if _, ok := ent.(*Player); !ok {
			continue
		}
		if ent.Pos().Sub(cp.Position).Mag() > 2*cp.Stats().Size() {
			continue
		}
		sides[ent.Side()]++
		theSide = ent.Side()
	}
	if len(sides) == 1 {
		// amt := 0.003 * float64(sides[theSide])
		amt := 0.03 * float64(sides[theSide])
		if !cp.Controlled || theSide == cp.Controller {
			if cp.Control < 1.0 {
				cp.Control += amt
				if cp.Control >= 0.999 {
					cp.Control = 1.0
					cp.Controlled = true
					cp.Controller = theSide
				}
			}
		} else {
			if cp.Control > 0.0 {
				cp.Control -= amt
				if cp.Control <= 0.0001 {
					cp.Control = 0
					cp.Controlled = false
					cp.Controller = -1
				}
			}
		}
	}

	// Now check for targets
	if cp.AttackTimer > 0 {
		cp.AttackTimer--
	}
	if cp.Controlled && cp.AttackTimer == 0 {
		for _, ent := range g.temp.AllEnts {
			if _, ok := ent.(*Player); !ok || ent.Side() == cp.Side() {
				continue
			}
			x := int(ent.Pos().X+0.5) / LosGridSize
			y := int(ent.Pos().Y+0.5) / LosGridSize
			// NEXT: This is somehow not stopping it from hitting targets without LOS
			res := g.Moba.Sides[cp.Side()].losCache.Get(int(cp.Position.X), int(cp.Position.Y), cp.Stats().Vision())
			hit := false
			for _, v := range res {
				if v.X == x && v.Y == y {
					hit = true
					break
				}
			}
			if hit {
				cp.AttackTimer = 100
				g.Processes = append(g.Processes, &controlPointAttackProcess{
					Target:      ent.Id(),
					Side:        cp.Side(),
					Timer:       0,
					LockTime:    30,
					FireTime:    60,
					ProjPos:     cp.Position,
					ProjSpeed:   8.0,
					BlastRadius: 50,
				})
			}
		}
	}
}

func (cp *ControlPoint) Draw(g *Game, side int) {
	base.EnableShader("circle")
	base.SetUniformF("circle", "edge", 0.95)
	gl.Color4ub(50, 50, 100, 150)
	texture.Render(
		cp.Position.X-cp.Stats().Size()*2,
		cp.Position.Y-cp.Stats().Size()*2,
		2*cp.Stats().Size()*2,
		2*cp.Stats().Size()*2)

	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.0)
	base.SetUniformF("status_bar", "outer", 0.5)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(50, 50, 50, 200)
	texture.Render(
		cp.Position.X-cp.Stats().Size(),
		cp.Position.Y-cp.Stats().Size(),
		2*cp.Stats().Size(),
		2*cp.Stats().Size())

	base.SetUniformF("status_bar", "frac", float32(cp.Control))
	if cp.Controlled {
		if side == cp.Controller {
			gl.Color4ub(0, 255, 0, 255)
		} else {
			gl.Color4ub(255, 0, 0, 255)
		}
	} else {
		gl.Color4ub(100, 100, 100, 255)
	}
	texture.Render(
		cp.Position.X-cp.Stats().Size(),
		cp.Position.Y-cp.Stats().Size(),
		2*cp.Stats().Size(),
		2*cp.Stats().Size())
	base.EnableShader("")
}
func (cp *ControlPoint) Supply(mana Mana) Mana { return Mana{} }
func (cp *ControlPoint) Walls() [][]linear.Vec2 {
	return nil
}

type controlPointAttackProcess struct {
	Target   Gid
	Side     int
	Timer    int
	LockTime int
	LockPos  linear.Vec2

	FireTime  int
	ProjPos   linear.Vec2
	ProjSpeed float64

	BlastRadius float64
	Killed      bool
}

func (cpap *controlPointAttackProcess) Supply(supply Mana) Mana {
	return supply
}
func (cpap *controlPointAttackProcess) Think(g *Game) {
	cpap.Timer++
	if cpap.Timer == cpap.LockTime {
		target := g.Ents[cpap.Target]
		if target == nil {
			cpap.Kill(g)
			return
		}
		cpap.LockPos = target.Pos()
	}
	if cpap.Timer >= cpap.FireTime {
		dir := cpap.LockPos.Sub(cpap.ProjPos)
		max := dir.Mag()
		hit := false
		if max < cpap.ProjSpeed {
			cpap.ProjPos = cpap.LockPos
			hit = true
		} else {
			cpap.ProjPos = cpap.ProjPos.Add(dir.Norm().Scale(cpap.ProjSpeed))
		}
		if hit {
			for _, ent := range g.temp.AllEnts {
				if ent.Pos().Sub(cpap.ProjPos).Mag() < cpap.BlastRadius {
					ent.Stats().ApplyDamage(stats.Damage{stats.DamageFire, 100})
				}
			}
			cpap.Killed = true
		}
	}
}
func (cpap *controlPointAttackProcess) Kill(g *Game) {
	cpap.Killed = true
}
func (cpap *controlPointAttackProcess) Phase() Phase {
	if cpap.Killed {
		return PhaseComplete
	}
	return PhaseRunning
}
func (controlPointAttackProcess) ModifyBase(base stats.Base) stats.Base {
	return base
}
func (controlPointAttackProcess) ModifyDamage(damage stats.Damage) stats.Damage {
	return damage
}
func (controlPointAttackProcess) CauseDamage() stats.Damage {
	return stats.Damage{}
}
func (cpap *controlPointAttackProcess) Draw(id Gid, g *Game, side int) {
	target, ok := g.Ents[cpap.Target].(*Player)
	if !ok {
		return
	}
	position := target.Position
	base.EnableShader("circle")
	base.SetUniformF("circle", "edge", 0.9)
	if cpap.Side == side {
		gl.Color4ub(200, 200, 200, 80)
		texture.Render(
			position.X-50,
			position.Y-50,
			2*50,
			2*50)
	}
	if cpap.Timer >= cpap.FireTime {
		gl.Color4ub(255, 50, 50, 240)
		texture.Render(
			cpap.ProjPos.X-5,
			cpap.ProjPos.Y-5,
			2*5,
			2*5)
	}

	base.EnableShader("")

}
