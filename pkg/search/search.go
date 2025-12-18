package search

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"

	"github.com/linkedin-automation/pkg/browser"
	"github.com/linkedin-automation/pkg/config"
	"github.com/linkedin-automation/pkg/logger"
	"github.com/linkedin-automation/pkg/stealth"
	"github.com/linkedin-automation/pkg/storage"
)

const (
	linkedInSearchURL = "https://www.linkedin.com/search/results/people/"
)

type Searcher struct {
	config  *config.SearchConfig
	browser *browser.Browser
	storage *storage.Storage
	timing  *stealth.TimingController
	scroll  *stealth.ScrollController
	log     *logger.Logger
}

type SearchResult struct {
	ProfileURL string
	Name       string
	Title      string
	Company    string
	Location   string
	Degree     string
}

type SearchQuery struct {
	Keywords string
	JobTitle string
	Company  string
	Location string
	Industry string
	Page     int
}

func New(cfg *config.SearchConfig, b *browser.Browser, s *storage.Storage, timing *stealth.TimingController, scroll *stealth.ScrollController) *Searcher {
	return &Searcher{
		config:  cfg,
		browser: b,
		storage: s,
		timing:  timing,
		scroll:  scroll,
		log:     logger.WithComponent("search"),
	}
}

func (s *Searcher) Search(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	searchURL := s.buildSearchURL(query)
	s.log.Info("Searching: %s", searchURL)

	if err := s.browser.Navigate(ctx, searchURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to search: %w", err)
	}

	if err := s.timing.SleepPageLoad(ctx); err != nil {
		return nil, err
	}

	if err := s.scrollToLoadResults(ctx); err != nil {
		s.log.Warn("Error scrolling: %v", err)
	}

	results, err := s.extractResults(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract results: %w", err)
	}

	uniqueResults := s.filterDuplicates(results)

	for _, result := range uniqueResults {
		profile := storage.Profile{
			URL:              result.ProfileURL,
			Name:             result.Name,
			Title:            result.Title,
			Company:          result.Company,
			Location:         result.Location,
			ConnectionDegree: result.Degree,
			FoundAt:          time.Now(),
			Source:           "search",
			Processed:        false,
		}
		if err := s.storage.AddProfile(profile); err != nil {
			s.log.Warn("Failed to save profile: %v", err)
		}
	}

	s.log.Info("Found %d unique profiles", len(uniqueResults))
	return uniqueResults, nil
}

func (s *Searcher) buildSearchURL(query SearchQuery) string {
	params := url.Values{}

	if query.Keywords != "" {
		params.Set("keywords", query.Keywords)
	}

	if query.JobTitle != "" {
		params.Set("title", query.JobTitle)
	}

	if query.Company != "" {
		params.Set("company", query.Company)
	}

	if query.Location != "" {
		params.Set("geoUrn", query.Location)
	}

	if query.Page > 1 {
		params.Set("page", fmt.Sprintf("%d", query.Page))
	}

	return linkedInSearchURL + "?" + params.Encode()
}

func (s *Searcher) scrollToLoadResults(ctx context.Context) error {
	for i := 0; i < 3; i++ {
		if err := s.browser.Scroll(ctx, 500); err != nil {
			return err
		}

		if err := s.timing.SleepAction(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (s *Searcher) extractResults(ctx context.Context) ([]SearchResult, error) {
	resultSelectors := []string{
		".reusable-search__result-container",
		".entity-result",
		"li.reusable-search__result-container",
	}

	var err error

	for _, selector := range resultSelectors {
		elems, e := s.browser.Elements(ctx, selector)
		if e == nil && len(elems) > 0 {
			s.log.Debug("Found %d results with selector: %s", len(elems), selector)
			results := make([]SearchResult, 0, len(elems))

			for _, elem := range elems {
				result, extractErr := s.extractResultFromElement(ctx, elem)
				if extractErr != nil {
					s.log.Debug("Failed to extract result: %v", extractErr)
					continue
				}
				if result.ProfileURL != "" {
					results = append(results, *result)
				}
			}

			return results, nil
		}
		if e != nil {
			err = e
		}
	}

	if err != nil {
		return nil, err
	}

	return []SearchResult{}, nil
}

func (s *Searcher) extractResultFromElement(ctx context.Context, elem interface{}) (*SearchResult, error) {
	result := &SearchResult{}

	rodElem, ok := elem.(*rod.Element)
	if !ok {
		return nil, fmt.Errorf("invalid element type")
	}

	linkSelectors := []string{
		"span.entity-result__title-text a.app-aware-link",
		"span.entity-result__title-text a",
		"a.app-aware-link",
		".entity-result__title-text a",
		"a[href*='/in/']",
	}

	for _, selector := range linkSelectors {
		elements, err := rodElem.Elements(selector)
		if err != nil || len(elements) == 0 {
			continue
		}

		for _, el := range elements {
			href, err := el.Attribute("href")
			if err != nil || href == nil {
				continue
			}

			if strings.Contains(*href, "/in/") {
				result.ProfileURL = s.normalizeProfileURL(*href)

				text, err := el.Text()
				if err == nil {
					result.Name = strings.TrimSpace(text)
				}
				break
			}
		}

		if result.ProfileURL != "" {
			break
		}
	}

	titleSelectors := []string{
		".entity-result__primary-subtitle",
		".subline-level-1",
	}

	for _, selector := range titleSelectors {
		elements, err := rodElem.Elements(selector)
		if err != nil || len(elements) == 0 {
			continue
		}

		text, err := elements[0].Text()
		if err == nil {
			result.Title = strings.TrimSpace(text)
			break
		}
	}

	locationSelectors := []string{
		".entity-result__secondary-subtitle",
		".subline-level-2",
	}

	for _, selector := range locationSelectors {
		elements, err := rodElem.Elements(selector)
		if err != nil || len(elements) == 0 {
			continue
		}

		text, err := elements[0].Text()
		if err == nil {
			result.Location = strings.TrimSpace(text)
			break
		}
	}

	degreeSelectors := []string{
		".entity-result__badge-text",
		".member-insights-badge",
	}

	for _, selector := range degreeSelectors {
		elements, err := rodElem.Elements(selector)
		if err != nil || len(elements) == 0 {
			continue
		}

		text, err := elements[0].Text()
		if err == nil {
			result.Degree = strings.TrimSpace(text)
			break
		}
	}

	return result, nil
}

func (s *Searcher) normalizeProfileURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	re := regexp.MustCompile(`/in/([^/?]+)`)
	matches := re.FindStringSubmatch(parsed.Path)
	if len(matches) > 1 {
		return fmt.Sprintf("https://www.linkedin.com/in/%s/", matches[1])
	}

	return rawURL
}

func (s *Searcher) filterDuplicates(results []SearchResult) []SearchResult {
	seen := make(map[string]bool)
	unique := make([]SearchResult, 0, len(results))

	for _, result := range results {
		if result.ProfileURL == "" {
			continue
		}

		exists, err := s.storage.ProfileExists(result.ProfileURL)
		if err != nil {
			s.log.Warn("Error checking profile existence: %v", err)
		}

		if !seen[result.ProfileURL] && !exists {
			seen[result.ProfileURL] = true
			unique = append(unique, result)
		}
	}

	return unique
}

func (s *Searcher) SearchAll(ctx context.Context) ([]SearchResult, error) {
	var allResults []SearchResult

	for _, keyword := range s.config.Keywords {
		query := SearchQuery{Keywords: keyword}
		results, err := s.searchWithPagination(ctx, query)
		if err != nil {
			s.log.Warn("Search failed for keyword '%s': %v", keyword, err)
			continue
		}
		allResults = append(allResults, results...)

		if err := s.timing.SleepThink(ctx); err != nil {
			return allResults, err
		}
	}

	for _, title := range s.config.JobTitles {
		query := SearchQuery{JobTitle: title}
		results, err := s.searchWithPagination(ctx, query)
		if err != nil {
			s.log.Warn("Search failed for job title '%s': %v", title, err)
			continue
		}
		allResults = append(allResults, results...)

		if err := s.timing.SleepThink(ctx); err != nil {
			return allResults, err
		}
	}

	for _, company := range s.config.Companies {
		query := SearchQuery{Company: company}
		results, err := s.searchWithPagination(ctx, query)
		if err != nil {
			s.log.Warn("Search failed for company '%s': %v", company, err)
			continue
		}
		allResults = append(allResults, results...)

		if err := s.timing.SleepThink(ctx); err != nil {
			return allResults, err
		}
	}

	return s.filterDuplicates(allResults), nil
}

func (s *Searcher) searchWithPagination(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	var allResults []SearchResult

	for page := 1; page <= s.config.PagesPerSearch; page++ {
		query.Page = page

		results, err := s.Search(ctx, query)
		if err != nil {
			return allResults, err
		}

		if len(results) == 0 {
			break
		}

		allResults = append(allResults, results...)

		if len(allResults) >= s.config.MaxResults {
			allResults = allResults[:s.config.MaxResults]
			break
		}

		if page < s.config.PagesPerSearch {
			if err := s.timing.SleepThink(ctx); err != nil {
				return allResults, err
			}
		}
	}

	return allResults, nil
}

func (s *Searcher) HasNextPage(ctx context.Context) bool {
	nextButtonSelectors := []string{
		"button[aria-label='Next']",
		".artdeco-pagination__button--next",
	}

	for _, selector := range nextButtonSelectors {
		if s.browser.Exists(selector) {
			return true
		}
	}

	return false
}

func (s *Searcher) GoToNextPage(ctx context.Context) error {
	nextButtonSelectors := []string{
		"button[aria-label='Next']",
		".artdeco-pagination__button--next",
	}

	for _, selector := range nextButtonSelectors {
		if s.browser.Exists(selector) {
			if err := s.browser.Click(ctx, selector); err != nil {
				continue
			}

			if err := s.timing.SleepPageLoad(ctx); err != nil {
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("next page button not found")
}
