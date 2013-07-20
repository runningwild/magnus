package stats

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"github.com/runningwild/magnus/base"
)

type Dynamic struct {
	Health float64
}

type Base struct {
	Health float64
	Mass   float64

	// Max rate for accelerating and turning.
	Max_turn float64
	Max_acc  float64

	// Max_rate and Influence determine the rate that a player can drain mana
	// from a node a distance D away:
	// Rate(D) = max(0, Max_rate - (D / Influence) ^ 2)
	Max_rate  float64
	Influence float64
}

type DamageKind int

const (
	DamageFire DamageKind = iota
	DamageAcid
	DamageCrushing
)

type Damage struct {
	Kind DamageKind
	Amt  float64
}

type Condition interface {
	// Called any time a base stats is queried, this will modify the base stats
	// only temporarily.
	ModifyBase(base Base) Base

	// Called any time the entity with this condition takes damage.
	ModifyDamage(damage Damage) Damage

	// Returns true if this condition should be removed.
	Terminated() bool

	// Run every frame, this damage is applied to the entity with this condition.
	CauseDamage() Damage
}

type inst struct {
	Base       Base
	Dynamic    Dynamic
	Conditions []Condition
}

type Inst struct {
	inst inst
}

func (s Inst) HealthCur() float64 {
	return s.inst.Dynamic.Health
}

func (s Inst) ModifyBase(base Base) Base {
	for _, condition := range s.inst.Conditions {
		base = condition.ModifyBase(base)
	}
	return base
}
func (s Inst) HealthMax() float64 {
	return s.ModifyBase(s.inst.Base).Health
}
func (s Inst) Mass() float64 {
	return s.ModifyBase(s.inst.Base).Mass
}
func (s Inst) MaxTurn() float64 {
	return s.ModifyBase(s.inst.Base).Max_turn
}
func (s Inst) MaxAcc() float64 {
	return s.ModifyBase(s.inst.Base).Max_acc
}
func (s Inst) MaxRate() float64 {
	return s.ModifyBase(s.inst.Base).Max_rate
}
func (s Inst) Influence() float64 {
	return s.ModifyBase(s.inst.Base).Influence
}

func (s *Inst) SetHealth(health float64) {
	s.inst.Dynamic.Health = health
}
func (s *Inst) ApplyDamage(damage Damage) {
	for _, cond := range s.inst.Conditions {
		damage = cond.ModifyDamage(damage)
	}
	if damage.Amt > 0 {
		s.inst.Dynamic.Health -= damage.Amt
	}
}
func (s *Inst) ApplyCondition(condition Condition) {
	s.inst.Conditions = append(s.inst.Conditions, condition)
}
func (s *Inst) Think() {
	// Allow any conditions to apply damage
	for _, condition := range s.inst.Conditions {
		s.ApplyDamage(condition.CauseDamage())
	}
	s.inst.Conditions = s.inst.Conditions[0:0]
}

// Encoding routines - only support json and gob right now

func (si Inst) MarshalJSON() ([]byte, error) {
	return json.Marshal(si.inst)
}

func (si *Inst) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &si.inst)
}

func (si Inst) GobEncode() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(si.inst)
	return buf.Bytes(), err
}

func (si *Inst) GobDecode(data []byte) error {
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	err := dec.Decode(&si.inst)
	return err
}

func (si *Inst) Copy() *Inst {
	data, err := si.GobEncode()
	if err != nil {
		base.Error().Fatalf("%v", err)
	}
	var si2 Inst
	err = si2.GobDecode(data)
	if err != nil {
		base.Error().Fatalf("%v", err)
	}
	return &si2
}
