package search

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/linkedin-automation/pkg/browser"
	"github.com/linkedin-automation/pkg/logger"
	"github.com/linkedin-automation/pkg/stealth"
)

// VisualSearcher performs searches with visible mouse movements and interactions
type VisualSearcher struct {
	browser *browser.Browser
	timing  *stealth.TimingController
	scroll  *stealth.ScrollController
	mouse   *stealth.MouseController
	log     *logger.Logger
}

func NewVisualSearcher(b *browser.Browser, timing *stealth.TimingController, scroll *stealth.ScrollController, mouse *stealth.MouseController) *VisualSearcher {
	return &VisualSearcher{
		browser: b,
		timing:  timing,
		scroll:  scroll,
		mouse:   mouse,
		log:     logger.WithComponent("visual-search"),
	}
}

// CaptureScreenshot takes a screenshot for visual verification
func (v *VisualSearcher) CaptureScreenshot(name string) {
	page := v.browser.Page()
	filename := fmt.Sprintf("step_%d_%s.png", time.Now().Unix(), name)
	v.log.Info("📸 VISUAL VERIFICATION: Capturing screenshot: %s", filename)

	_ = page.MustWaitLoad()
	data, err := page.Screenshot(true, nil)
	if err != nil {
		v.log.Warn("Failed to take screenshot: %v", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		v.log.Warn("Failed to save screenshot: %v", err)
	}
}

// ensureCursor makes the cursor IMPOSSIBLE to miss
func (v *VisualSearcher) ensureCursor(page *rod.Page) {
	_, _ = page.Evaluate(rod.Eval(`() => { var old=document.getElementById('visible-cursor'); if(old) old.remove(); if(!document.getElementById('cursor-style')){ var s=document.createElement('style'); s.id='cursor-style'; s.innerHTML='@keyframes pulse{0%,100%{transform:translate(-50%,-50%) scale(1);box-shadow:0 0 20px red}50%{transform:translate(-50%,-50%) scale(1.3);box-shadow:0 0 40px red}}'; document.head.appendChild(s); } var c=document.createElement('div'); c.id='visible-cursor'; c.style.cssText='position:fixed!important;width:30px!important;height:30px!important;background:red!important;border-radius:50%!important;pointer-events:none!important;z-index:2147483647!important;border:3px solid white!important;display:block!important;visibility:visible!important;opacity:1!important;animation:pulse 1s ease-in-out infinite!important'; c.style.left=(window.innerWidth/2)+'px'; c.style.top=(window.innerHeight/2)+'px'; document.body.appendChild(c); if(window.cursorKeeper) clearInterval(window.cursorKeeper); window.cursorKeeper=setInterval(function(){ var x=document.getElementById('visible-cursor'); if(!x){ x=c.cloneNode(true); x.id='visible-cursor'; document.body.appendChild(x); } x.style.display='block'; x.style.visibility='visible'; x.style.opacity='1';  },50); window.moveCursor=function(x,y){ var d=document.getElementById('visible-cursor'); if(d){ d.style.left=x+'px'; d.style.top=y+'px';  }}; window.flashCursor=function(){ var d=document.getElementById('visible-cursor'); if(d){ d.style.background='yellow'; d.style.transform='translate(-50%,-50%) scale(2)'; setTimeout(function(){ d.style.background='red'; d.style.transform='translate(-50%,-50()) scale(1)'; },300);  }}; }`).ByUser())
}

// getCursorPos retrieves cursor position
func (v *VisualSearcher) getCursorPos(page *rod.Page) (float64, float64) {
	res, _ := page.Eval(`() => {
		var c = document.getElementById('visible-cursor');
		if (c) {
			var rect = c.getBoundingClientRect();
			return { x: rect.left, y: rect.top };
		}
		return { x: window.innerWidth/2, y: window.innerHeight/2 };
	}`)

	var pos struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	if res != nil {
		res.Value.Unmarshal(&pos)
	}
	return pos.X, pos.Y
}

// SearchWithUI performs visible search
func (v *VisualSearcher) SearchWithUI(ctx context.Context, searchTerm string) error {
	v.log.Info("Starting visual search for: %s", searchTerm)
	page := v.browser.Page()
	v.ensureCursor(page)
	time.Sleep(1 * time.Second)

	searchBoxSelectors := []string{
		"input.search-global-typeahead__input",
		"input[placeholder*='Search']",
		".search-global-typeahead__input",
	}

	var searchBoxClicked bool
	for _, selector := range searchBoxSelectors {
		v.ensureCursor(page)
		if v.browser.Exists(selector) {
			v.log.Info("Found search box: %s", selector)
			v.browser.Click(ctx, selector)
			searchBoxClicked = true
			break
		}
	}

	if !searchBoxClicked {
		return fmt.Errorf("search box not found")
	}

	v.CaptureScreenshot("search_box_clicked")
	time.Sleep(1 * time.Second)

	v.log.Info("Typing search term...")
	for _, selector := range searchBoxSelectors {
		if v.browser.Exists(selector) {
			el := page.MustElement(selector)
			el.Click(proto.InputMouseButtonLeft, 1)
			time.Sleep(200 * time.Millisecond)

			page.MustEval(`(sel) => { var el = document.querySelector(sel); if (el) { el.value = ""; el.dispatchEvent(new Event('input', { bubbles: true })); } }`, selector)

			for i := 0; i < 5; i++ {
				page.Keyboard.Type(input.Backspace)
			}

			v.browser.Type(ctx, selector, searchTerm)
			break
		}
	}

	time.Sleep(1500 * time.Millisecond)
	v.log.Info("Submitting search...")
	v.ensureCursor(page)

	for _, selector := range searchBoxSelectors {
		if v.browser.Exists(selector) {
			page.MustElement(selector).MustClick()
			page.MustElement(selector).MustType(input.Enter)
			break
		}
	}

	time.Sleep(4 * time.Second)
	v.ensureCursor(page)

	v.log.Info("Looking for People tab...")
	page = v.browser.Page()
	v.ensureCursor(page)
	time.Sleep(2 * time.Second)

	peopleSelectors := []string{
		"button[aria-label='People']",
		"button[aria-label*='People']",
		".search-reusables__primary-filter button",
	}

	var peopleBtn *rod.Element
	for _, sel := range peopleSelectors {
		elements, err := page.Elements(sel)
		if err == nil {
			for _, el := range elements {
				text, _ := el.Text()
				if strings.Contains(strings.ToLower(text), "people") {
					peopleBtn = el
					break
				}
			}
		}
		if peopleBtn != nil {
			break
		}
	}

	if peopleBtn != nil {
		peopleBtn.MustScrollIntoView()
		time.Sleep(500 * time.Millisecond)
		v.ensureCursor(page)

		shape, _ := peopleBtn.Shape()
		box := shape.Box()
		centerX := box.X + box.Width/2
		centerY := box.Y + box.Height/2

		// Move cursor visibly to People button
		path := v.mouse.GeneratePath(stealth.Point{X: 100, Y: 100}, stealth.Point{X: centerX, Y: centerY})
		for _, point := range path {
			page.Mouse.MustMoveTo(point.X, point.Y)
			page.MustEval(`(x, y) => { if(window.moveCursor) window.moveCursor(x, y); }`, point.X, point.Y)
			time.Sleep(8 * time.Millisecond)
		}

		time.Sleep(300 * time.Millisecond)
		page.MustEval(`() => { if(window.flashCursor) window.flashCursor(); }`)
		page.Mouse.MustClick(proto.InputMouseButtonLeft)

		v.log.Info("✓ People clicked! Keeping pointer visible...")

		// CRITICAL: Keep cursor visible during reload (snappier transition)
		for i := 0; i < 5; i++ {
			v.ensureCursor(page)
			time.Sleep(200 * time.Millisecond)
		}
	}

	// Faster scrolling
	v.log.Info("Scrolling...")
	for i := 0; i < 4; i++ {
		v.ensureCursor(page)
		v.browser.Scroll(ctx, 800)
		time.Sleep(300 * time.Millisecond)
	}

	v.log.Info("Scrolling back...")
	for i := 0; i < 3; i++ {
		v.ensureCursor(page)
		v.browser.Scroll(ctx, -700)
		time.Sleep(200 * time.Millisecond)
	}

	v.ensureCursor(page)
	time.Sleep(200 * time.Millisecond)
	return nil
}

// ClickProfile with ULTRA-VISIBLE cursor movements
func (v *VisualSearcher) ClickProfile(ctx context.Context, profileKeywords []string) error {
	v.log.Info("Clicking profile: %v", profileKeywords)
	page := v.browser.Page()
	v.ensureCursor(page)
	time.Sleep(200 * time.Millisecond)

	profiles, err := page.Elements("a[href*='/in/']")
	if err != nil {
		return fmt.Errorf("failed to find profiles: %w", err)
	}

	type ScoredProfile struct {
		Element *rod.Element
		Score   int
		Name    string
	}

	var scoredProfiles []ScoredProfile
	v.log.Info("Evaluating %d candidates...", len(profiles))

	for _, p := range profiles {
		v.ensureCursor(page)
		text, _ := p.Text()
		hrefAttr, _ := p.Attribute("href")
		if text == "" || len(text) < 3 || hrefAttr == nil {
			continue
		}

		textLower := strings.ToLower(text)
		if strings.Contains(textLower, "premium") || strings.Contains(textLower, "connection") {
			continue
		}

		score := 0
		contextText := ""

		ancestor, _ := p.Parent()
		for j := 0; j < 6 && ancestor != nil; j++ {
			attr, _ := ancestor.Attribute("class")
			if attr != nil && (strings.Contains(*attr, "entity-result") || strings.Contains(*attr, "reusable-search__result-container")) {
				ct, _ := ancestor.Text()
				contextText = strings.ToLower(ct)
				break
			}
			ancestor, _ = ancestor.Parent()
		}

		for _, kw := range profileKeywords {
			kwLower := strings.ToLower(kw)
			if strings.Contains(textLower, kwLower) {
				score += 50
			}
			if strings.Contains(contextText, kwLower) {
				score += 20
			}
		}

		fullText := textLower + " " + contextText
		if strings.Contains(fullText, "united states") || strings.Contains(fullText, "usa") {
			score -= 600
		} else if strings.Contains(fullText, "india") || strings.Contains(fullText, "bangalore") {
			score += 250
		}

		if strings.Contains(fullText, "pes") {
			score += 150
		}

		// EXACT FULL NAME MATCH BOOST: If ALL name parts appear, huge bonus
		nameKeywords := []string{}
		for _, kw := range profileKeywords {
			kwLower := strings.ToLower(kw)
			// Skip location/university keywords
			if kwLower != "pes" && kwLower != "university" && kwLower != "india" && kwLower != "karnataka" {
				nameKeywords = append(nameKeywords, kwLower)
			}
		}

		if len(nameKeywords) >= 2 {
			allNamePartsMatch := true
			for _, namePart := range nameKeywords {
				if !strings.Contains(textLower, namePart) && !strings.Contains(contextText, namePart) {
					allNamePartsMatch = false
					break
				}
			}
			if allNamePartsMatch {
				score += 500 // Massive boost for full name match
			}
		}

		hasMatch := false
		for _, kw := range profileKeywords {
			if strings.Contains(textLower, strings.ToLower(kw)) || strings.Contains(contextText, strings.ToLower(kw)) {
				hasMatch = true
				break
			}
		}

		if hasMatch {
			scoredProfiles = append(scoredProfiles, ScoredProfile{
				Element: p,
				Score:   score,
				Name:    text,
			})
		}
	}

	if len(scoredProfiles) == 0 {
		return fmt.Errorf("no matches")
	}

	sort.Slice(scoredProfiles, func(i, j int) bool {
		return scoredProfiles[i].Score > scoredProfiles[j].Score
	})

	best := scoredProfiles[0]
	v.log.Info("✓ Selected: %s (Score: %d)", best.Name, best.Score)

	v.ensureCursor(page)
	best.Element.MustScrollIntoView()
	time.Sleep(200 * time.Millisecond)

	shape, _ := best.Element.Shape()
	box := shape.Box()
	targetX, targetY := box.X+box.Width/2, box.Y+box.Height/2

	// ULTRA-VISIBLE movement from far away (BOMBSHELL SPEED)
	v.log.Info("Moving cursor to profile...")
	path := v.mouse.GeneratePath(stealth.Point{X: 50, Y: 50}, stealth.Point{X: targetX, Y: targetY})
	for _, point := range path {
		page.Mouse.MustMoveTo(point.X, point.Y)
		page.MustEval(`(x, y) => { if(window.moveCursor) window.moveCursor(x, y); }`, point.X, point.Y)
		time.Sleep(5 * time.Millisecond)
	}

	time.Sleep(50 * time.Millisecond)
	page.MustEval(`() => { if(window.flashCursor) window.flashCursor(); }`)
	time.Sleep(100 * time.Millisecond)
	page.Mouse.MustClick(proto.InputMouseButtonLeft)

	v.log.Info("✓ Profile clicked!")
	time.Sleep(6 * time.Second)
	v.CaptureScreenshot("profile_opened")
	v.ensureCursor(page)
	return nil
}

func (v *VisualSearcher) ScrollProfile(ctx context.Context) error {
	for i := 0; i < 4; i++ {
		v.ensureCursor(v.browser.Page())
		v.browser.Scroll(ctx, 450)
		time.Sleep(1 * time.Second)
	}
	v.browser.Scroll(ctx, -200)
	return nil
}

func (v *VisualSearcher) ApplyFilters(ctx context.Context, filterType, filterValue string) error {
	v.log.Info("Filter placeholder: %s=%s", filterType, filterValue)
	return nil
}

func (v *VisualSearcher) ExecuteTyping(ctx context.Context, selector, text string) error {
	v.ensureCursor(v.browser.Page())
	el := v.browser.Page().MustElement(selector)
	el.MustClick()
	for _, char := range text {
		el.MustInput(string(char))
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}
