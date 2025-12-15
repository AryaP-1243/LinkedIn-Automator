package stealth

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/linkedin-automation/pkg/config"
	"github.com/linkedin-automation/pkg/logger"
)

type ScrollController struct {
	config *config.ScrollingConfig
	timing *TimingController
	log    *logger.Logger
	rand   *rand.Rand
}

type ScrollAction struct {
	Delta     int
	Duration  time.Duration
	Direction string
}

func NewScrollController(cfg *config.ScrollingConfig, timing *TimingController) *ScrollController {
	return &ScrollController{
		config: cfg,
		timing: timing,
		log:    logger.WithComponent("scroll"),
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *ScrollController) GenerateScrollSequence(totalDistance int) []ScrollAction {
	if !s.config.Enabled {
		return []ScrollAction{{Delta: totalDistance, Duration: 0, Direction: "down"}}
	}

	actions := make([]ScrollAction, 0)
	remaining := totalDistance
	direction := "down"

	for remaining > 0 {
		speed := s.config.MinScrollSpeed +
			s.rand.Intn(s.config.MaxScrollSpeed-s.config.MinScrollSpeed+1)

		scrollAmount := s.calculateScrollAmount(remaining, speed)

		if remaining > scrollAmount*2 && s.rand.Float64() < s.config.ScrollBackChance {
			backAmount := scrollAmount / 3
			actions = append(actions, ScrollAction{
				Delta:     backAmount,
				Duration:  s.calculateScrollDuration(backAmount),
				Direction: "up",
			})
		}

		actions = append(actions, ScrollAction{
			Delta:     scrollAmount,
			Duration:  s.calculateScrollDuration(scrollAmount),
			Direction: direction,
		})

		remaining -= scrollAmount

		if s.rand.Float64() < s.config.PauseChance {
			pauseDuration := time.Duration(500+s.rand.Intn(2000)) * time.Millisecond
			actions = append(actions, ScrollAction{
				Delta:     0,
				Duration:  pauseDuration,
				Direction: "pause",
			})
		}
	}

	return actions
}

func (s *ScrollController) calculateScrollAmount(remaining, speed int) int {
	baseAmount := speed + s.rand.Intn(speed/2)

	variation := float64(baseAmount) * 0.2 * (s.rand.Float64()*2 - 1)
	amount := int(float64(baseAmount) + variation)

	if amount > remaining {
		amount = remaining
	}

	if amount < 10 {
		amount = 10
	}

	return amount
}

func (s *ScrollController) calculateScrollDuration(distance int) time.Duration {
	if s.config.SmoothScrolling {
		baseDuration := float64(distance) / 2.0
		variation := baseDuration * 0.3 * (s.rand.Float64()*2 - 1)
		return time.Duration(baseDuration+variation) * time.Millisecond
	}

	return time.Duration(50+s.rand.Intn(100)) * time.Millisecond
}

func (s *ScrollController) GenerateSmoothScrollSteps(totalDelta int) []int {
	if !s.config.SmoothScrolling {
		return []int{totalDelta}
	}

	numSteps := int(math.Abs(float64(totalDelta)) / 20)
	if numSteps < 5 {
		numSteps = 5
	}
	if numSteps > 50 {
		numSteps = 50
	}

	steps := make([]int, numSteps)
	remaining := totalDelta

	for i := 0; i < numSteps; i++ {
		t := float64(i) / float64(numSteps-1)

		eased := 1 - math.Pow(1-t, 3)

		if i == numSteps-1 {
			steps[i] = remaining
		} else {
			step := int(float64(totalDelta) * (eased - float64(totalDelta-remaining)/float64(totalDelta)))

			variation := int(float64(step) * 0.1 * (s.rand.Float64()*2 - 1))
			step += variation

			if step > remaining && remaining > 0 {
				step = remaining
			}
			if step < -remaining && remaining < 0 {
				step = -remaining
			}

			steps[i] = step
			remaining -= step
		}
	}

	return steps
}

func (s *ScrollController) ExecuteScroll(ctx context.Context, scrollFn func(delta int) error, actions []ScrollAction) error {
	for _, action := range actions {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if action.Direction == "pause" {
			if err := s.timing.Sleep(ctx, action.Duration); err != nil {
				return err
			}
			continue
		}

		delta := action.Delta
		if action.Direction == "up" {
			delta = -delta
		}

		if s.config.SmoothScrolling {
			steps := s.GenerateSmoothScrollSteps(delta)
			stepDelay := action.Duration / time.Duration(len(steps))

			for _, step := range steps {
				if err := scrollFn(step); err != nil {
					return err
				}
				if err := s.timing.Sleep(ctx, stepDelay); err != nil {
					return err
				}
			}
		} else {
			if err := scrollFn(delta); err != nil {
				return err
			}
			if err := s.timing.Sleep(ctx, action.Duration); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *ScrollController) RandomViewportScroll() ScrollAction {
	delta := 100 + s.rand.Intn(300)
	direction := "down"
	if s.rand.Float64() < 0.3 {
		direction = "up"
	}

	return ScrollAction{
		Delta:     delta,
		Duration:  s.calculateScrollDuration(delta),
		Direction: direction,
	}
}

func (s *ScrollController) ScrollToElement(currentY, targetY, viewportHeight int) []ScrollAction {
	distance := targetY - currentY - viewportHeight/3

	if distance > 0 {
		return s.GenerateScrollSequence(distance)
	} else if distance < -viewportHeight/2 {
		actions := s.GenerateScrollSequence(-distance)
		for i := range actions {
			if actions[i].Direction == "down" {
				actions[i].Direction = "up"
			} else if actions[i].Direction == "up" {
				actions[i].Direction = "down"
			}
		}
		return actions
	}

	return nil
}
