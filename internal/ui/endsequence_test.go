package ui

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// waitForStatus spins the test until the probe goroutine has had a chance to
// run. Uses Gosched + short sleeps so we don't hang the suite on a stalled
// scheduler.
func waitForStatus(t *testing.T, es *EndSequenceState, want EndStatus) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if es.Status() == want {
			return
		}
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("probe never settled to %v: got %v", want, es.Status())
}

// pinStatus swaps the network-probe client for a deterministic stub for the
// duration of the test. Returns a cleanup function the caller must defer.
func pinStatus(t *testing.T, s EndStatus) func() {
	t.Helper()
	prev := endStatusClient
	endStatusClient = func(ctx context.Context, username string) EndStatus { return s }
	return func() { endStatusClient = prev }
}

// TestEndSequenceTimelineSuccess walks the full successful timeline and
// confirms that EndSequenceDone only fires after every staged beat has had a
// chance to play (flash → cd → build → 3 spinners → git status).
func TestEndSequenceTimelineSuccess(t *testing.T) {
	defer pinStatus(t, EndStatusOK)()

	const startTick = 100
	es := NewEndSequence(startTick, "octocat")
	defer es.Cancel()

	waitForStatus(t, es, EndStatusOK)

	// Compute the expected total ticks from the public constants. If any
	// of the pacing changes, this test follows automatically.
	cdStart := endFlashTicks + endPostFlash
	cdDone := cdStart + endCdTotalTicks()
	buildStart := cdDone + endPostCdPause
	buildDone := buildStart + len(endBuildText)*endTypeRate
	decisionScene := buildDone + endPostBuildPause
	spinners := len(endSpinnerMessages) * endSpinnerTicks
	gitStart := spinners + endPostOutputPause
	gitDone := gitStart + len(endGitText)*endTypeRate
	total := decisionScene + gitDone + endPostGitPause

	// One tick before the finale should NOT be done.
	if EndSequenceDone(es, startTick+total-1) {
		t.Errorf("EndSequenceDone fired too early at scene tick %d", total-1)
	}
	// Exactly at the finale tick we expect to be done.
	if !EndSequenceDone(es, startTick+total) {
		t.Errorf("EndSequenceDone did not fire at scene tick %d", total)
	}
}

// TestEndSequenceTimelineFailure confirms that the offline / no-user paths
// finish much sooner than the success path (we don't make the player sit
// through fake spinners just to be told they're offline).
func TestEndSequenceTimelineFailure(t *testing.T) {
	for _, status := range []EndStatus{EndStatusOffline, EndStatusNoUser} {
		t.Run(statusName(status), func(t *testing.T) {
			defer pinStatus(t, status)()
			const startTick = 50
			es := NewEndSequence(startTick, "ghost")
			defer es.Cancel()
			waitForStatus(t, es, status)

			cdStart := endFlashTicks + endPostFlash
			cdDone := cdStart + endCdTotalTicks()
			buildStart := cdDone + endPostCdPause
			buildDone := buildStart + len(endBuildText)*endTypeRate
			decisionScene := buildDone + endPostBuildPause
			total := decisionScene + endFailureTotalTicks()

			if EndSequenceDone(es, startTick+total-1) {
				t.Errorf("%v: done too early at scene tick %d", status, total-1)
			}
			if !EndSequenceDone(es, startTick+total) {
				t.Errorf("%v: not done at scene tick %d", status, total)
			}

			// The failure path must be strictly shorter than the
			// success path — otherwise we're punishing the user.
			successTotal := decisionScene +
				len(endSpinnerMessages)*endSpinnerTicks +
				endPostOutputPause +
				len(endGitText)*endTypeRate +
				endPostGitPause
			if total >= successTotal {
				t.Errorf("%v: failure timeline %d >= success %d (should be shorter)",
					status, total, successTotal)
			}
		})
	}
}

// TestEndSequencePendingDoesNotComplete makes sure a stalled probe leaves
// the sequence stuck in the decision wait window — until the timeout grace
// period kicks it over to the offline fail path.
func TestEndSequencePendingDoesNotComplete(t *testing.T) {
	// Stub keeps the probe pending by never returning a definitive answer
	// before the test ends. Cancel will tear it down.
	prev := endStatusClient
	done := make(chan struct{})
	endStatusClient = func(ctx context.Context, username string) EndStatus {
		<-done
		return EndStatusOK
	}
	t.Cleanup(func() { close(done); endStatusClient = prev })

	const startTick = 0
	es := NewEndSequence(startTick, "")
	defer es.Cancel()

	// Mid-decision-window scene tick: probe pending → should not be done.
	cdStart := endFlashTicks + endPostFlash
	cdDone := cdStart + endCdTotalTicks()
	buildStart := cdDone + endPostCdPause
	buildDone := buildStart + len(endBuildText)*endTypeRate
	decisionScene := buildDone + endPostBuildPause
	mid := decisionScene + endStatusTimeout/2
	if EndSequenceDone(es, mid) {
		t.Errorf("EndSequenceDone fired while probe was still pending at tick %d", mid)
	}

	// After the grace period plus the failure hold, we should fall through
	// to the offline branch and finish.
	timeoutDone := decisionScene + endStatusTimeout + endFailureTotalTicks()
	if !EndSequenceDone(es, timeoutDone) {
		t.Errorf("EndSequenceDone did not fire after probe timeout at tick %d", timeoutDone)
	}
}

func statusName(s EndStatus) string {
	switch s {
	case EndStatusOK:
		return "ok"
	case EndStatusOffline:
		return "offline"
	case EndStatusNoUser:
		return "no_user"
	default:
		return "pending"
	}
}

// TestEndCdScheduleIsFasterAndJittery proves two properties of the jittery
// per-character typing schedule for `cd developers/developers/developers`:
//  1. It is meaningfully faster than the old uniform pace would have been
//     (otherwise we haven't actually sped anything up).
//  2. It uses more than one delta between consecutive characters (otherwise
//     it's a constant-rate metronome dressed up in random-numbers clothes).
func TestEndCdScheduleIsFasterAndJittery(t *testing.T) {
	if got, want := len(endCdSchedule), len(endCdText); got != want {
		t.Fatalf("schedule length %d != text length %d", got, want)
	}
	uniform := len(endCdText) * endTypeRate
	if endCdTotalTicks() >= uniform {
		t.Errorf("schedule is not faster than uniform: %d ticks vs uniform %d",
			endCdTotalTicks(), uniform)
	}

	// Reject schedules whose first-difference distribution collapses to a
	// single value — that's a uniform schedule by another name.
	seen := map[int]int{}
	prev := 0
	for _, t := range endCdSchedule {
		seen[t-prev]++
		prev = t
	}
	if len(seen) < 2 {
		t.Errorf("schedule is not jittery — only one delta seen: %v", seen)
	}

	// Sanity: monotonic non-decreasing.
	for i := 1; i < len(endCdSchedule); i++ {
		if endCdSchedule[i] < endCdSchedule[i-1] {
			t.Errorf("schedule must be monotonic; sched[%d]=%d < sched[%d]=%d",
				i, endCdSchedule[i], i-1, endCdSchedule[i-1])
		}
	}
}
