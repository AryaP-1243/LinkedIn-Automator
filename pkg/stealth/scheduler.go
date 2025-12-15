package stealth

import (
	"context"
	"math/rand"
	"time"

	"github.com/linkedin-automation/pkg/config"
	"github.com/linkedin-automation/pkg/logger"
)

type ActivityScheduler struct {
	config *config.ScheduleConfig
	log    *logger.Logger
	rand   *rand.Rand
	
	lastBreak     time.Time
	activityStart time.Time
	breaksTaken   int
}

func NewActivityScheduler(cfg *config.ScheduleConfig) *ActivityScheduler {
	return &ActivityScheduler{
		config:        cfg,
		log:           logger.WithComponent("scheduler"),
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
		activityStart: time.Now(),
	}
}

func (s *ActivityScheduler) IsWithinWorkingHours() bool {
	if !s.config.Enabled {
		return true
	}

	loc, err := time.LoadLocation(s.config.Timezone)
	if err != nil {
		s.log.Warn("Failed to load timezone %s, using local time", s.config.Timezone)
		loc = time.Local
	}

	now := time.Now().In(loc)
	hour := now.Hour()
	weekday := int(now.Weekday())

	isWorkDay := false
	for _, day := range s.config.WorkDays {
		if day == weekday {
			isWorkDay = true
			break
		}
	}

	if !isWorkDay {
		s.log.Debug("Not a work day (weekday=%d)", weekday)
		return false
	}

	isWorkHour := hour >= s.config.StartHour && hour < s.config.EndHour

	if !isWorkHour {
		s.log.Debug("Outside working hours (hour=%d, range=%d-%d)", hour, s.config.StartHour, s.config.EndHour)
	}

	return isWorkHour
}

func (s *ActivityScheduler) WaitForWorkingHours(ctx context.Context) error {
	if s.IsWithinWorkingHours() {
		return nil
	}

	loc, _ := time.LoadLocation(s.config.Timezone)
	if loc == nil {
		loc = time.Local
	}

	now := time.Now().In(loc)
	nextStart := s.getNextWorkingTime(now, loc)

	waitDuration := time.Until(nextStart)
	s.log.Info("Waiting until working hours resume at %s (in %s)", nextStart.Format(time.RFC3339), waitDuration)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitDuration):
		return nil
	}
}

func (s *ActivityScheduler) getNextWorkingTime(from time.Time, loc *time.Location) time.Time {
	current := from

	for i := 0; i < 8; i++ {
		weekday := int(current.Weekday())
		isWorkDay := false
		for _, day := range s.config.WorkDays {
			if day == weekday {
				isWorkDay = true
				break
			}
		}

		if isWorkDay {
			startTime := time.Date(current.Year(), current.Month(), current.Day(),
				s.config.StartHour, 0, 0, 0, loc)

			if current.Before(startTime) {
				return startTime
			}

			if current.Hour() < s.config.EndHour {
				return current
			}
		}

		current = time.Date(current.Year(), current.Month(), current.Day()+1,
			0, 0, 0, 0, loc)
	}

	return current
}

func (s *ActivityScheduler) ShouldTakeBreak() bool {
	if !s.config.RandomBreaks {
		return false
	}

	timeSinceLastBreak := time.Since(s.lastBreak)
	timeSinceStart := time.Since(s.activityStart)

	breakInterval := 30*time.Minute + time.Duration(s.rand.Intn(30))*time.Minute

	if timeSinceStart < 15*time.Minute {
		return false
	}

	if timeSinceLastBreak > breakInterval {
		return true
	}

	if timeSinceLastBreak > 20*time.Minute && s.rand.Float64() < 0.1 {
		return true
	}

	return false
}

func (s *ActivityScheduler) TakeBreak(ctx context.Context) error {
	breakDuration := s.calculateBreakDuration()
	s.log.Info("Taking a break for %s", breakDuration)

	s.lastBreak = time.Now()
	s.breaksTaken++

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(breakDuration):
		s.log.Info("Break finished, resuming activity")
		return nil
	}
}

func (s *ActivityScheduler) calculateBreakDuration() time.Duration {
	minBreak := 2 * time.Minute
	maxBreak := 15 * time.Minute

	if s.breaksTaken == 0 {
		minBreak = 1 * time.Minute
		maxBreak = 5 * time.Minute
	} else if s.breaksTaken >= 3 {
		minBreak = 5 * time.Minute
		maxBreak = 20 * time.Minute
	}

	breakRange := int64(maxBreak - minBreak)
	randomDuration := time.Duration(s.rand.Int63n(breakRange))

	return minBreak + randomDuration
}

func (s *ActivityScheduler) GetTimeUntilEndOfDay() time.Duration {
	if !s.config.Enabled {
		return 24 * time.Hour
	}

	loc, _ := time.LoadLocation(s.config.Timezone)
	if loc == nil {
		loc = time.Local
	}

	now := time.Now().In(loc)
	endTime := time.Date(now.Year(), now.Month(), now.Day(),
		s.config.EndHour, 0, 0, 0, loc)

	if now.After(endTime) {
		return 0
	}

	return time.Until(endTime)
}

func (s *ActivityScheduler) CalculateDailyActivityWindow() (start, end time.Time) {
	loc, _ := time.LoadLocation(s.config.Timezone)
	if loc == nil {
		loc = time.Local
	}

	now := time.Now().In(loc)

	startVariation := s.rand.Intn(30)
	endVariation := s.rand.Intn(30)

	start = time.Date(now.Year(), now.Month(), now.Day(),
		s.config.StartHour, startVariation, 0, 0, loc)

	end = time.Date(now.Year(), now.Month(), now.Day(),
		s.config.EndHour, 0, 0, 0, loc)
	end = end.Add(-time.Duration(endVariation) * time.Minute)

	return start, end
}

func (s *ActivityScheduler) Stats() map[string]interface{} {
	return map[string]interface{}{
		"activity_duration": time.Since(s.activityStart).String(),
		"breaks_taken":      s.breaksTaken,
		"last_break":        s.lastBreak.Format(time.RFC3339),
		"is_working_hours":  s.IsWithinWorkingHours(),
	}
}
