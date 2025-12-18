package stealth

import (
	"context"
	"math/rand"
	"strings"
	"time"
	"unicode"

	"github.com/linkedin-automation/pkg/config"
	"github.com/linkedin-automation/pkg/logger"
)

type TypingController struct {
	config *config.TypingConfig
	log    *logger.Logger
	rand   *rand.Rand
}

type KeyStroke struct {
	Char      rune
	Delay     time.Duration
	IsTypo    bool
	Backspace bool
}

func NewTypingController(cfg *config.TypingConfig) *TypingController {
	return &TypingController{
		config: cfg,
		log:    logger.WithComponent("typing"),
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (t *TypingController) GenerateKeystrokes(text string) []KeyStroke {
	if !t.config.Enabled {
		keystrokes := make([]KeyStroke, len(text))
		for i, char := range text {
			keystrokes[i] = KeyStroke{Char: char, Delay: 0}
		}
		return keystrokes
	}

	keystrokes := make([]KeyStroke, 0, len(text)*2)
	runes := []rune(text)

	for i, char := range runes {
		delay := t.calculateKeyDelay(char, i, runes)

		if t.rand.Float64() < t.config.ThinkPauseChance {
			if t.isWordBoundary(i, runes) {
				delay += time.Duration(500+t.rand.Intn(1500)) * time.Millisecond
			}
		}

		if t.rand.Float64() < t.config.TypoChance && !unicode.IsSpace(char) {
			typoChar := t.generateTypo(char)
			keystrokes = append(keystrokes, KeyStroke{
				Char:   typoChar,
				Delay:  delay,
				IsTypo: true,
			})

			keystrokes = append(keystrokes, KeyStroke{
				Char:      0,
				Delay:     t.config.CorrectionDelay,
				Backspace: true,
			})

			keystrokes = append(keystrokes, KeyStroke{
				Char:  char,
				Delay: t.calculateKeyDelay(char, i, runes),
			})
		} else {
			keystrokes = append(keystrokes, KeyStroke{
				Char:  char,
				Delay: delay,
			})
		}
	}

	return keystrokes
}

func (t *TypingController) calculateKeyDelay(char rune, position int, text []rune) time.Duration {
	base := t.config.MinKeyDelay +
		time.Duration(t.rand.Int63n(int64(t.config.MaxKeyDelay-t.config.MinKeyDelay)))

	if unicode.IsSpace(char) {
		base = time.Duration(float64(base) * 0.7)
	}

	if unicode.IsUpper(char) {
		base = time.Duration(float64(base) * 1.3)
	}

	if strings.ContainsRune("@#$%^&*()_+{}|:<>?", char) {
		base = time.Duration(float64(base) * 1.5)
	}

	if position > 0 && t.areAdjacent(text[position-1], char) {
		base = time.Duration(float64(base) * 0.85)
	}

	variation := float64(base) * 0.2 * (t.rand.Float64()*2 - 1)
	return base + time.Duration(variation)
}

func (t *TypingController) areAdjacent(prev, curr rune) bool {
	keyboardRows := []string{
		"qwertyuiop",
		"asdfghjkl",
		"zxcvbnm",
	}

	prevLower := unicode.ToLower(prev)
	currLower := unicode.ToLower(curr)

	for _, row := range keyboardRows {
		prevIdx := strings.IndexRune(row, prevLower)
		currIdx := strings.IndexRune(row, currLower)

		if prevIdx >= 0 && currIdx >= 0 && abs(prevIdx-currIdx) == 1 {
			return true
		}
	}

	return false
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (t *TypingController) isWordBoundary(position int, text []rune) bool {
	if position == 0 {
		return false
	}

	prevIsSpace := unicode.IsSpace(text[position-1])
	currIsSpace := unicode.IsSpace(text[position])

	return !prevIsSpace && currIsSpace
}

func (t *TypingController) generateTypo(char rune) rune {
	keyboardLayout := map[rune][]rune{
		'q': {'w', 'a'},
		'w': {'q', 'e', 's', 'a'},
		'e': {'w', 'r', 'd', 's'},
		'r': {'e', 't', 'f', 'd'},
		't': {'r', 'y', 'g', 'f'},
		'y': {'t', 'u', 'h', 'g'},
		'u': {'y', 'i', 'j', 'h'},
		'i': {'u', 'o', 'k', 'j'},
		'o': {'i', 'p', 'l', 'k'},
		'p': {'o', 'l'},
		'a': {'q', 'w', 's', 'z'},
		's': {'a', 'w', 'e', 'd', 'x', 'z'},
		'd': {'s', 'e', 'r', 'f', 'c', 'x'},
		'f': {'d', 'r', 't', 'g', 'v', 'c'},
		'g': {'f', 't', 'y', 'h', 'b', 'v'},
		'h': {'g', 'y', 'u', 'j', 'n', 'b'},
		'j': {'h', 'u', 'i', 'k', 'm', 'n'},
		'k': {'j', 'i', 'o', 'l', 'm'},
		'l': {'k', 'o', 'p'},
		'z': {'a', 's', 'x'},
		'x': {'z', 's', 'd', 'c'},
		'c': {'x', 'd', 'f', 'v'},
		'v': {'c', 'f', 'g', 'b'},
		'b': {'v', 'g', 'h', 'n'},
		'n': {'b', 'h', 'j', 'm'},
		'm': {'n', 'j', 'k'},
	}

	lowerChar := unicode.ToLower(char)
	if adjacent, ok := keyboardLayout[lowerChar]; ok {
		typo := adjacent[t.rand.Intn(len(adjacent))]
		if unicode.IsUpper(char) {
			return unicode.ToUpper(typo)
		}
		return typo
	}

	return char
}

func (t *TypingController) ExecuteTyping(ctx context.Context, typeFn func(char rune) error, backspaceFn func() error, text string) error {
	typingStart := time.Now()
	keystrokes := t.GenerateKeystrokes(text)

	for _, ks := range keystrokes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if ks.Delay > 0 {
			time.Sleep(ks.Delay)
		}

		if ks.Backspace {
			if err := backspaceFn(); err != nil {
				return err
			}
			if t.config.Enabled {
				t.log.Debug("Backspace pressed")
			}
		} else {
			if err := typeFn(ks.Char); err != nil {
				return err
			}
			if t.config.Enabled {
				t.log.Debug("Typed char: %c", ks.Char)
			}
		}
	}

	if t.config.Enabled {
		t.log.Info("Typing completed in %v", time.Since(typingStart))
	}
	return nil
}

func (t *TypingController) TypingDuration(text string) time.Duration {
	keystrokes := t.GenerateKeystrokes(text)
	total := time.Duration(0)
	for _, ks := range keystrokes {
		total += ks.Delay
	}
	return total
}
