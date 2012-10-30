package main

import (
  "encoding/gob"
  "fmt"
  "math"
  "runningwild/tron/base"
)

type Ability interface {
  Activate(player *Player, params map[string]int) Process
}

type Process interface {
  // Request returns the most mana that this Process could use right now.
  // Some Processes can operate at any amount of mana, and some will need to
  // get all of their requested mana before they are able to do anything.
  Request() Mana

  // Supplies mana to the Process and returns the unused portion.
  Supply(Mana) Mana

  Think(*Player, *Game)

  // Kills a process.  Any Killed process will return true on any future
  // calls to Complete().
  Kill(player *Player)

  Complete() bool
}

// BLINK
// Causes the player to disappear for [frames] frames, where a frame is 16ms.
// Cost 50000 + [frames]^2 blue mana.
func init() {
  gob.Register(blinkAbility{})
  gob.Register(&blinkProcess{})
}

type blinkAbility struct {
}

func (a *blinkAbility) Activate(player *Player, params map[string]int) Process {
  if len(params) != 1 {
    panic(fmt.Sprintf("Blink requires exactly one parameter, not %v", params))
  }
  if _, ok := params["frames"]; !ok {
    panic(fmt.Sprintf("Blink requires [frames] to be specified, not %v", params))
  }
  frames := params["frames"]
  if frames < 0 {
    panic(fmt.Sprintf("Blink requires [frames] > 0, not %d", frames))
  }
  return &blinkProcess{
    Frames:    int32(frames),
    Remaining: Mana{0, 50000 + float64(frames*frames), 0},
  }
}

type blinkProcess struct {
  Frames    int32
  Remaining Mana
  Killed    bool
}

func (p *blinkProcess) Request() Mana {
  return p.Remaining
}

// Supplies mana to the process.  Any mana that is unused is returned.
func (p *blinkProcess) Supply(supply Mana) Mana {
  for color := range supply {
    if p.Remaining[color] == 0 {
      continue
    }
    if supply[color] == 0 {
      continue
    }
    if supply[color] > p.Remaining[color] {
      supply[color] -= p.Remaining[color]
      p.Remaining[color] = 0
    } else {
      p.Remaining[color] -= supply[color]
      supply[color] = 0
    }
  }
  return supply
}

func (p *blinkProcess) Think(player *Player, game *Game) {
  if p.Remaining.Magnitude() == 0 {
    player.Exile_frames += p.Frames
    p.Frames = 0
  }
}
func (p *blinkProcess) Kill(player *Player) {
  p.Killed = true
}
func (p *blinkProcess) Complete() bool {
  return p.Killed || p.Frames == 0
}

// BURST
// All nearby players are pushed radially outward from this one.  The force
// applied to each player is max(0, [max]*(1 - (x / [radius])^2)).  This fore
// is applied constantly for [frames] frames, or until the continual cost
// cannot be paid.
// Initial cost: [radius] * [force] red mana.
// Continual cost: [frames] red mana per frame.
func init() {
  gob.Register(burstAbility{})
  gob.Register(&burstProcess{})
}

type burstAbility struct {
}

func (a *burstAbility) Activate(player *Player, params map[string]int) Process {
  if len(params) != 2 {
    panic(fmt.Sprintf("Burst requires exactly two parameters, not %v", params))
  }
  for _, req := range []string{"frames", "force"} {
    if _, ok := params[req]; !ok {
      panic(fmt.Sprintf("Burst requires [%s] to be specified, not %v", req, params))
    }
  }
  frames := params["frames"]
  force := params["force"]
  if frames < 0 {
    panic(fmt.Sprintf("Burst requires [frames] > 0, not %d", frames))
  }
  if force < 0 {
    panic(fmt.Sprintf("Burst requires [force] > 0, not %d", force))
  }
  return &burstProcess{
    Frames:            int32(frames),
    Force:             float64(force),
    Remaining_initial: Mana{math.Pow(float64(force)*float64(frames), 2) / 1.0e7, 0, 0},
    Continual:         Mana{float64(force) / 50, 0, 0},
  }
}

type burstProcess struct {
  Frames            int32
  Force             float64
  Remaining_initial Mana
  Continual         Mana
  Killed            bool

  // Counting how long to cast
  count int
}

func (p *burstProcess) Request() Mana {
  if p.Remaining_initial.Magnitude() > 0 {
    return p.Remaining_initial
  }
  return p.Continual
}

// Supplies mana to the process.  Any mana that is unused is returned.
func (p *burstProcess) Supply(supply Mana) Mana {
  if p.Remaining_initial.Magnitude() > 0 {
    p.count++
    for color := range supply {
      if p.Remaining_initial[color] == 0 {
        continue
      }
      if supply[color] == 0 {
        continue
      }
      if supply[color] > p.Remaining_initial[color] {
        supply[color] -= p.Remaining_initial[color]
        p.Remaining_initial[color] = 0
      } else {
        p.Remaining_initial[color] -= supply[color]
        supply[color] = 0
      }
    }
  } else {
    for color := range p.Continual {
      if supply[color] < p.Continual[color] {
        p.Frames = 0
        return supply
      }
    }
    for color := range p.Continual {
      supply[color] -= p.Continual[color]
    }
  }
  return supply
}

func (p *burstProcess) Think(player *Player, game *Game) {
  if p.Remaining_initial.Magnitude() == 0 {
    if p.count > 0 {
      base.Log().Printf("Frames: %d", p.count)
      p.count = -1
    }
    p.Frames--
    for i := range game.Players {
      other := &game.Players[i]
      if other == player {
        continue
      }
      dx := other.X - player.X
      dy := other.Y - player.Y
      dist := math.Sqrt(dx*dx + dy*dy)
      if dist < 1 {
        dist = 1
      }
      acc := p.Force / (dist * other.Mass)
      angle := math.Atan2(dy, dx)
      other.Vx += acc * math.Cos(angle)
      other.Vy += acc * math.Sin(angle)
    }
  }
}
func (p *burstProcess) Kill(player *Player) {
  p.Killed = true
}
func (p *burstProcess) Complete() bool {
  return p.Killed || p.Frames <= 0
}

// NITRO
// Increases Max_acc by up to [inc]/nitro_acc_factor.
// Continual cost: up to [inc]*[inc]/nitro_mana_factor red mana per frame.
const nitro_mana_factor = 1000
const nitro_acc_factor = 2500

func init() {
  gob.Register(nitroAbility{})
  gob.Register(&nitroProcess{})
}

type nitroAbility struct {
}

func (a *nitroAbility) Activate(player *Player, params map[string]int) Process {
  if len(params) != 1 {
    panic(fmt.Sprintf("Nitro requires exactly one parameter, not %v", params))
  }
  for _, req := range []string{"inc"} {
    if _, ok := params[req]; !ok {
      panic(fmt.Sprintf("Nitro requires [%s] to be specified, not %v", req, params))
    }
  }
  inc := params["inc"]
  if inc <= 0 {
    panic(fmt.Sprintf("Nitro requires [inc] > 0, not %d", inc))
  }
  return &nitroProcess{
    Inc:       int32(inc),
    Continual: Mana{float64(inc) * float64(inc) / nitro_mana_factor, 0, 0},
  }
}

type nitroProcess struct {
  Inc       int32
  Continual Mana
  Killed    bool

  Prev_delta float64
  Supplied   Mana
}

func (p *nitroProcess) Request() Mana {
  return p.Continual
}

// Supplies mana to the process.  Any mana that is unused is returned.
func (p *nitroProcess) Supply(supply Mana) Mana {
  for color := range p.Continual {
    if supply[color] < p.Continual[color] {
      p.Supplied[color] += supply[color]
      supply[color] = 0
    } else {
      p.Supplied[color] += p.Continual[color]
      supply[color] -= p.Continual[color]
    }
  }
  return supply
}

func (p *nitroProcess) Think(player *Player, game *Game) {
  player.Max_acc -= p.Prev_delta
  delta := math.Sqrt(p.Supplied.Magnitude()*nitro_mana_factor) / nitro_acc_factor
  base.Log().Printf("Delta: %.3f", delta)
  p.Supplied = Mana{}
  player.Max_acc += delta
  p.Prev_delta = delta
}
func (p *nitroProcess) Kill(player *Player) {
  p.Killed = true
  player.Max_acc -= p.Prev_delta
}
func (p *nitroProcess) Complete() bool {
  return p.Killed
}
