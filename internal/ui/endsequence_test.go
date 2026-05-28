package ui

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
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

// pinContribGenerator swaps the contribution-graph generator hook for the
// duration of a test so we can assert on the fact of the call (and on its
// arguments) without doing any real HTTP. Returns a cleanup the caller
// must defer along with a *waitable* counter that the test can wait on.
type contribCall struct {
	username string
	outPath  string
}

func pinContribGenerator(t *testing.T) (calls *[]contribCall, fired chan struct{}, cleanup func()) {
	t.Helper()
	var mu sync.Mutex
	got := make([]contribCall, 0, 2)
	fired = make(chan struct{}, 1)
	prev := endContribGenerator
	endContribGenerator = func(ctx context.Context, username, outPath string) error {
		mu.Lock()
		got = append(got, contribCall{username: username, outPath: outPath})
		mu.Unlock()
		select {
		case fired <- struct{}{}:
		default:
		}
		return nil
	}
	cleanup = func() { endContribGenerator = prev }
	calls = &got
	return calls, fired, cleanup
}

// TestContribGraphFiresOnceOnSuccess confirms that the spec's "once the
// spinners start, AND username is valid, AND there's internet" gate
// fires the generator exactly one time per end sequence, even when
// MaybeStartContribGraph is called many ticks in a row.
func TestContribGraphFiresOnceOnSuccess(t *testing.T) {
	defer pinStatus(t, EndStatusOK)()
	calls, fired, cleanup := pinContribGenerator(t)
	defer cleanup()

	const startTick = 100
	es := NewEndSequence(startTick, "octocat")
	defer es.Cancel()
	waitForStatus(t, es, EndStatusOK)

	cdStart := endFlashTicks + endPostFlash
	cdDone := cdStart + endCdTotalTicks()
	buildStart := cdDone + endPostCdPause
	buildDone := buildStart + len(endBuildText)*endTypeRate
	decisionScene := buildDone + endPostBuildPause

	// Before the decision scene we should never schedule a fetch — the
	// spinners haven't started yet.
	for tick := startTick; tick < startTick+decisionScene; tick += 5 {
		if es.MaybeStartContribGraph(tick) {
			t.Fatalf("graph fired before decision scene at tick %d", tick)
		}
	}

	// First tick at or after the decision scene fires the generator…
	if !es.MaybeStartContribGraph(startTick + decisionScene) {
		t.Errorf("graph did not fire at decision-scene tick")
	}
	// …and every subsequent tick is a no-op (sync.Once does its job).
	for i := 1; i < 10; i++ {
		if es.MaybeStartContribGraph(startTick + decisionScene + i) {
			t.Errorf("graph fired more than once (extra firing at +%d)", i)
		}
	}

	select {
	case <-fired:
	case <-time.After(2 * time.Second):
		t.Fatal("generator goroutine never ran")
	}
	// Wait for any second goroutine that might have been scheduled to
	// settle — there shouldn't be one, but the scheduler is async.
	for i := 0; i < 20; i++ {
		runtime.Gosched()
		time.Sleep(2 * time.Millisecond)
	}
	if got := len(*calls); got != 1 {
		t.Errorf("generator call count = %d, want 1", got)
	}
	if got := (*calls)[0]; got.username != "octocat" || got.outPath == "" {
		t.Errorf("generator called with %+v, want username=octocat and a non-empty outPath", got)
	}
}

// TestContribGraphSkippedWhenOffline guards the "and there's internet"
// half of the gate — an Offline probe must never write a file (which
// would be misleading: the player is being told the graph couldn't be
// produced, then we'd silently produce one).
func TestContribGraphSkippedWhenOffline(t *testing.T) {
	for _, s := range []EndStatus{EndStatusOffline, EndStatusNoUser, EndStatusPending} {
		t.Run(statusName(s), func(t *testing.T) {
			if s != EndStatusPending {
				defer pinStatus(t, s)()
			}
			var fireCount int32
			prev := endContribGenerator
			endContribGenerator = func(ctx context.Context, username, outPath string) error {
				atomic.AddInt32(&fireCount, 1)
				return nil
			}
			defer func() { endContribGenerator = prev }()

			const startTick = 0
			es := NewEndSequence(startTick, "octocat")
			defer es.Cancel()
			if s != EndStatusPending {
				waitForStatus(t, es, s)
			}

			cdStart := endFlashTicks + endPostFlash
			cdDone := cdStart + endCdTotalTicks()
			buildStart := cdDone + endPostCdPause
			buildDone := buildStart + len(endBuildText)*endTypeRate
			decisionScene := buildDone + endPostBuildPause

			for tick := 0; tick < decisionScene+endStatusTimeout+50; tick += 7 {
				es.MaybeStartContribGraph(tick)
			}
			// Give any (incorrectly) scheduled goroutine a chance to run.
			for i := 0; i < 20; i++ {
				runtime.Gosched()
				time.Sleep(2 * time.Millisecond)
			}
			if got := atomic.LoadInt32(&fireCount); got != 0 {
				t.Errorf("expected generator to NOT fire for status %v, got %d call(s)",
					s, got)
			}
		})
	}
}

// TestContribGraphSkippedWhenNoUsername mirrors the "username is valid"
// half of the gate — if the player skipped the title-screen handle entry
// there's nothing to fetch.
func TestContribGraphSkippedWhenNoUsername(t *testing.T) {
	defer pinStatus(t, EndStatusOK)()
	var fireCount int32
	prev := endContribGenerator
	endContribGenerator = func(ctx context.Context, username, outPath string) error {
		atomic.AddInt32(&fireCount, 1)
		return nil
	}
	defer func() { endContribGenerator = prev }()

	const startTick = 0
	es := NewEndSequence(startTick, "") // no username
	defer es.Cancel()
	waitForStatus(t, es, EndStatusOK)

	cdStart := endFlashTicks + endPostFlash
	cdDone := cdStart + endCdTotalTicks()
	buildStart := cdDone + endPostCdPause
	buildDone := buildStart + len(endBuildText)*endTypeRate
	decisionScene := buildDone + endPostBuildPause

	for tick := 0; tick < decisionScene+200; tick += 5 {
		if es.MaybeStartContribGraph(tick) {
			t.Fatalf("graph fired with empty username at tick %d", tick)
		}
	}
	// Give any (incorrectly) scheduled goroutine a chance to run.
	for i := 0; i < 20; i++ {
		runtime.Gosched()
		time.Sleep(2 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&fireCount); got != 0 {
		t.Errorf("expected generator to NOT fire with empty username, got %d call(s)", got)
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
