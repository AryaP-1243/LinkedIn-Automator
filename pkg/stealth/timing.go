package stealth

import (
        "context"
        "math/rand"
        "time"

        "github.com/linkedin-automation/pkg/config"
        "github.com/linkedin-automation/pkg/logger"
)

type TimingController struct {
        config *config.TimingConfig
        log    *logger.Logger
        rand   *rand.Rand
}

func NewTimingController(cfg *config.TimingConfig) *TimingController {
        return &TimingController{
                config: cfg,
                log:    logger.WithComponent("timing"),
                rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
        }
}

func (t *TimingController) RandomDelay(min, max time.Duration) time.Duration {
        if min >= max {
                return min
        }

        base := min + time.Duration(t.rand.Int63n(int64(max-min)))

        variation := float64(base) * t.config.HumanVariation * (t.rand.Float64()*2 - 1)
        final := base + time.Duration(variation)

        if final < min {
                final = min
        }

        return final
}

func (t *TimingController) ActionDelay() time.Duration {
        return t.RandomDelay(t.config.MinActionDelay, t.config.MaxActionDelay)
}

func (t *TimingController) ThinkDelay() time.Duration {
        return t.RandomDelay(t.config.MinThinkTime, t.config.MaxThinkTime)
}

func (t *TimingController) PageLoadDelay() time.Duration {
        base := t.config.PageLoadWait
        variation := time.Duration(float64(base) * t.config.HumanVariation * t.rand.Float64())
        return base + variation
}

func (t *TimingController) Sleep(ctx context.Context, d time.Duration) error {
        select {
        case <-ctx.Done():
                return ctx.Err()
        case <-time.After(d):
                return nil
        }
}

func (t *TimingController) SleepAction(ctx context.Context) error {
        return t.Sleep(ctx, t.ActionDelay())
}

func (t *TimingController) SleepThink(ctx context.Context) error {
        return t.Sleep(ctx, t.ThinkDelay())
}

func (t *TimingController) SleepPageLoad(ctx context.Context) error {
        return t.Sleep(ctx, t.PageLoadDelay())
}

func (t *TimingController) SleepWithJitter(ctx context.Context, base time.Duration) error {
        jitter := time.Duration(float64(base) * 0.2 * (t.rand.Float64()*2 - 1))
        return t.Sleep(ctx, base+jitter)
}

func (t *TimingController) GaussianDelay(mean, stddev time.Duration) time.Duration {
        z := t.rand.NormFloat64()

        delay := float64(mean) + z*float64(stddev)

        if delay < float64(mean)/4 {
                delay = float64(mean) / 4
        }
        if delay > float64(mean)*4 {
                delay = float64(mean) * 4
        }

        return time.Duration(delay)
}

func (t *TimingController) ExponentialBackoff(attempt int, base time.Duration, maxDelay time.Duration) time.Duration {
        delay := base * time.Duration(1<<uint(attempt))

        if delay > maxDelay {
                delay = maxDelay
        }

        jitter := time.Duration(float64(delay) * 0.3 * t.rand.Float64())
        return delay + jitter
}

type ActionTimer struct {
        timing     *TimingController
        lastAction time.Time
        minGap     time.Duration
}

func NewActionTimer(timing *TimingController, minGap time.Duration) *ActionTimer {
        return &ActionTimer{
                timing:     timing,
                lastAction: time.Time{},
                minGap:     minGap,
        }
}

func (a *ActionTimer) WaitForNext(ctx context.Context) error {
        if a.lastAction.IsZero() {
                a.lastAction = time.Now()
                return nil
        }

        elapsed := time.Since(a.lastAction)
        if elapsed < a.minGap {
                waitTime := a.minGap - elapsed
                waitTime += a.timing.RandomDelay(0, waitTime/2)
                if err := a.timing.Sleep(ctx, waitTime); err != nil {
                        return err
                }
        }

        a.lastAction = time.Now()
        return nil
}

func (a *ActionTimer) Record() {
        a.lastAction = time.Now()
}
