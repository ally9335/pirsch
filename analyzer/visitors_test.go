package analyzer

import (
	"github.com/pirsch-analytics/pirsch/v4"
	"github.com/pirsch-analytics/pirsch/v4/db"
	"github.com/pirsch-analytics/pirsch/v4/model"
	"github.com/pirsch-analytics/pirsch/v4/util"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAnalyzer_ActiveVisitors(t *testing.T) {
	db.CleanupDB(t, dbClient)
	assert.NoError(t, dbClient.SavePageViews([]model.PageView{
		{VisitorID: 1, Time: time.Now().Add(-time.Minute * 30), Path: "/", Title: "Home"},
		{VisitorID: 1, Time: time.Now().Add(-time.Minute * 20), Path: "/", Title: "Home"},
		{VisitorID: 1, Time: time.Now().Add(-time.Minute * 15), Path: "/bar", Title: "Bar"},
		{VisitorID: 2, Time: time.Now().Add(-time.Minute * 4), Path: "/bar", Title: "Bar"},
		{VisitorID: 2, Time: time.Now().Add(-time.Minute * 3), Path: "/foo", Title: "Foo"},
		{VisitorID: 3, Time: time.Now().Add(-time.Minute * 3), Path: "/", Title: "Home"},
		{VisitorID: 4, Time: time.Now().Add(-time.Minute), Path: "/", Title: "Home"},
	}))
	saveSessions(t, [][]model.Session{
		{
			{Sign: 1, VisitorID: 1, Time: time.Now().Add(-time.Minute * 25), Start: time.Now()},
		},
		{
			{Sign: -1, VisitorID: 1, Time: time.Now().Add(-time.Minute * 25), Start: time.Now()},
			{Sign: 1, VisitorID: 1, Time: time.Now().Add(-time.Minute * 15), Start: time.Now()},
			{Sign: 1, VisitorID: 2, Time: time.Now().Add(-time.Minute * 3), Start: time.Now()},
			{Sign: 1, VisitorID: 3, Time: time.Now().Add(-time.Minute * 5), Start: time.Now()},
		},
		{
			{Sign: -1, VisitorID: 3, Time: time.Now().Add(-time.Minute * 5), Start: time.Now()},
			{Sign: 1, VisitorID: 3, Time: time.Now().Add(-time.Minute * 3), Start: time.Now()},
			{Sign: 1, VisitorID: 4, Time: time.Now().Add(-time.Minute), Start: time.Now()},
		},
	})
	time.Sleep(time.Millisecond * 20)
	analyzer := NewAnalyzer(dbClient, nil)
	visitors, count, err := analyzer.Visitors.Active(nil, time.Minute*10)
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.Len(t, visitors, 3)
	assert.Equal(t, "/", visitors[0].Path)
	assert.Equal(t, "/bar", visitors[1].Path)
	assert.Equal(t, "/foo", visitors[2].Path)
	assert.Empty(t, visitors[0].Title)
	assert.Empty(t, visitors[1].Title)
	assert.Empty(t, visitors[2].Title)
	assert.Equal(t, 2, visitors[0].Visitors)
	assert.Equal(t, 1, visitors[1].Visitors)
	assert.Equal(t, 1, visitors[2].Visitors)
	visitors, count, err = analyzer.Visitors.Active(&Filter{Path: []string{"/bar"}}, time.Minute*30)
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Len(t, visitors, 1)
	assert.Equal(t, "/bar", visitors[0].Path)
	assert.Equal(t, 2, visitors[0].Visitors)
	_, _, err = analyzer.Visitors.Active(getMaxFilter(""), time.Minute*10)
	assert.NoError(t, err)
	visitors, count, err = analyzer.Visitors.Active(&Filter{IncludeTitle: true}, time.Minute*10)
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.Len(t, visitors, 3)
	assert.Equal(t, "Home", visitors[0].Title)
	assert.Equal(t, "Bar", visitors[1].Title)
	assert.Equal(t, "Foo", visitors[2].Title)
	_, _, err = analyzer.Visitors.Active(getMaxFilter(""), time.Minute*10)
	assert.NoError(t, err)
	_, _, err = analyzer.Visitors.Active(getMaxFilter("event"), time.Minute*10)
	assert.NoError(t, err)
}

func TestAnalyzer_TotalVisitors(t *testing.T) {
	db.CleanupDB(t, dbClient)
	saveSessions(t, [][]model.Session{
		{
			{Sign: 1, VisitorID: 1, Time: util.PastDay(4).Add(time.Minute * 10), Start: time.Now(), SessionID: 4, ExitPath: "/bar", PageViews: 1, IsBounce: true},
		},
		{
			{Sign: -1, VisitorID: 1, Time: util.PastDay(4).Add(time.Minute * 10), Start: time.Now(), SessionID: 4, ExitPath: "/bar", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 1, Time: util.PastDay(4).Add(time.Minute * 5), Start: time.Now(), SessionID: 4, ExitPath: "/", PageViews: 2, IsBounce: false, DurationSeconds: 300},
			{Sign: 1, VisitorID: 1, Time: util.PastDay(4), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 2, Time: util.PastDay(4), Start: time.Now(), SessionID: 4, ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 2, Time: util.PastDay(4).Add(time.Minute * 10), Start: time.Now(), SessionID: 3, ExitPath: "/bar", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 3, Time: util.PastDay(4), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 4, Time: util.PastDay(4), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 5, Time: util.PastDay(2).Add(time.Minute * 10), Start: time.Now(), SessionID: 2, ExitPath: "/bar", PageViews: 1, IsBounce: true},
		},
		{
			{Sign: -1, VisitorID: 5, Time: util.PastDay(2).Add(time.Minute * 10), Start: time.Now(), SessionID: 2, ExitPath: "/bar", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 5, Time: util.PastDay(2).Add(time.Minute * 5), Start: time.Now(), SessionID: 2, ExitPath: "/bar", PageViews: 2, IsBounce: false, DurationSeconds: 300},
			{Sign: 1, VisitorID: 6, Time: util.PastDay(2).Add(time.Minute * 10), Start: time.Now(), SessionID: 2, ExitPath: "/bar", PageViews: 1, IsBounce: false, DurationSeconds: 600},
			{Sign: 1, VisitorID: 7, Time: util.PastDay(2), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 8, Time: util.PastDay(2), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 9, Time: time.Now().UTC().Add(-time.Minute * 15), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
		},
	})
	assert.NoError(t, dbClient.SavePageViews([]model.PageView{
		{VisitorID: 1, Time: util.PastDay(4).Add(time.Minute * 10), SessionID: 4, Path: "/bar"},
		{VisitorID: 1, Time: util.PastDay(4).Add(time.Minute * 5), SessionID: 4, Path: "/"},
		{VisitorID: 1, Time: util.PastDay(4), Path: "/"},
		{VisitorID: 2, Time: util.PastDay(4), SessionID: 4, Path: "/"},
		{VisitorID: 2, Time: util.PastDay(4).Add(time.Minute * 10), SessionID: 3, Path: "/bar"},
		{VisitorID: 3, Time: util.PastDay(4), Path: "/"},
		{VisitorID: 4, Time: util.PastDay(4), Path: "/"},
		{VisitorID: 5, Time: util.PastDay(2).Add(time.Minute * 10), SessionID: 2, Path: "/bar"},
		{VisitorID: 5, Time: util.PastDay(2).Add(time.Minute * 5), SessionID: 2, Path: "/bar"},
		{VisitorID: 6, Time: util.PastDay(2).Add(time.Minute * 10), SessionID: 2, Path: "/bar"},
		{VisitorID: 7, Time: util.PastDay(2), Path: "/"},
		{VisitorID: 8, Time: util.PastDay(2), Path: "/"},
		{VisitorID: 9, Time: time.Now().UTC().Add(-time.Minute * 15), Path: "/"},
	}))
	assert.NoError(t, dbClient.SaveEvents([]model.Event{
		{VisitorID: 1, SessionID: 4, Time: util.PastDay(4), Name: "event", MetaKeys: []string{"foo", "bar"}, MetaValues: []string{"val0", "val1"}},
	}))
	analyzer := NewAnalyzer(dbClient, nil)
	visitors, err := analyzer.Visitors.Total(&Filter{From: util.PastDay(4), To: util.Today()})
	assert.NoError(t, err)
	assert.Equal(t, 9, visitors.Visitors)
	assert.Equal(t, 11, visitors.Sessions)
	assert.Equal(t, 13, visitors.Views)
	assert.Equal(t, 8, visitors.Bounces)
	assert.InDelta(t, 0.7272, visitors.BounceRate, 0.01)
	visitors, err = analyzer.Visitors.Total(&Filter{From: util.PastDay(2), To: util.Today()})
	assert.NoError(t, err)
	assert.Equal(t, 5, visitors.Visitors)
	assert.Equal(t, 5, visitors.Sessions)
	assert.Equal(t, 6, visitors.Views)
	assert.Equal(t, 3, visitors.Bounces)
	assert.InDelta(t, 0.6, visitors.BounceRate, 0.01)
	visitors, err = analyzer.Visitors.Total(&Filter{From: util.PastDay(1), To: util.Today()})
	assert.NoError(t, err)
	assert.Equal(t, 1, visitors.Visitors)
	assert.Equal(t, 1, visitors.Sessions)
	assert.Equal(t, 1, visitors.Views)
	assert.Equal(t, 1, visitors.Bounces)
	assert.InDelta(t, 1, visitors.BounceRate, 0.01)
	visitors, err = analyzer.Visitors.Total(&Filter{From: util.PastDay(1), To: util.Today()})
	assert.NoError(t, err)
	assert.Equal(t, 1, visitors.Visitors)
	assert.Equal(t, 1, visitors.Sessions)
	assert.Equal(t, 1, visitors.Views)
	assert.Equal(t, 1, visitors.Bounces)
	assert.InDelta(t, 1, visitors.BounceRate, 0.01)
	visitors, err = analyzer.Visitors.Total(&Filter{From: time.Now().UTC().Add(-time.Minute * 15), To: util.Today(), IncludeTime: true})
	assert.NoError(t, err)
	assert.Equal(t, 1, visitors.Visitors)
	assert.Equal(t, 1, visitors.Sessions)
	assert.Equal(t, 1, visitors.Views)
	assert.Equal(t, 1, visitors.Bounces)
	assert.InDelta(t, 1, visitors.BounceRate, 0.01)
	visitors, err = analyzer.Visitors.Total(&Filter{
		From:      util.PastDay(4),
		To:        util.Today(),
		EventName: []string{"event"},
		EventMeta: map[string]string{"foo": "val0"},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, visitors.Visitors)
	assert.Equal(t, 1, visitors.Sessions)
	assert.Equal(t, 1, visitors.Views)
	assert.Equal(t, 0, visitors.Bounces)
	assert.InDelta(t, 0, visitors.BounceRate, 0.01)

	// ignore metadata when event name is not set
	visitors, err = analyzer.Visitors.Total(&Filter{
		From:      util.PastDay(4),
		To:        util.Today(),
		EventMeta: map[string]string{"foo": "val0"},
	})
	assert.NoError(t, err)
}

func TestAnalyzer_VisitorsAndAvgSessionDuration(t *testing.T) {
	db.CleanupDB(t, dbClient)
	saveSessions(t, [][]model.Session{
		{
			{Sign: 1, VisitorID: 1, Time: util.PastDay(4).Add(time.Minute * 10), Start: time.Now(), SessionID: 4, ExitPath: "/bar", PageViews: 1, IsBounce: true},
		},
		{
			{Sign: -1, VisitorID: 1, Time: util.PastDay(4).Add(time.Minute * 10), Start: time.Now(), SessionID: 4, ExitPath: "/bar", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 1, Time: util.PastDay(4).Add(time.Minute * 5), Start: time.Now(), SessionID: 4, ExitPath: "/", PageViews: 2, IsBounce: false, DurationSeconds: 300},
			{Sign: 1, VisitorID: 1, Time: util.PastDay(4), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 2, Time: util.PastDay(4), Start: time.Now(), SessionID: 4, ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 2, Time: util.PastDay(4).Add(time.Minute * 10), Start: time.Now(), SessionID: 3, ExitPath: "/bar", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 3, Time: util.PastDay(4), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 4, Time: util.PastDay(4), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 5, Time: util.PastDay(2).Add(time.Minute * 10), Start: time.Now(), SessionID: 2, ExitPath: "/bar", PageViews: 1, IsBounce: true},
		},
		{
			{Sign: -1, VisitorID: 5, Time: util.PastDay(2).Add(time.Minute * 10), Start: time.Now(), SessionID: 2, ExitPath: "/bar", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 5, Time: util.PastDay(2).Add(time.Minute * 5), Start: time.Now(), SessionID: 2, ExitPath: "/bar", PageViews: 2, IsBounce: false, DurationSeconds: 300},
			{Sign: 1, VisitorID: 6, Time: util.PastDay(2).Add(time.Minute * 10), Start: time.Now(), SessionID: 2, ExitPath: "/bar", PageViews: 1, IsBounce: false, DurationSeconds: 600},
			{Sign: 1, VisitorID: 7, Time: util.PastDay(2), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 8, Time: util.PastDay(2), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 9, Time: util.Today(), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
		},
	})
	assert.NoError(t, dbClient.SavePageViews([]model.PageView{
		{VisitorID: 1, Time: util.PastDay(4).Add(time.Minute * 10), SessionID: 4, Path: "/bar"},
		{VisitorID: 1, Time: util.PastDay(4).Add(time.Minute * 5), SessionID: 4, Path: "/"},
		{VisitorID: 1, Time: util.PastDay(4), Path: "/"},
		{VisitorID: 2, Time: util.PastDay(4), SessionID: 4, Path: "/"},
		{VisitorID: 2, Time: util.PastDay(4).Add(time.Minute * 10), SessionID: 3, Path: "/bar"},
		{VisitorID: 3, Time: util.PastDay(4), Path: "/"},
		{VisitorID: 4, Time: util.PastDay(4), Path: "/"},
		{VisitorID: 5, Time: util.PastDay(2).Add(time.Minute * 10), SessionID: 2, Path: "/bar"},
		{VisitorID: 5, Time: util.PastDay(2).Add(time.Minute * 5), SessionID: 2, Path: "/bar"},
		{VisitorID: 6, Time: util.PastDay(2).Add(time.Minute * 10), SessionID: 2, Path: "/bar"},
		{VisitorID: 7, Time: util.PastDay(2), Path: "/"},
		{VisitorID: 8, Time: util.PastDay(2), Path: "/"},
		{VisitorID: 9, Time: util.Today(), Path: "/"},
	}))
	analyzer := NewAnalyzer(dbClient, nil)
	visitors, err := analyzer.Visitors.ByPeriod(&Filter{From: util.PastDay(4), To: util.Today()})
	assert.NoError(t, err)
	assert.Len(t, visitors, 5)
	assert.Equal(t, util.PastDay(4), visitors[0].Day.Time)
	assert.Equal(t, util.PastDay(3), visitors[1].Day.Time)
	assert.Equal(t, util.PastDay(2), visitors[2].Day.Time)
	assert.Equal(t, util.PastDay(1), visitors[3].Day.Time)
	assert.Equal(t, util.Today(), visitors[4].Day.Time)
	assert.Equal(t, 4, visitors[0].Visitors)
	assert.Equal(t, 0, visitors[1].Visitors)
	assert.Equal(t, 4, visitors[2].Visitors)
	assert.Equal(t, 0, visitors[3].Visitors)
	assert.Equal(t, 1, visitors[4].Visitors)
	assert.Equal(t, 6, visitors[0].Sessions)
	assert.Equal(t, 0, visitors[1].Sessions)
	assert.Equal(t, 4, visitors[2].Sessions)
	assert.Equal(t, 0, visitors[3].Sessions)
	assert.Equal(t, 1, visitors[4].Sessions)
	assert.Equal(t, 7, visitors[0].Views)
	assert.Equal(t, 0, visitors[1].Views)
	assert.Equal(t, 5, visitors[2].Views)
	assert.Equal(t, 0, visitors[3].Views)
	assert.Equal(t, 1, visitors[4].Views)
	assert.Equal(t, 5, visitors[0].Bounces)
	assert.Equal(t, 0, visitors[1].Bounces)
	assert.Equal(t, 2, visitors[2].Bounces)
	assert.Equal(t, 0, visitors[3].Bounces)
	assert.Equal(t, 1, visitors[4].Bounces)
	assert.InDelta(t, 0.8333, visitors[0].BounceRate, 0.01)
	assert.InDelta(t, 0, visitors[1].BounceRate, 0.01)
	assert.InDelta(t, 0.5, visitors[2].BounceRate, 0.01)
	assert.InDelta(t, 0, visitors[3].BounceRate, 0.01)
	assert.InDelta(t, 1, visitors[4].BounceRate, 0.01)
	visitors, err = analyzer.Visitors.ByPeriod(&Filter{Path: []string{"/"}, From: util.PastDay(4), To: util.Today()})
	assert.NoError(t, err)
	assert.Len(t, visitors, 5)
	assert.Equal(t, 4, visitors[0].Visitors)
	assert.Equal(t, 0, visitors[1].Visitors)
	assert.Equal(t, 2, visitors[2].Visitors)
	assert.Equal(t, 0, visitors[3].Visitors)
	assert.Equal(t, 1, visitors[4].Visitors)
	assert.Equal(t, 5, visitors[0].Sessions)
	assert.Equal(t, 0, visitors[1].Sessions)
	assert.Equal(t, 2, visitors[2].Sessions)
	assert.Equal(t, 0, visitors[3].Sessions)
	assert.Equal(t, 1, visitors[4].Sessions)
	assert.Equal(t, 5, visitors[0].Views)
	assert.Equal(t, 0, visitors[1].Views)
	assert.Equal(t, 2, visitors[2].Views)
	assert.Equal(t, 0, visitors[3].Views)
	assert.Equal(t, 1, visitors[4].Views)
	assert.Equal(t, 4, visitors[0].Bounces)
	assert.Equal(t, 0, visitors[1].Bounces)
	assert.Equal(t, 2, visitors[2].Bounces)
	assert.Equal(t, 0, visitors[3].Bounces)
	assert.Equal(t, 1, visitors[4].Bounces)
	assert.InDelta(t, 0.8, visitors[0].BounceRate, 0.01)
	assert.InDelta(t, 0, visitors[1].BounceRate, 0.01)
	assert.InDelta(t, 1, visitors[2].BounceRate, 0.01)
	assert.InDelta(t, 0, visitors[3].BounceRate, 0.01)
	assert.InDelta(t, 1, visitors[4].BounceRate, 0.01)
	asd, err := analyzer.Time.AvgSessionDuration(nil)
	assert.NoError(t, err)
	assert.Len(t, asd, 2)
	assert.Equal(t, util.PastDay(4), asd[0].Day.Time)
	assert.Equal(t, util.PastDay(2), asd[1].Day.Time)
	assert.Equal(t, 300, asd[0].AverageTimeSpentSeconds)
	assert.Equal(t, 450, asd[1].AverageTimeSpentSeconds)
	tsd, err := analyzer.Visitors.totalSessionDuration(&Filter{})
	assert.NoError(t, err)
	assert.Equal(t, 1200, tsd)
	visitors, err = analyzer.Visitors.ByPeriod(&Filter{From: util.PastDay(4), To: util.PastDay(1)})
	assert.NoError(t, err)
	assert.Len(t, visitors, 4)
	assert.Equal(t, util.PastDay(4), visitors[0].Day.Time)
	assert.Equal(t, util.PastDay(2), visitors[2].Day.Time)
	asd, err = analyzer.Time.AvgSessionDuration(&Filter{From: util.PastDay(3), To: util.PastDay(1)})
	assert.NoError(t, err)
	assert.Len(t, asd, 3)
	tsd, err = analyzer.Visitors.totalSessionDuration(&Filter{From: util.PastDay(3), To: util.PastDay(1)})
	assert.NoError(t, err)
	assert.Equal(t, 900, tsd)
	visitors, err = analyzer.Visitors.ByPeriod(&Filter{
		From:   util.PastDay(90),
		To:     util.Today(),
		Period: pirsch.PeriodWeek,
	})
	assert.NoError(t, err)
	assert.True(t, visitors[0].Week.Valid)
	_, err = analyzer.Visitors.ByPeriod(&Filter{
		From:   util.Today(),
		To:     util.Today(),
		Period: pirsch.PeriodWeek,
	})
	assert.NoError(t, err)
	_, err = analyzer.Visitors.ByPeriod(&Filter{
		From:   util.PastDay(1),
		To:     util.Today(),
		Period: pirsch.PeriodWeek,
	})
	assert.NoError(t, err)
	_, err = analyzer.Visitors.ByPeriod(&Filter{
		From:        util.PastDay(90),
		To:          util.Today(),
		PathPattern: []string{"(?i)^/bar"},
	})
	assert.NoError(t, err)
	_, err = analyzer.Visitors.ByPeriod(getMaxFilter(""))
	assert.NoError(t, err)
	_, err = analyzer.Visitors.ByPeriod(getMaxFilter("event"))
	assert.NoError(t, err)
	_, err = analyzer.Time.AvgSessionDuration(&Filter{
		From:   util.PastDay(90),
		To:     util.Today(),
		Period: pirsch.PeriodWeek,
	})
	assert.NoError(t, err)
	_, err = analyzer.Time.AvgSessionDuration(&Filter{
		From:        util.PastDay(90),
		To:          util.Today(),
		PathPattern: []string{"(?i)^/bar"},
	})
	assert.NoError(t, err)
	_, err = analyzer.Time.AvgSessionDuration(getMaxFilter(""))
	assert.NoError(t, err)
	_, err = analyzer.Time.AvgSessionDuration(getMaxFilter("event"))
	assert.NoError(t, err)
	_, err = analyzer.Visitors.totalSessionDuration(getMaxFilter(""))
	assert.NoError(t, err)
}

func TestAnalyzer_Growth(t *testing.T) {
	db.CleanupDB(t, dbClient)
	saveSessions(t, [][]model.Session{
		{
			{Sign: 1, VisitorID: 1, Time: util.PastDay(9).Add(time.Minute * 15), Start: time.Now(), SessionID: 4, ExitPath: "/bar", DurationSeconds: 600, PageViews: 3, IsBounce: false},
			{Sign: 1, VisitorID: 2, Time: util.PastDay(9), Start: time.Now(), ExitPath: "/", PageViews: 5, IsBounce: true},
			{Sign: 1, VisitorID: 3, Time: util.PastDay(4).Add(time.Minute * 15), Start: time.Now(), SessionID: 4, ExitPath: "/bar", DurationSeconds: 600, PageViews: 3, IsBounce: false},
			{Sign: 1, VisitorID: 4, Time: util.PastDay(4), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 5, Time: util.PastDay(4), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 6, Time: util.PastDay(3).Add(time.Minute * 10), Start: time.Now(), SessionID: 3, ExitPath: "/", PageViews: 1, IsBounce: true},
		},
		{
			{Sign: -1, VisitorID: 6, Time: util.PastDay(3).Add(time.Minute * 10), Start: time.Now(), SessionID: 3, ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 6, Time: util.PastDay(3).Add(time.Minute * 5), Start: time.Now(), SessionID: 3, ExitPath: "/foo", DurationSeconds: 300, PageViews: 2, IsBounce: false},
			{Sign: 1, VisitorID: 6, Time: util.PastDay(3), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 7, Time: util.PastDay(3), Start: time.Now(), SessionID: 3, ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 7, Time: util.PastDay(3).Add(time.Minute * 10), Start: time.Now(), SessionID: 31, ExitPath: "/bar", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 8, Time: util.PastDay(3), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 9, Time: util.PastDay(3), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 10, Time: util.PastDay(2).Add(time.Minute * 5), Start: time.Now(), SessionID: 2, ExitPath: "/bar", DurationSeconds: 300, PageViews: 2, IsBounce: false},
			{Sign: 1, VisitorID: 11, Time: util.PastDay(2), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 12, Time: util.PastDay(2), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 13, Time: util.Today(), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
		},
	})
	time.Sleep(time.Millisecond * 20)
	analyzer := NewAnalyzer(dbClient, nil)
	growth, err := analyzer.Visitors.Growth(nil)
	assert.ErrorIs(t, err, ErrNoPeriodOrDay)
	assert.Nil(t, growth)
	growth, err = analyzer.Visitors.Growth(&Filter{From: util.PastDay(2), To: util.PastDay(2)})
	assert.NoError(t, err)
	assert.NotNil(t, growth)
	assert.InDelta(t, 0.5, growth.VisitorsGrowth, 0.001)
	assert.InDelta(t, -0.5, growth.ViewsGrowth, 0.001)
	assert.InDelta(t, 0.5, growth.SessionsGrowth, 0.001)
	assert.InDelta(t, 0.3333, growth.BouncesGrowth, 0.001)
	assert.InDelta(t, -0.5, growth.TimeSpentGrowth, 0.001)
	growth, err = analyzer.Visitors.Growth(&Filter{From: util.PastDay(3), To: util.PastDay(2)})
	assert.NoError(t, err)
	assert.NotNil(t, growth)
	assert.InDelta(t, 1.3333, growth.VisitorsGrowth, 0.001)
	assert.InDelta(t, 1.2, growth.ViewsGrowth, 0.001)
	assert.InDelta(t, 2, growth.SessionsGrowth, 0.001)
	assert.InDelta(t, 0.1666, growth.BouncesGrowth, 0.001)
	assert.InDelta(t, 0, growth.TimeSpentGrowth, 0.001)
	_, err = analyzer.Visitors.Growth(getMaxFilter(""))
	assert.NoError(t, err)
	_, err = analyzer.Visitors.Growth(getMaxFilter("event"))
	assert.NoError(t, err)
}

func TestAnalyzer_GrowthDay(t *testing.T) {
	db.CleanupDB(t, dbClient)
	assert.NoError(t, dbClient.SaveSessions([]model.Session{
		{Sign: 1, VisitorID: 1, Time: util.PastDay(8).Add(time.Hour * 5), Start: time.Now()},
		{Sign: 1, VisitorID: 2, Time: util.PastDay(1).Add(time.Hour * 3), Start: time.Now()},
		{Sign: 1, VisitorID: 3, Time: util.PastDay(1).Add(time.Hour * 4), Start: time.Now()},
		{Sign: 1, VisitorID: 4, Time: util.PastDay(1).Add(time.Hour * 9), Start: time.Now()},
		{Sign: 1, VisitorID: 5, Time: util.Today().Add(time.Hour * 4), Start: time.Now()},
		{Sign: 1, VisitorID: 6, Time: util.Today().Add(time.Hour * 9), Start: time.Now()},
		{Sign: 1, VisitorID: 7, Time: util.Today().Add(time.Hour * 12), Start: time.Now()},
		{Sign: 1, VisitorID: 8, Time: util.Today().Add(time.Hour * 17), Start: time.Now()},
		{Sign: 1, VisitorID: 9, Time: util.Today().Add(time.Hour * 21), Start: time.Now()},
	}))
	time.Sleep(time.Millisecond * 20)
	analyzer := NewAnalyzer(dbClient, nil)

	// Testing for today is hard because it would require messing with the time.Now function.
	// I don't want to do that, so let's assume it works (tested in debug mode) and just get no error for today.
	growth, err := analyzer.Visitors.Growth(&Filter{From: util.Today(), To: util.Today()})
	assert.NoError(t, err)
	assert.NotNil(t, growth)

	growth, err = analyzer.Visitors.Growth(&Filter{From: util.PastDay(1), To: util.PastDay(1)})
	assert.NoError(t, err)
	assert.NotNil(t, growth)
	assert.InDelta(t, 2, growth.VisitorsGrowth, 0.001)
}

func TestAnalyzer_GrowthDayFirstHour(t *testing.T) {
	db.CleanupDB(t, dbClient)
	assert.NoError(t, dbClient.SaveSessions([]model.Session{
		{Sign: 1, VisitorID: 1, Time: util.PastDay(1), Start: time.Now()},
		{Sign: 1, VisitorID: 2, Time: util.PastDay(1).Add(time.Hour * 4), Start: time.Now()},
		{Sign: 1, VisitorID: 3, Time: util.Today(), Start: time.Now()},
	}))
	time.Sleep(time.Millisecond * 20)
	analyzer := NewAnalyzer(dbClient, nil)
	growth, err := analyzer.Visitors.Growth(&Filter{From: util.Today(), To: util.Today().Add(time.Hour * 4), IncludeTime: true})
	assert.NoError(t, err)
	assert.NotNil(t, growth)
	assert.InDelta(t, -0.5, growth.VisitorsGrowth, 0.01)
	growth, err = analyzer.Visitors.Growth(&Filter{From: util.Today(), To: util.Today().Add(time.Hour * 2), IncludeTime: true})
	assert.NoError(t, err)
	assert.NotNil(t, growth)
	assert.InDelta(t, 0, growth.VisitorsGrowth, 0.01)
}

func TestAnalyzer_GrowthNoData(t *testing.T) {
	db.CleanupDB(t, dbClient)
	analyzer := NewAnalyzer(dbClient, nil)
	growth, err := analyzer.Visitors.Growth(&Filter{From: util.PastDay(7), To: util.PastDay(7)})
	assert.NoError(t, err)
	assert.NotNil(t, growth)
	assert.InDelta(t, 0, growth.VisitorsGrowth, 0.001)
	assert.InDelta(t, 0, growth.ViewsGrowth, 0.001)
	assert.InDelta(t, 0, growth.SessionsGrowth, 0.001)
	assert.InDelta(t, 0, growth.BouncesGrowth, 0.001)
	assert.InDelta(t, 0, growth.TimeSpentGrowth, 0.001)
	_, err = analyzer.Visitors.Growth(getMaxFilter(""))
	assert.NoError(t, err)
	_, err = analyzer.Visitors.Growth(getMaxFilter("event"))
	assert.NoError(t, err)
}

func TestAnalyzer_GrowthEvents(t *testing.T) {
	db.CleanupDB(t, dbClient)
	saveSessions(t, [][]model.Session{
		{
			{Sign: 1, VisitorID: 12, SessionID: 3, Time: util.PastDay(9).Add(time.Second * 3), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: -1, VisitorID: 12, SessionID: 3, Time: util.PastDay(9).Add(time.Second * 3), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: 1, VisitorID: 12, SessionID: 3, Time: util.PastDay(9).Add(time.Minute * 5), Start: time.Now(), EntryPath: "/", ExitPath: "/foo"},
			{Sign: 1, VisitorID: 12, Time: util.PastDay(9).Add(time.Second * 5), EntryPath: "/", Start: time.Now(), ExitPath: "/"},
			{Sign: 1, VisitorID: 13, SessionID: 3, Time: util.PastDay(9).Add(time.Second * 6), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: 1, VisitorID: 13, SessionID: 31, Time: util.PastDay(9).Add(time.Minute * 10), Start: time.Now(), EntryPath: "/bar", ExitPath: "/bar"},
			{Sign: 1, VisitorID: 14, Time: util.PastDay(9).Add(time.Second * 7), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: 1, VisitorID: 15, Time: util.PastDay(9).Add(time.Second * 8), Start: time.Now(), EntryPath: "/", ExitPath: "/"},

			{Sign: 1, VisitorID: 1, SessionID: 4, Time: util.PastDay(4).Add(-time.Second), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
		},
		{
			{Sign: -1, VisitorID: 1, SessionID: 4, Time: util.PastDay(4).Add(-time.Second), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: 1, VisitorID: 1, SessionID: 4, Time: util.PastDay(4).Add(time.Minute * 5), Start: time.Now(), EntryPath: "/", ExitPath: "/foo"},
			{Sign: -1, VisitorID: 1, SessionID: 4, Time: util.PastDay(4).Add(time.Minute * 5), Start: time.Now(), EntryPath: "/", ExitPath: "/foo"},
			{Sign: 1, VisitorID: 1, SessionID: 4, Time: util.PastDay(4).Add(time.Minute * 15), Start: time.Now(), EntryPath: "/", ExitPath: "/bar"},
			{Sign: 1, VisitorID: 2, Time: util.PastDay(4).Add(time.Second * 2), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: 1, VisitorID: 3, Time: util.PastDay(4).Add(time.Second * 3), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: 1, VisitorID: 4, SessionID: 3, Time: util.PastDay(3).Add(time.Second * 3), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: -1, VisitorID: 4, SessionID: 3, Time: util.PastDay(3).Add(time.Second * 3), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: 1, VisitorID: 4, SessionID: 3, Time: util.PastDay(3).Add(time.Minute * 5), Start: time.Now(), EntryPath: "/", ExitPath: "/foo"},
			{Sign: 1, VisitorID: 4, Time: util.PastDay(3).Add(time.Second * 5), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: 1, VisitorID: 5, SessionID: 3, Time: util.PastDay(3).Add(time.Second * 6), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: 1, VisitorID: 5, SessionID: 31, Time: util.PastDay(3).Add(time.Minute * 10), Start: time.Now(), EntryPath: "/bar", ExitPath: "/bar"},
			{Sign: 1, VisitorID: 6, Time: util.PastDay(3).Add(time.Second * 7), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: 1, VisitorID: 7, Time: util.PastDay(3).Add(time.Second * 8), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: 1, VisitorID: 8, SessionID: 2, Time: util.PastDay(2).Add(time.Second * 9), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: -1, VisitorID: 8, SessionID: 2, Time: util.PastDay(2).Add(time.Second * 9), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: 1, VisitorID: 8, SessionID: 2, Time: util.PastDay(2).Add(time.Minute * 5), Start: time.Now(), EntryPath: "/", ExitPath: "/bar"},
			{Sign: 1, VisitorID: 9, Time: util.PastDay(2).Add(time.Second * 10), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: 1, VisitorID: 10, Time: util.PastDay(2).Add(time.Second * 11), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
			{Sign: 1, VisitorID: 11, Time: util.Today().Add(time.Second * 12), Start: time.Now(), EntryPath: "/", ExitPath: "/"},
		},
	})
	assert.NoError(t, dbClient.SaveEvents([]model.Event{
		{Name: "event1", VisitorID: 13, Time: util.PastDay(9).Add(time.Second * 4), SessionID: 3, Path: "/"},
		{Name: "event1", DurationSeconds: 300, VisitorID: 14, Time: util.PastDay(9).Add(time.Minute * 5), SessionID: 3, Path: "/foo"},
		{Name: "event1", VisitorID: 14, Time: util.PastDay(9).Add(time.Second * 5), Path: "/"},
		{Name: "event1", VisitorID: 15, Time: util.PastDay(9).Add(time.Second * 6), SessionID: 3, Path: "/"},
		{Name: "event1", VisitorID: 15, Time: util.PastDay(9).Add(time.Minute * 10), SessionID: 31, Path: "/bar"},
		{Name: "event1", VisitorID: 16, Time: util.PastDay(9).Add(time.Second * 7), Path: "/"},
		{Name: "event1", VisitorID: 17, Time: util.PastDay(9).Add(time.Second * 8), Path: "/"},

		{Name: "event1", VisitorID: 1, Time: util.PastDay(4).Add(time.Second), SessionID: 4, Path: "/"},
		{Name: "event1", DurationSeconds: 300, VisitorID: 1, Time: util.PastDay(4).Add(time.Minute * 5), SessionID: 4, Path: "/foo"},
		{Name: "event1", DurationSeconds: 600, VisitorID: 1, Time: util.PastDay(4).Add(time.Minute * 15), SessionID: 4, Path: "/bar"},
		{Name: "event1", VisitorID: 2, Time: util.PastDay(4).Add(time.Second * 2), Path: "/"},
		{Name: "event1", VisitorID: 3, Time: util.PastDay(4).Add(time.Second * 3), Path: "/"},
		{Name: "event1", VisitorID: 4, Time: util.PastDay(3).Add(time.Second * 4), SessionID: 3, Path: "/"},
		{Name: "event1", DurationSeconds: 300, VisitorID: 4, Time: util.PastDay(3).Add(time.Minute * 5), SessionID: 3, Path: "/foo"},
		{Name: "event1", VisitorID: 4, Time: util.PastDay(3).Add(time.Second * 5), Path: "/"},
		{Name: "event1", VisitorID: 5, Time: util.PastDay(3).Add(time.Second * 6), SessionID: 3, Path: "/"},
		{Name: "event1", VisitorID: 5, Time: util.PastDay(3).Add(time.Minute * 10), SessionID: 31, Path: "/bar"},
		{Name: "event1", VisitorID: 6, Time: util.PastDay(3).Add(time.Second * 7), Path: "/"},
		{Name: "event1", VisitorID: 7, Time: util.PastDay(3).Add(time.Second * 8), Path: "/"},
		{Name: "event1", VisitorID: 8, Time: util.PastDay(2).Add(time.Second * 9), SessionID: 2, Path: "/"},
		{Name: "event1", DurationSeconds: 300, VisitorID: 8, Time: util.PastDay(2).Add(time.Minute * 5), SessionID: 2, Path: "/bar"},
		{Name: "event1", VisitorID: 9, Time: util.PastDay(2).Add(time.Second * 10), Path: "/"},
		{Name: "event1", VisitorID: 10, Time: util.PastDay(2).Add(time.Second * 11), Path: "/"},
		{Name: "event1", VisitorID: 11, Time: util.Today().Add(time.Second * 12), Path: "/"},
	}))
	time.Sleep(time.Millisecond * 20)
	analyzer := NewAnalyzer(dbClient, nil)
	growth, err := analyzer.Visitors.Growth(nil)
	assert.ErrorIs(t, err, ErrNoPeriodOrDay)
	assert.Nil(t, growth)
	growth, err = analyzer.Visitors.Growth(&Filter{From: util.PastDay(2), To: util.PastDay(2), EventName: []string{"event1"}})
	assert.NoError(t, err)
	assert.NotNil(t, growth)
	assert.InDelta(t, 0.5, growth.VisitorsGrowth, 0.001)
	assert.InDelta(t, 1, growth.ViewsGrowth, 0.001)
	assert.InDelta(t, 0.5, growth.SessionsGrowth, 0.001)
	assert.InDelta(t, 1, growth.TimeSpentGrowth, 0.001)
	analyzer = NewAnalyzer(dbClient, &Config{
		DisableBotFilter: true,
	})
	growth, err = analyzer.Visitors.Growth(&Filter{From: util.PastDay(3), To: util.PastDay(2), EventName: []string{"event1"}})
	assert.NoError(t, err)
	assert.NotNil(t, growth)
	assert.InDelta(t, 1.3333, growth.VisitorsGrowth, 0.001)
	assert.InDelta(t, 1.2, growth.ViewsGrowth, 0.001)
	assert.InDelta(t, 2, growth.SessionsGrowth, 0.001)
	assert.InDelta(t, -0.3333, growth.TimeSpentGrowth, 0.001)
	growth, err = analyzer.Visitors.Growth(&Filter{From: util.PastDay(3), To: util.PastDay(2), EventName: []string{"event1"}, Path: []string{"/bar"}})
	assert.NoError(t, err)
	assert.NotNil(t, growth)
	assert.InDelta(t, 1, growth.VisitorsGrowth, 0.001)
	assert.InDelta(t, 1, growth.ViewsGrowth, 0.001)
	assert.InDelta(t, 1, growth.SessionsGrowth, 0.001)
	assert.InDelta(t, -0.5, growth.TimeSpentGrowth, 0.001)
	_, err = analyzer.Visitors.Growth(getMaxFilter("event1"))
	assert.NoError(t, err)
}

func TestAnalyzer_VisitorHours(t *testing.T) {
	db.CleanupDB(t, dbClient)
	saveSessions(t, [][]model.Session{
		{
			{Sign: 1, VisitorID: 1, Time: util.PastDay(2).Add(time.Hour * 3), Start: time.Now(), ExitPath: "/foo", PageViews: 1, IsBounce: true},
		},
		{
			{Sign: -1, VisitorID: 1, Time: util.PastDay(2).Add(time.Hour * 3), Start: time.Now(), ExitPath: "/foo", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 1, Time: util.PastDay(2).Add(time.Hour * 3), Start: time.Now(), ExitPath: "/", PageViews: 2, IsBounce: false},
			{Sign: 1, VisitorID: 2, Time: util.PastDay(2).Add(time.Hour * 8), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 3, Time: util.PastDay(1).Add(time.Hour * 4), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 4, Time: util.PastDay(1).Add(time.Hour * 5), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 5, Time: util.PastDay(1).Add(time.Hour * 8), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 6, Time: util.Today().Add(time.Hour * 5), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 7, Time: util.Today().Add(time.Hour * 10), Start: time.Now(), ExitPath: "/", PageViews: 1, IsBounce: true},
		},
	})
	assert.NoError(t, dbClient.SavePageViews([]model.PageView{
		{VisitorID: 1, Time: util.PastDay(2).Add(time.Hour*2 + time.Minute*30), Path: "/foo"},
		{VisitorID: 1, Time: util.PastDay(2).Add(time.Hour * 3), Path: "/"},
		{VisitorID: 2, Time: util.PastDay(2).Add(time.Hour * 8), Path: "/"},
		{VisitorID: 3, Time: util.PastDay(1).Add(time.Hour * 4), Path: "/"},
		{VisitorID: 4, Time: util.PastDay(1).Add(time.Hour * 5), Path: "/"},
		{VisitorID: 5, Time: util.PastDay(1).Add(time.Hour * 8), Path: "/"},
		{VisitorID: 6, Time: util.Today().Add(time.Hour * 5), Path: "/"},
		{VisitorID: 7, Time: util.Today().Add(time.Hour * 10), Path: "/"},
	}))
	time.Sleep(time.Millisecond * 20)
	analyzer := NewAnalyzer(dbClient, nil)
	visitors, err := analyzer.Visitors.ByHour(nil)
	assert.NoError(t, err)
	assert.Len(t, visitors, 24)
	assert.Equal(t, 1, visitors[3].Visitors)
	assert.Equal(t, 1, visitors[4].Visitors)
	assert.Equal(t, 2, visitors[5].Visitors)
	assert.Equal(t, 2, visitors[8].Visitors)
	assert.Equal(t, 1, visitors[10].Visitors)
	assert.Equal(t, 2, visitors[3].Views)
	assert.Equal(t, 1, visitors[4].Views)
	assert.Equal(t, 2, visitors[5].Views)
	assert.Equal(t, 2, visitors[8].Views)
	assert.Equal(t, 1, visitors[10].Views)
	assert.Equal(t, 1, visitors[3].Sessions)
	assert.Equal(t, 1, visitors[4].Sessions)
	assert.Equal(t, 2, visitors[5].Sessions)
	assert.Equal(t, 2, visitors[8].Sessions)
	assert.Equal(t, 1, visitors[10].Sessions)
	assert.Equal(t, 0, visitors[3].Bounces)
	assert.Equal(t, 1, visitors[4].Bounces)
	assert.Equal(t, 2, visitors[5].Bounces)
	assert.Equal(t, 2, visitors[8].Bounces)
	assert.Equal(t, 1, visitors[10].Bounces)
	assert.InDelta(t, 0, visitors[3].BounceRate, 0.01)
	assert.InDelta(t, 1, visitors[4].BounceRate, 0.01)
	assert.InDelta(t, 1, visitors[5].BounceRate, 0.01)
	assert.InDelta(t, 1, visitors[8].BounceRate, 0.01)
	assert.InDelta(t, 1, visitors[10].BounceRate, 0.01)
	visitors, err = analyzer.Visitors.ByHour(&Filter{From: util.PastDay(1), To: util.Today()})
	assert.NoError(t, err)
	assert.Len(t, visitors, 24)
	assert.Equal(t, 1, visitors[4].Visitors)
	assert.Equal(t, 2, visitors[5].Visitors)
	assert.Equal(t, 1, visitors[8].Visitors)
	assert.Equal(t, 1, visitors[10].Visitors)
	assert.Equal(t, 1, visitors[4].Views)
	assert.Equal(t, 2, visitors[5].Views)
	assert.Equal(t, 1, visitors[8].Views)
	assert.Equal(t, 1, visitors[10].Views)
	assert.Equal(t, 1, visitors[4].Sessions)
	assert.Equal(t, 2, visitors[5].Sessions)
	assert.Equal(t, 1, visitors[8].Sessions)
	assert.Equal(t, 1, visitors[10].Sessions)
	assert.Equal(t, 1, visitors[4].Bounces)
	assert.Equal(t, 2, visitors[5].Bounces)
	assert.Equal(t, 1, visitors[8].Bounces)
	assert.Equal(t, 1, visitors[10].Bounces)
	assert.InDelta(t, 1, visitors[4].BounceRate, 0.01)
	assert.InDelta(t, 1, visitors[5].BounceRate, 0.01)
	assert.InDelta(t, 1, visitors[8].BounceRate, 0.01)
	assert.InDelta(t, 1, visitors[10].BounceRate, 0.01)
	_, err = analyzer.Visitors.ByHour(getMaxFilter(""))
	assert.NoError(t, err)
	_, err = analyzer.Visitors.ByHour(getMaxFilter("event"))
	assert.NoError(t, err)
}

func TestAnalyzer_Referrer(t *testing.T) {
	db.CleanupDB(t, dbClient)
	saveSessions(t, [][]model.Session{
		{
			{Sign: 1, VisitorID: 1, Time: time.Now().Add(time.Minute * 2), Start: time.Now(), ExitPath: "/exit", Referrer: "ref2/foo", ReferrerName: "Ref2", PageViews: 3, IsBounce: true},
		},
		{
			{Sign: -1, VisitorID: 1, Time: time.Now().Add(time.Minute * 2), Start: time.Now(), ExitPath: "/exit", Referrer: "ref2/foo", ReferrerName: "Ref2", PageViews: 3, IsBounce: true},
			{Sign: 1, VisitorID: 1, Time: time.Now().Add(time.Minute * 2), Start: time.Now(), ExitPath: "/", Referrer: "ref2/foo", ReferrerName: "Ref2", PageViews: 3, IsBounce: false},
			{Sign: 1, VisitorID: 2, Time: time.Now().Add(time.Minute), Start: time.Now(), ExitPath: "/bar", Referrer: "ref3/foo", ReferrerName: "Ref3", PageViews: 2, IsBounce: false},
			{Sign: 1, VisitorID: 3, Time: time.Now(), Start: time.Now(), ExitPath: "/", Referrer: "ref1/foo", ReferrerName: "Ref1", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 4, Time: time.Now(), Start: time.Now(), ExitPath: "/", Referrer: "ref1/bar", ReferrerName: "Ref1", PageViews: 1, IsBounce: true},
		},
	})
	time.Sleep(time.Millisecond * 20)
	analyzer := NewAnalyzer(dbClient, nil)
	visitors, err := analyzer.Visitors.Referrer(nil)
	assert.NoError(t, err)
	assert.Len(t, visitors, 3)
	assert.Equal(t, "Ref1", visitors[0].ReferrerName)
	assert.Equal(t, "Ref2", visitors[1].ReferrerName)
	assert.Equal(t, "Ref3", visitors[2].ReferrerName)
	assert.Equal(t, 2, visitors[0].Visitors)
	assert.Equal(t, 1, visitors[1].Visitors)
	assert.Equal(t, 1, visitors[2].Visitors)
	assert.InDelta(t, 0.5, visitors[0].RelativeVisitors, 0.01)
	assert.InDelta(t, 0.25, visitors[1].RelativeVisitors, 0.01)
	assert.InDelta(t, 0.25, visitors[2].RelativeVisitors, 0.01)
	assert.Equal(t, 2, visitors[0].Bounces)
	assert.Equal(t, 0, visitors[1].Bounces)
	assert.Equal(t, 0, visitors[2].Bounces)
	assert.InDelta(t, 1, visitors[0].BounceRate, 0.01)
	assert.InDelta(t, 0, visitors[1].BounceRate, 0.01)
	assert.InDelta(t, 0, visitors[2].BounceRate, 0.01)
	_, err = analyzer.Visitors.Referrer(getMaxFilter(""))
	assert.NoError(t, err)
	_, err = analyzer.Visitors.Referrer(getMaxFilter("event"))
	assert.NoError(t, err)
	visitors, err = analyzer.Visitors.Referrer(&Filter{Limit: 1})
	assert.NoError(t, err)
	assert.Len(t, visitors, 1)
	_, err = analyzer.Visitors.Referrer(&Filter{Offset: 1, Limit: 10, Sort: []Sort{
		{
			Field:     FieldReferrerName,
			Direction: pirsch.DirectionASC,
		},
	}, Search: []Search{
		{
			Field: FieldReferrerName,
			Input: "ref",
		},
	}})
	assert.NoError(t, err)

	// filter for referrer name
	visitors, err = analyzer.Visitors.Referrer(&Filter{ReferrerName: []string{"Ref1"}})
	assert.NoError(t, err)
	assert.Len(t, visitors, 2)
	assert.Equal(t, "Ref1", visitors[0].ReferrerName)
	assert.Equal(t, "Ref1", visitors[1].ReferrerName)
	assert.Equal(t, "ref1/bar", visitors[0].Referrer)
	assert.Equal(t, "ref1/foo", visitors[1].Referrer)
	assert.Equal(t, 1, visitors[0].Visitors)
	assert.Equal(t, 1, visitors[1].Visitors)
	assert.Equal(t, 1, visitors[0].Bounces)
	assert.Equal(t, 1, visitors[1].Bounces)
	assert.InDelta(t, 1, visitors[0].BounceRate, 0.01)
	assert.InDelta(t, 1, visitors[1].BounceRate, 0.01)

	// filter for full referrer
	visitors, err = analyzer.Visitors.Referrer(&Filter{Referrer: []string{"ref1/foo"}})
	assert.NoError(t, err)
	assert.Len(t, visitors, 1)
	assert.Equal(t, "Ref1", visitors[0].ReferrerName)
	assert.Equal(t, "ref1/foo", visitors[0].Referrer)
	assert.Equal(t, 1, visitors[0].Visitors)
	assert.Equal(t, 1, visitors[0].Bounces)
	assert.InDelta(t, 1, visitors[0].BounceRate, 0.01)

	// filter for referrer name and full referrer
	visitors, err = analyzer.Visitors.Referrer(&Filter{ReferrerName: []string{"Ref1"}, Referrer: []string{"ref1/foo"}})
	assert.NoError(t, err)
	assert.Len(t, visitors, 1)
	assert.Equal(t, "Ref1", visitors[0].ReferrerName)
	assert.Equal(t, "ref1/foo", visitors[0].Referrer)
	assert.Equal(t, 1, visitors[0].Visitors)
	assert.Equal(t, 1, visitors[0].Bounces)
	assert.InDelta(t, 1, visitors[0].BounceRate, 0.01)
}

func TestAnalyzer_ReferrerUnknown(t *testing.T) {
	db.CleanupDB(t, dbClient)
	saveSessions(t, [][]model.Session{
		{
			{Sign: 1, VisitorID: 1, Time: time.Now().Add(time.Minute * 2), Start: time.Now(), SessionID: 1, ExitPath: "/exit", PageViews: 3, IsBounce: true},
		},
		{
			{Sign: -1, VisitorID: 1, Time: time.Now().Add(time.Minute * 2), Start: time.Now(), SessionID: 1, ExitPath: "/exit", PageViews: 3, IsBounce: true},
			{Sign: 1, VisitorID: 1, Time: time.Now().Add(time.Minute * 2), Start: time.Now(), SessionID: 1, ExitPath: "/", PageViews: 3, IsBounce: true},
			{Sign: 1, VisitorID: 2, Time: time.Now().Add(time.Minute * 2), Start: time.Now(), SessionID: 1, ExitPath: "/", PageViews: 3, IsBounce: false},
			{Sign: 1, VisitorID: 3, Time: time.Now().Add(time.Minute), Start: time.Now(), SessionID: 3, ExitPath: "/bar", Referrer: "ref3", PageViews: 2, IsBounce: false},
			{Sign: 1, VisitorID: 4, Time: time.Now(), Start: time.Now(), ExitPath: "/", Referrer: "ref1", PageViews: 1, IsBounce: true},
			{Sign: 1, VisitorID: 5, Time: time.Now(), Start: time.Now(), ExitPath: "/", Referrer: "ref1", PageViews: 1, IsBounce: true},
		},
	})
	time.Sleep(time.Millisecond * 20)
	analyzer := NewAnalyzer(dbClient, nil)
	visitors, err := analyzer.Visitors.Referrer(&Filter{Referrer: []string{pirsch.Unknown}})
	assert.NoError(t, err)
	assert.Len(t, visitors, 1)
	assert.Empty(t, visitors[0].Referrer)
	assert.Equal(t, 2, visitors[0].Visitors)
	assert.InDelta(t, 0.4, visitors[0].RelativeVisitors, 0.01)
	assert.Equal(t, 1, visitors[0].Bounces)
	assert.InDelta(t, 0.5, visitors[0].BounceRate, 0.01)
}

func TestAnalyzer_Timezone(t *testing.T) {
	db.CleanupDB(t, dbClient)
	assert.NoError(t, dbClient.SaveSessions([]model.Session{
		{Sign: 1, VisitorID: 1, Time: util.PastDay(3).Add(time.Hour * 18), Start: time.Now(), ExitPath: "/"}, // 18:00 UTC -> 03:00 Asia/Tokyo
		{Sign: 1, VisitorID: 2, Time: util.PastDay(2), Start: time.Now(), ExitPath: "/"},                     // 00:00 UTC -> 09:00 Asia/Tokyo
		{Sign: 1, VisitorID: 3, Time: util.PastDay(1).Add(time.Hour * 19), Start: time.Now(), ExitPath: "/"}, // 19:00 UTC -> 04:00 Asia/Tokyo
	}))
	assert.NoError(t, dbClient.SavePageViews([]model.PageView{
		{VisitorID: 1, Time: util.PastDay(3).Add(time.Hour * 18), Path: "/"}, // 18:00 UTC -> 03:00 Asia/Tokyo
		{VisitorID: 2, Time: util.PastDay(2), Path: "/"},                     // 00:00 UTC -> 09:00 Asia/Tokyo
		{VisitorID: 3, Time: util.PastDay(1).Add(time.Hour * 19), Path: "/"}, // 19:00 UTC -> 04:00 Asia/Tokyo
	}))
	time.Sleep(time.Millisecond * 20)
	analyzer := NewAnalyzer(dbClient, nil)
	visitors, err := analyzer.Visitors.ByPeriod(&Filter{From: util.PastDay(3), To: util.PastDay(1)})
	assert.NoError(t, err)
	assert.Len(t, visitors, 3)
	assert.Equal(t, 1, visitors[0].Visitors)
	assert.Equal(t, 1, visitors[1].Visitors)
	assert.Equal(t, 1, visitors[2].Visitors)
	hours, err := analyzer.Visitors.ByHour(&Filter{From: util.PastDay(3), To: util.PastDay(1)})
	assert.NoError(t, err)
	assert.Equal(t, 1, hours[0].Visitors)
	assert.Equal(t, 1, hours[18].Visitors)
	assert.Equal(t, 1, hours[19].Visitors)
	timezone, err := time.LoadLocation("Asia/Tokyo")
	assert.NoError(t, err)
	visitors, err = analyzer.Visitors.ByPeriod(&Filter{From: util.PastDay(3), To: util.PastDay(1), Timezone: timezone})
	assert.NoError(t, err)
	assert.Len(t, visitors, 3)
	assert.Equal(t, 0, visitors[0].Visitors)
	assert.Equal(t, 2, visitors[1].Visitors)
	assert.Equal(t, 0, visitors[2].Visitors)
	hours, err = analyzer.Visitors.ByHour(&Filter{From: util.PastDay(3), To: util.PastDay(1), Timezone: timezone})
	assert.NoError(t, err)
	assert.Equal(t, 1, hours[3].Visitors)
	assert.Equal(t, 0, hours[4].Visitors) // pushed to the next day, so outside of filter range
	assert.Equal(t, 1, hours[9].Visitors)
}

func TestAnalyzer_CalculateGrowth(t *testing.T) {
	growth := calculateGrowth(0, 0)
	assert.InDelta(t, 0, growth, 0.001)
	growth = calculateGrowth(1000, 0)
	assert.InDelta(t, 1, growth, 0.001)
	growth = calculateGrowth(0, 1000)
	assert.InDelta(t, -1, growth, 0.001)
	growth = calculateGrowth(100, 50)
	assert.InDelta(t, 1, growth, 0.001)
	growth = calculateGrowth(50, 100)
	assert.InDelta(t, -0.5, growth, 0.001)
	growth = calculateGrowth(0.0, 0.0)
	assert.InDelta(t, 0, growth, 0.001)
	growth = calculateGrowth(1000.0, 0.0)
	assert.InDelta(t, 1, growth, 0.001)
	growth = calculateGrowth(0.0, 1000.0)
	assert.InDelta(t, -1, growth, 0.001)
	growth = calculateGrowth(100.0, 50.0)
	assert.InDelta(t, 1, growth, 0.001)
	growth = calculateGrowth(50.0, 100.0)
	assert.InDelta(t, -0.5, growth, 0.001)
}
