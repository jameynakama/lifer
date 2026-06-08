package api

// Mastery tier thresholds, in FSRS stability (days). The bucketing CASE in
// GetCardStateCounts (store/queries/cards.sql) MUST mirror these literals.
// "banked" (stability >= bankedStability) is the shared "genuinely learned"
// cut used by Fading, lane gaps, and Ear-vs-Eye.
const (
	stabFledgling   = 1.0  // >= this -> Fledgling
	stabJuvenile    = 7.0  // >= this -> Juvenile
	stabImmature    = 30.0 // >= this -> Immature
	stabAdult       = 90.0 // >= this -> Adult
	bankedStability = stabJuvenile
)

// tierStages is the ordered ladder, Egg (never quizzed) first.
var tierStages = []string{"egg", "nestling", "fledgling", "juvenile", "immature", "adult"}

// tierWindow describes how to select the cards in one stage.
type tierWindow struct {
	egg       bool    // egg: reps = 0, ignore bounds
	min       float64 // inclusive lower stability bound
	max       float64 // exclusive upper stability bound (ignored if unbounded)
	unbounded bool    // true for the top stage (no upper bound)
}

// tierWindowFor maps a stage name to its selection window. ok=false if unknown.
func tierWindowFor(stage string) (tierWindow, bool) {
	switch stage {
	case "egg":
		return tierWindow{egg: true}, true
	case "nestling":
		return tierWindow{min: 0, max: stabFledgling}, true
	case "fledgling":
		return tierWindow{min: stabFledgling, max: stabJuvenile}, true
	case "juvenile":
		return tierWindow{min: stabJuvenile, max: stabImmature}, true
	case "immature":
		return tierWindow{min: stabImmature, max: stabAdult}, true
	case "adult":
		return tierWindow{min: stabAdult, unbounded: true}, true
	default:
		return tierWindow{}, false
	}
}
