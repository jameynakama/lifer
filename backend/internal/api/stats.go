package api

import (
	"log"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/store"
)

// statsFSRS is a package-level FSRS instance used by the stats handler.
// FSRS instances are read-only after construction, so sharing is safe.
var statsFSRS = fsrs.NewFSRS(fsrs.DefaultParam())

// retrievability evaluates the FSRS forgetting curve via the same library
// that schedules reviews, so stats can never disagree with the scheduler.
func retrievability(stability float64, lastReview, at time.Time) float64 {
	return statsFSRS.GetRetrievability(fsrs.Card{
		State:      fsrs.Review,
		Stability:  stability,
		LastReview: lastReview,
	}, at)
}

// expectedRecall is the probabilistic number of cards remembered at time at:
// the rounded sum of each card's retrievability.
func expectedRecall(cards []store.GetKnownCardsRow, at time.Time) int {
	var sum float64
	for _, c := range cards {
		if !c.LastReview.Valid {
			continue
		}
		sum += retrievability(c.Stability, c.LastReview.Time, at)
	}
	return int(math.Round(sum))
}

type statsSpecies struct {
	EbirdCode      string `json:"ebird_code"`
	CommonName     string `json:"common_name"`
	ScientificName string `json:"scientific_name"`
}

type statsTotals struct {
	Species       int64 `json:"species"`
	Cards         int64 `json:"cards"`
	Known         int64 `json:"known"`
	Reviews       int64 `json:"reviews"`
	Lapses        int64 `json:"lapses"`
	Attempts      int64 `json:"attempts"`
	Correct       int64 `json:"correct"`
	ReviewsLast7d int64 `json:"reviews_last_7d"`
}

type statsProgress struct {
	NotSeen    int64 `json:"not_seen"`
	Learning   int64 `json:"learning"`
	Known      int64 `json:"known"`
	Relearning int64 `json:"relearning"`
}

type statsLane struct {
	Cards int64 `json:"cards"`
	Known int64 `json:"known"`
}

type statsGap struct {
	statsSpecies
	KnownLane string `json:"known_lane"`
	WeakLane  string `json:"weak_lane"`
}

type statsLanes struct {
	Audio statsLane  `json:"audio"`
	Image statsLane  `json:"image"`
	Gaps  []statsGap `json:"gaps"`
}

type statsConfusion struct {
	Actual  statsSpecies `json:"actual"`
	Guessed statsSpecies `json:"guessed"`
	Count   int64        `json:"count"`
}

type statsFamily struct {
	Family   string `json:"family"`
	Attempts int64  `json:"attempts"`
	Correct  int64  `json:"correct"`
}

type statsFading struct {
	statsSpecies
	Lane           string  `json:"lane"`
	Retrievability float64 `json:"retrievability"`
	DueInDays      int     `json:"due_in_days"`
}

type statsRemember struct {
	Now      int `json:"now"`
	InAWeek  int `json:"in_a_week"`
	InAMonth int `json:"in_a_month"`
}

type statsHardMedia struct {
	statsSpecies
	Lane     string `json:"lane"`
	MediaID  string `json:"media_id"`
	MediaURL string `json:"media_url"`
	Attempts int64  `json:"attempts"`
	Correct  int64  `json:"correct"`
}

type statsResponse struct {
	Totals     statsTotals      `json:"totals"`
	Progress   statsProgress    `json:"progress"`
	Lanes      *statsLanes      `json:"lanes,omitempty"`
	Confusions []statsConfusion `json:"confusions"`
	Families   []statsFamily    `json:"families"`
	Fading     []statsFading    `json:"fading"`
	Remember   statsRemember    `json:"remember"`
	HardMedia  []statsHardMedia `json:"hard_media"`
}

func laneArg(lane string) pgtype.Text {
	if lane == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: lane, Valid: true}
}

// knownOf extracts the known count from a slice of bucket rows.
func knownOf(rows []store.GetCardStateCountsRow) int64 {
	for _, r := range rows {
		if r.Bucket == "known" {
			return r.Count
		}
	}
	return 0
}

func (h *Handler) getStats(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	lane := r.URL.Query().Get("lane")
	if lane != "" && lane != "audio" && lane != "image" {
		writeError(w, http.StatusBadRequest, "lane must be audio or image")
		return
	}

	ctx := r.Context()
	la := laneArg(lane)

	fail := func(what string, err error) {
		log.Printf("getStats %s: %v", what, err)
		writeError(w, http.StatusInternalServerError, "server error")
	}

	totalsRow, err := h.queries.GetCardTotals(ctx, store.GetCardTotalsParams{UserID: userID, Lane: la})
	if err != nil {
		fail("GetCardTotals", err)
		return
	}

	buckets, err := h.queries.GetCardStateCounts(ctx, store.GetCardStateCountsParams{UserID: userID, Lane: la})
	if err != nil {
		fail("GetCardStateCounts", err)
		return
	}

	known, err := h.queries.GetKnownCards(ctx, store.GetKnownCardsParams{UserID: userID, Lane: la})
	if err != nil {
		fail("GetKnownCards", err)
		return
	}

	confusionRows, err := h.queries.GetConfusionPairs(ctx, store.GetConfusionPairsParams{UserID: userID, Lane: la})
	if err != nil {
		fail("GetConfusionPairs", err)
		return
	}

	familyRows, err := h.queries.GetFamilyAccuracy(ctx, store.GetFamilyAccuracyParams{UserID: userID, Lane: la})
	if err != nil {
		fail("GetFamilyAccuracy", err)
		return
	}

	hardMediaRows, err := h.queries.GetHardMedia(ctx, store.GetHardMediaParams{UserID: userID, Lane: la})
	if err != nil {
		fail("GetHardMedia", err)
		return
	}

	accuracy, err := h.queries.GetReviewAccuracy(ctx, store.GetReviewAccuracyParams{UserID: userID, Lane: la})
	if err != nil {
		fail("GetReviewAccuracy", err)
		return
	}

	sevenDaysAgo := pgtype.Timestamptz{}
	if err := sevenDaysAgo.Scan(time.Now().Add(-7 * 24 * time.Hour)); err != nil {
		fail("scan sevenDaysAgo", err)
		return
	}
	reviewsLast7d, err := h.queries.CountReviewsSince(ctx, store.CountReviewsSinceParams{
		UserID:     userID,
		ReviewedAt: sevenDaysAgo,
		Lane:       la,
	})
	if err != nil {
		fail("CountReviewsSince", err)
		return
	}

	// Build progress from bucket rows.
	var progress statsProgress
	for _, b := range buckets {
		switch b.Bucket {
		case "not_seen":
			progress.NotSeen = b.Count
		case "learning":
			progress.Learning = b.Count
		case "known":
			progress.Known = b.Count
		case "relearning":
			progress.Relearning = b.Count
		}
	}

	now := time.Now()

	// Build Fading: known cards sorted by retrievability ascending (worst first).
	type fadingEntry struct {
		row  store.GetKnownCardsRow
		retr float64
	}
	fadingAll := make([]fadingEntry, 0, len(known))
	for _, c := range known {
		if !c.LastReview.Valid {
			continue
		}
		retr := retrievability(c.Stability, c.LastReview.Time, now)
		fadingAll = append(fadingAll, fadingEntry{row: c, retr: retr})
	}
	sort.Slice(fadingAll, func(i, j int) bool {
		return fadingAll[i].retr < fadingAll[j].retr
	})
	maxFading := 10
	if len(fadingAll) < maxFading {
		maxFading = len(fadingAll)
	}
	fading := make([]statsFading, 0, maxFading)
	for _, fe := range fadingAll[:maxFading] {
		// DueInDays is clamped to 0 for overdue cards; the FE renders 0 as "due
		// now". Note: fading is sorted by retrievability (forgetting curve), not
		// by the scheduler's due date -- the two orderings can differ.
		dueInDays := 0
		if fe.row.Due.Valid {
			d := fe.row.Due.Time.Sub(now).Hours() / 24
			if d > 0 {
				dueInDays = int(math.Ceil(d))
			}
		}
		fading = append(fading, statsFading{
			statsSpecies: statsSpecies{
				EbirdCode:      fe.row.SpeciesCode,
				CommonName:     fe.row.CommonName,
				ScientificName: fe.row.ScientificName,
			},
			Lane:           fe.row.Lane,
			Retrievability: math.Round(fe.retr*10000) / 10000,
			DueInDays:      dueInDays,
		})
	}

	// Build confusions.
	confusions := make([]statsConfusion, 0, len(confusionRows))
	for _, c := range confusionRows {
		confusions = append(confusions, statsConfusion{
			Actual: statsSpecies{
				EbirdCode:      c.SpeciesCode,
				CommonName:     c.ActualCommonName,
				ScientificName: c.ActualScientificName,
			},
			Guessed: statsSpecies{
				EbirdCode:      c.GuessedSpeciesCode,
				CommonName:     c.GuessedCommonName,
				ScientificName: c.GuessedScientificName,
			},
			Count: c.Count,
		})
	}

	// Build families.
	families := make([]statsFamily, 0, len(familyRows))
	for _, f := range familyRows {
		families = append(families, statsFamily{
			Family:   f.Family,
			Attempts: f.Attempts,
			Correct:  f.Correct,
		})
	}

	// Build hard media.
	hardMedia := make([]statsHardMedia, 0, len(hardMediaRows))
	for _, m := range hardMediaRows {
		hardMedia = append(hardMedia, statsHardMedia{
			statsSpecies: statsSpecies{
				EbirdCode:      m.SpeciesCode,
				CommonName:     m.CommonName,
				ScientificName: m.ScientificName,
			},
			Lane:     m.Lane,
			MediaID:  m.MediaID,
			MediaURL: m.MediaUrl,
			Attempts: m.Attempts,
			Correct:  m.Correct,
		})
	}

	resp := statsResponse{
		Totals: statsTotals{
			Species:       totalsRow.Species,
			Cards:         totalsRow.Cards,
			Known:         progress.Known,
			Reviews:       totalsRow.Reviews,
			Lapses:        totalsRow.Lapses,
			Attempts:      accuracy.Attempts,
			Correct:       accuracy.Correct,
			ReviewsLast7d: reviewsLast7d,
		},
		Progress:   progress,
		Confusions: confusions,
		Families:   families,
		Fading:     fading,
		Remember: statsRemember{
			Now:      expectedRecall(known, now),
			InAWeek:  expectedRecall(known, now.Add(7*24*time.Hour)),
			InAMonth: expectedRecall(known, now.Add(30*24*time.Hour)),
		},
		HardMedia: hardMedia,
	}

	// Combined mode only: add per-lane breakdown and gaps.
	if lane == "" {
		audioTotals, err := h.queries.GetCardTotals(ctx, store.GetCardTotalsParams{
			UserID: userID, Lane: pgtype.Text{String: "audio", Valid: true},
		})
		if err != nil {
			fail("GetCardTotals audio", err)
			return
		}
		audioBuckets, err := h.queries.GetCardStateCounts(ctx, store.GetCardStateCountsParams{
			UserID: userID, Lane: pgtype.Text{String: "audio", Valid: true},
		})
		if err != nil {
			fail("GetCardStateCounts audio", err)
			return
		}
		imageTotals, err := h.queries.GetCardTotals(ctx, store.GetCardTotalsParams{
			UserID: userID, Lane: pgtype.Text{String: "image", Valid: true},
		})
		if err != nil {
			fail("GetCardTotals image", err)
			return
		}
		imageBuckets, err := h.queries.GetCardStateCounts(ctx, store.GetCardStateCountsParams{
			UserID: userID, Lane: pgtype.Text{String: "image", Valid: true},
		})
		if err != nil {
			fail("GetCardStateCounts image", err)
			return
		}
		gapRows, err := h.queries.GetLaneGaps(ctx, userID)
		if err != nil {
			fail("GetLaneGaps", err)
			return
		}
		gaps := make([]statsGap, 0, len(gapRows))
		for _, g := range gapRows {
			gaps = append(gaps, statsGap{
				statsSpecies: statsSpecies{
					EbirdCode:      g.SpeciesCode,
					CommonName:     g.CommonName,
					ScientificName: g.ScientificName,
				},
				KnownLane: g.KnownLane,
				WeakLane:  g.WeakLane,
			})
		}
		resp.Lanes = &statsLanes{
			Audio: statsLane{Cards: audioTotals.Cards, Known: knownOf(audioBuckets)},
			Image: statsLane{Cards: imageTotals.Cards, Known: knownOf(imageBuckets)},
			Gaps:  gaps,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
