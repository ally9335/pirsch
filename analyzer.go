package pirsch

import (
	"errors"
	"fmt"
	"time"
)

const (
	byAttributeQuery = `SELECT "%s", count(DISTINCT fingerprint) visitors, visitors / (
			SELECT sum(s) FROM (
				SELECT count(DISTINCT fingerprint) s
				FROM hit
				WHERE %s
				GROUP BY "%s"
			)
		) relative_visitors
		FROM hit
		WHERE %s
		GROUP BY "%s"
		ORDER BY visitors DESC, "%s" ASC
		%s`
)

var (
	// ErrNoPeriodOrDay is returned in case no period or day was specified to calculate the growth rate.
	ErrNoPeriodOrDay = errors.New("no period or day specified")
)

type growthStats struct {
	Visitors int `json:"visitors"`
	Views    int `json:"views"`
	Sessions int `json:"sessions"`
	Bounces  int `json:"bounces"`
}

// Analyzer provides an interface to analyze statistics.
type Analyzer struct {
	store Store
}

// NewAnalyzer returns a new Analyzer for given Store.
func NewAnalyzer(store Store) *Analyzer {
	return &Analyzer{
		store,
	}
}

// ActiveVisitors returns the active visitors per path and the total number of active visitors for given duration.
// Use time.Minute*5 for example to get the active visitors for the past 5 minutes.
func (analyzer *Analyzer) ActiveVisitors(filter *Filter, duration time.Duration) ([]ActiveVisitorStats, int, error) {
	filter = analyzer.getFilter(filter)
	filter.Start = time.Now().In(filter.Timezone).Add(-duration)
	args, filterQuery := filter.query()
	query := fmt.Sprintf(`SELECT path, count(DISTINCT fingerprint) visitors
		FROM hit
		WHERE %s
		GROUP BY path
		ORDER BY visitors DESC, path ASC`, filterQuery)
	var stats []ActiveVisitorStats

	if err := analyzer.store.Select(&stats, query, args...); err != nil {
		return nil, 0, err
	}

	query = fmt.Sprintf(`SELECT count(DISTINCT fingerprint) visitors FROM hit WHERE %s`, filterQuery)
	count, err := analyzer.store.Count(query, args...)

	if err != nil {
		return nil, 0, err
	}

	return stats, count, nil
}

// Visitors returns the visitor count, session count, bounce rate, views, and average session duration grouped by day.
func (analyzer *Analyzer) Visitors(filter *Filter) ([]VisitorStats, error) {
	filter = analyzer.getFilter(filter)
	args, filterQuery := filter.query()
	withFillArgs, withFillQuery := filter.withFill()
	args = append(args, withFillArgs...)
	timezone := filter.Timezone.String()
	query := fmt.Sprintf(`SELECT day,
		sum(visitors) visitors,
		sum(sessions) sessions,
		sum(views) views,
		countIf(bounce = 1) bounces,
		bounces / IF(visitors = 0, 1, visitors) bounce_rate
		FROM (
			SELECT toDate(time, '%s') day,
			count(DISTINCT fingerprint) visitors,
			count(DISTINCT(fingerprint, session)) sessions,
			count(*) views,
			length(groupArray(path)) = 1 bounce
			FROM hit
			WHERE %s
			GROUP BY toDate(time, '%s'), fingerprint
		)
		GROUP BY day
		ORDER BY day ASC %s, visitors DESC`, timezone, filterQuery, timezone, withFillQuery)
	var stats []VisitorStats

	if err := analyzer.store.Select(&stats, query, args...); err != nil {
		return nil, err
	}

	return stats, nil
}

// Growth returns the growth rate for visitor count, session count, bounces, views, and average session duration or average time on page (if path is set).
// The growth rate is relative to the previous time range or day.
// The period or day for the filter must be set, else an error is returned.
func (analyzer *Analyzer) Growth(filter *Filter) (*Growth, error) {
	filter = analyzer.getFilter(filter)

	if filter.Day.IsZero() && (filter.From.IsZero() || filter.To.IsZero()) {
		return nil, ErrNoPeriodOrDay
	}

	args, filterQuery := filter.query()
	query := fmt.Sprintf(`SELECT sum(visitors) visitors,
		sum(sessions) sessions,
		sum(views) views,
		countIf(bounce = 1) bounces
		FROM (
			SELECT count(DISTINCT fingerprint) visitors,
			count(DISTINCT(fingerprint, session)) sessions,
			count(*) views,
			length(groupArray(path)) = 1 bounce
			FROM hit
			WHERE %s
			GROUP BY toDate(time, '%s'), fingerprint
		)`, filterQuery, filter.Timezone.String())
	current := new(growthStats)

	if err := analyzer.store.Get(current, query, args...); err != nil {
		return nil, err
	}

	var currentTimeSpent int
	var err error

	if filter.Path == "" {
		currentTimeSpent, err = analyzer.TotalSessionDuration(filter)
	} else {
		currentTimeSpent, err = analyzer.TotalTimeOnPage(filter)
	}

	if err != nil {
		return nil, err
	}

	if filter.Day.IsZero() {
		days := filter.To.Sub(filter.From)
		filter.To = filter.From.Add(-time.Hour * 24)
		filter.From = filter.To.Add(-days)
	} else {
		filter.Day = filter.Day.Add(-time.Hour * 24)
	}

	args, _ = filter.query()
	previous := new(growthStats)

	if err := analyzer.store.Get(previous, query, args...); err != nil {
		return nil, err
	}

	var previousTimeSpent int

	if filter.Path == "" {
		previousTimeSpent, err = analyzer.TotalSessionDuration(filter)
	} else {
		previousTimeSpent, err = analyzer.TotalTimeOnPage(filter)
	}

	if err != nil {
		return nil, err
	}

	return &Growth{
		VisitorsGrowth:  analyzer.calculateGrowth(current.Visitors, previous.Visitors),
		ViewsGrowth:     analyzer.calculateGrowth(current.Views, previous.Views),
		SessionsGrowth:  analyzer.calculateGrowth(current.Sessions, previous.Sessions),
		BouncesGrowth:   analyzer.calculateGrowth(current.Bounces, previous.Bounces),
		TimeSpentGrowth: analyzer.calculateGrowth(currentTimeSpent, previousTimeSpent),
	}, nil
}

// VisitorHours returns the visitor count grouped by time of day.
func (analyzer *Analyzer) VisitorHours(filter *Filter) ([]VisitorHourStats, error) {
	args, filterQuery := analyzer.getFilter(filter).query()
	query := fmt.Sprintf(`SELECT toHour(time) hour, count(DISTINCT fingerprint) visitors
		FROM hit
		WHERE %s
		GROUP BY hour
		ORDER BY hour WITH FILL FROM 0 TO 24`, filterQuery)
	var stats []VisitorHourStats

	if err := analyzer.store.Select(&stats, query, args...); err != nil {
		return nil, err
	}

	return stats, nil
}

// Pages returns the visitor count, session count, bounce rate, views, and average time on page grouped by path.
func (analyzer *Analyzer) Pages(filter *Filter) ([]PageStats, error) {
	filter = analyzer.getFilter(filter)
	filterArgs, filterQuery := filter.query()
	query := fmt.Sprintf(`SELECT path,
		count(DISTINCT fingerprint) visitors,
		visitors / (
			SELECT sum(s) FROM (
				SELECT count(DISTINCT fingerprint) s
				FROM hit
				WHERE %s
				GROUP BY path, fingerprint
			)
		) relative_visitors,
		sum(sessions) sessions,
		sum(views) views,
		views / (
			SELECT sum(s) FROM (
				SELECT count(*) s
				FROM hit
				WHERE %s
				GROUP BY path, fingerprint
			)
		) relative_views,
		countIf(bounce = 1) bounces,
		bounces / IF(visitors = 0, 1, visitors) bounce_rate
		FROM (
			SELECT path,
			fingerprint,
			count(DISTINCT(fingerprint, session)) sessions,
			count(*) views,
			length(groupArray(path)) = 1 bounce
			FROM hit
			WHERE %s
			GROUP BY path, fingerprint
		)
		GROUP BY path
		ORDER BY visitors DESC, path ASC
		%s`, filterQuery, filterQuery, filterQuery, filter.withLimit())
	args := make([]interface{}, 0, len(filterArgs)*3)
	args = append(args, filterArgs...)
	args = append(args, filterArgs...)
	args = append(args, filterArgs...)
	var stats []PageStats

	if err := analyzer.store.Select(&stats, query, args...); err != nil {
		return nil, err
	}

	if filter.IncludeAvgTimeOnPage {
		timeOnPage, err := analyzer.AvgTimeOnPages(filter)

		if err != nil {
			return nil, err
		}

		for i := range stats {
			for j := range timeOnPage {
				if stats[i].Path == timeOnPage[j].Path {
					stats[i].AverageTimeSpentSeconds = timeOnPage[j].AverageTimeSpentSeconds
					break
				}
			}
		}
	}

	return stats, nil
}

// Referrer returns the visitor count and bounce rate grouped by referrer.
func (analyzer *Analyzer) Referrer(filter *Filter) ([]ReferrerStats, error) {
	filter = analyzer.getFilter(filter)
	args, filterQuery := filter.query()
	query := fmt.Sprintf(`SELECT referrer,
		referrer_name,
		referrer_icon,
		count(DISTINCT fingerprint) visitors,
		visitors / (
			SELECT sum(s) FROM (
				SELECT count(DISTINCT fingerprint) s
				FROM hit
				WHERE %s
				GROUP BY fingerprint, referrer, referrer_name, referrer_icon
			)
		) relative_visitors,
		countIf(bounce = 1) bounces,
		bounces / IF(visitors = 0, 1, visitors) bounce_rate
		FROM (
			SELECT fingerprint,
			referrer,
			referrer_name,
			referrer_icon,
			length(groupArray(path)) = 1 bounce
			FROM hit
			WHERE %s
			GROUP BY fingerprint, referrer, referrer_name, referrer_icon
		)
		GROUP BY referrer, referrer_name, referrer_icon
		ORDER BY visitors DESC
		%s`, filterQuery, filterQuery, filter.withLimit())
	args = append(args, args...)
	var stats []ReferrerStats

	if err := analyzer.store.Select(&stats, query, args...); err != nil {
		return nil, err
	}

	return stats, nil
}

// Platform returns the visitor count grouped by platform.
func (analyzer *Analyzer) Platform(filter *Filter) (*PlatformStats, error) {
	filterArgs, filterQuery := analyzer.getFilter(filter).query()
	query := fmt.Sprintf(`SELECT (
			SELECT count(DISTINCT fingerprint)
			FROM "hit"
			WHERE %s
			AND desktop = 1
			AND mobile = 0
		) AS "platform_desktop",
		(
			SELECT count(DISTINCT fingerprint)
			FROM "hit"
			WHERE %s
			AND desktop = 0
			AND mobile = 1
		) AS "platform_mobile",
		(
			SELECT count(DISTINCT fingerprint)
			FROM "hit"
			WHERE %s
			AND desktop = 0
			AND mobile = 0
		) AS "platform_unknown",
		"platform_desktop" / IF("platform_desktop" + "platform_mobile" + "platform_unknown" = 0, 1, "platform_desktop" + "platform_mobile" + "platform_unknown") AS relative_platform_desktop,
		"platform_mobile" / IF("platform_desktop" + "platform_mobile" + "platform_unknown" = 0, 1, "platform_desktop" + "platform_mobile" + "platform_unknown") AS relative_platform_mobile,
		"platform_unknown" / IF("platform_desktop" + "platform_mobile" + "platform_unknown" = 0, 1, "platform_desktop" + "platform_mobile" + "platform_unknown") AS relative_platform_unknown`, filterQuery, filterQuery, filterQuery)
	args := make([]interface{}, 0, len(filterArgs)*3)
	args = append(args, filterArgs...)
	args = append(args, filterArgs...)
	args = append(args, filterArgs...)
	stats := new(PlatformStats)

	if err := analyzer.store.Get(stats, query, args...); err != nil {
		return nil, err
	}

	return stats, nil
}

// Languages returns the visitor count grouped by language.
func (analyzer *Analyzer) Languages(filter *Filter) ([]LanguageStats, error) {
	var stats []LanguageStats

	if err := analyzer.selectByAttribute(&stats, filter, "language"); err != nil {
		return nil, err
	}

	return stats, nil
}

// Countries returns the visitor count grouped by country.
func (analyzer *Analyzer) Countries(filter *Filter) ([]CountryStats, error) {
	var stats []CountryStats

	if err := analyzer.selectByAttribute(&stats, filter, "country_code"); err != nil {
		return nil, err
	}

	return stats, nil
}

// Browser returns the visitor count grouped by browser.
func (analyzer *Analyzer) Browser(filter *Filter) ([]BrowserStats, error) {
	var stats []BrowserStats

	if err := analyzer.selectByAttribute(&stats, filter, "browser"); err != nil {
		return nil, err
	}

	return stats, nil
}

// OS returns the visitor count grouped by operating system.
func (analyzer *Analyzer) OS(filter *Filter) ([]OSStats, error) {
	var stats []OSStats

	if err := analyzer.selectByAttribute(&stats, filter, "os"); err != nil {
		return nil, err
	}

	return stats, nil
}

// ScreenClass returns the visitor count grouped by screen class.
func (analyzer *Analyzer) ScreenClass(filter *Filter) ([]ScreenClassStats, error) {
	var stats []ScreenClassStats

	if err := analyzer.selectByAttribute(&stats, filter, "screen_class"); err != nil {
		return nil, err
	}

	return stats, nil
}

// UTMSource returns the visitor count grouped by utm source.
func (analyzer *Analyzer) UTMSource(filter *Filter) ([]UTMSourceStats, error) {
	var stats []UTMSourceStats

	if err := analyzer.selectByAttribute(&stats, filter, "utm_source"); err != nil {
		return nil, err
	}

	return stats, nil
}

// UTMMedium returns the visitor count grouped by utm medium.
func (analyzer *Analyzer) UTMMedium(filter *Filter) ([]UTMMediumStats, error) {
	var stats []UTMMediumStats

	if err := analyzer.selectByAttribute(&stats, filter, "utm_medium"); err != nil {
		return nil, err
	}

	return stats, nil
}

// UTMCampaign returns the visitor count grouped by utm source.
func (analyzer *Analyzer) UTMCampaign(filter *Filter) ([]UTMCampaignStats, error) {
	var stats []UTMCampaignStats

	if err := analyzer.selectByAttribute(&stats, filter, "utm_campaign"); err != nil {
		return nil, err
	}

	return stats, nil
}

// UTMContent returns the visitor count grouped by utm source.
func (analyzer *Analyzer) UTMContent(filter *Filter) ([]UTMContentStats, error) {
	var stats []UTMContentStats

	if err := analyzer.selectByAttribute(&stats, filter, "utm_content"); err != nil {
		return nil, err
	}

	return stats, nil
}

// UTMTerm returns the visitor count grouped by utm source.
func (analyzer *Analyzer) UTMTerm(filter *Filter) ([]UTMTermStats, error) {
	var stats []UTMTermStats

	if err := analyzer.selectByAttribute(&stats, filter, "utm_term"); err != nil {
		return nil, err
	}

	return stats, nil
}

// AvgSessionDuration returns the average session duration grouped by day.
func (analyzer *Analyzer) AvgSessionDuration(filter *Filter) ([]TimeSpentStats, error) {
	filter = analyzer.getFilter(filter)
	args, filterQuery := filter.query()
	withFillArgs, withFillQuery := filter.withFill()
	args = append(args, withFillArgs...)
	query := fmt.Sprintf(`SELECT day, toUInt64(avg(duration)) average_time_spent_seconds
			FROM (
				SELECT toDate(time, '%s') day, max(time)-min(time) duration
				FROM hit
				WHERE %s
				AND session IS NOT NULL
				GROUP BY day, fingerprint, session
			)
		WHERE duration != 0
		GROUP BY day
		ORDER BY day %s`, filter.Timezone.String(), filterQuery, withFillQuery)
	var stats []TimeSpentStats

	if err := analyzer.store.Select(&stats, query, args...); err != nil {
		return nil, err
	}

	return stats, nil
}

// TotalSessionDuration returns the total session duration in seconds.
func (analyzer *Analyzer) TotalSessionDuration(filter *Filter) (int, error) {
	filter = analyzer.getFilter(filter)
	args, filterQuery := filter.query()
	query := fmt.Sprintf(`SELECT sum(duration) average_time_spent_seconds
		FROM (
			SELECT toDate(time, '%s') day, max(time)-min(time) duration
			FROM hit
			WHERE %s
			AND session IS NOT NULL
			GROUP BY day, fingerprint, session
		)`, filter.Timezone.String(), filterQuery)
	stats := new(struct {
		AverageTimeSpentSeconds int `db:"average_time_spent_seconds" json:"average_time_spent_seconds"`
	})

	if err := analyzer.store.Get(stats, query, args...); err != nil {
		return 0, err
	}

	return stats.AverageTimeSpentSeconds, nil
}

// AvgTimeOnPages returns the average time on page grouped by path.
func (analyzer *Analyzer) AvgTimeOnPages(filter *Filter) ([]TimeSpentStats, error) {
	filter = analyzer.getFilter(filter)
	timeArgs, timeQuery := filter.queryTime()
	fieldArgs, fieldQuery := filter.queryFields()

	if len(fieldArgs) > 0 {
		fieldQuery = "AND " + fieldQuery
	}

	query := fmt.Sprintf(`SELECT path, toUInt64(avg(time_on_page)) average_time_spent_seconds
		FROM (
			SELECT path, neighbor(previous_time_on_page_seconds, 1, 0) time_on_page
			FROM (
				SELECT *
				FROM hit
				WHERE %s
				ORDER BY fingerprint, time
			)
			WHERE time_on_page > 0
			%s
		)
		GROUP BY path
		ORDER BY path`, timeQuery, fieldQuery)
	timeArgs = append(timeArgs, fieldArgs...)
	var stats []TimeSpentStats

	if err := analyzer.store.Select(&stats, query, timeArgs...); err != nil {
		return nil, err
	}

	return stats, nil
}

// AvgTimeOnPage returns the average time on page grouped by day.
func (analyzer *Analyzer) AvgTimeOnPage(filter *Filter) ([]TimeSpentStats, error) {
	filter = analyzer.getFilter(filter)
	timeArgs, timeQuery := filter.queryTime()
	fieldArgs, fieldQuery := filter.queryFields()

	if len(fieldArgs) > 0 {
		fieldQuery = "AND " + fieldQuery
	}

	withFillArgs, withFillQuery := filter.withFill()
	query := fmt.Sprintf(`SELECT day, toUInt64(avg(time_on_page)) average_time_spent_seconds
		FROM (
			SELECT toDate(time, '%s') day, neighbor(previous_time_on_page_seconds, 1, 0) time_on_page
			FROM (
				SELECT *
				FROM hit
				WHERE %s
				ORDER BY fingerprint, time
			)
			WHERE time_on_page > 0
			%s
		)
		GROUP BY day
		ORDER BY day %s`, filter.Timezone.String(), timeQuery, fieldQuery, withFillQuery)
	timeArgs = append(timeArgs, fieldArgs...)
	timeArgs = append(timeArgs, withFillArgs...)
	var stats []TimeSpentStats

	if err := analyzer.store.Select(&stats, query, timeArgs...); err != nil {
		return nil, err
	}

	return stats, nil
}

// TotalTimeOnPage returns the total time on page in seconds.
func (analyzer *Analyzer) TotalTimeOnPage(filter *Filter) (int, error) {
	filter = analyzer.getFilter(filter)
	timeArgs, timeQuery := filter.queryTime()
	fieldArgs, fieldQuery := filter.queryFields()

	if fieldQuery != "" {
		fieldQuery = "WHERE " + fieldQuery
	}

	query := fmt.Sprintf(`SELECT sum(time_on_page) average_time_spent_seconds
		FROM (
			SELECT neighbor(previous_time_on_page_seconds, 1, 0) time_on_page
			FROM (
				SELECT *
				FROM hit
				WHERE %s
				ORDER BY fingerprint, time
			)
			%s
		)`, timeQuery, fieldQuery)
	timeArgs = append(timeArgs, fieldArgs...)
	stats := new(struct {
		AverageTimeSpentSeconds int `db:"average_time_spent_seconds" json:"average_time_spent_seconds"`
	})

	if err := analyzer.store.Get(stats, query, timeArgs...); err != nil {
		return 0, err
	}

	return stats.AverageTimeSpentSeconds, nil
}

func (analyzer *Analyzer) calculateGrowth(current, previous int) float64 {
	if current == 0 && previous == 0 {
		return 0
	} else if previous == 0 {
		return 1
	}

	c := float64(current)
	p := float64(previous)
	return (c - p) / p
}

func (analyzer *Analyzer) selectByAttribute(results interface{}, filter *Filter, attr string) error {
	filter = analyzer.getFilter(filter)
	args, filterQuery := filter.query()
	query := fmt.Sprintf(byAttributeQuery, attr, filterQuery, attr, filterQuery, attr, attr, filter.withLimit())
	args = append(args, args...)
	return analyzer.store.Select(results, query, args...)
}

func (analyzer *Analyzer) getFilter(filter *Filter) *Filter {
	if filter == nil {
		filter = NewFilter(NullClient)
	}

	filter.validate()
	return filter
}
