// Copyright 2022 Tailscale Inc & Contributors
// SPDX-License-Identifier: BSD-3-Clause

package golink

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/go-cmp/cmp"
	cmpopts "github.com/google/go-cmp/cmp/cmpopts"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Test saving, loading, and deleting links for DB.
func Test_DB_SaveLoadDeleteLinks(t *testing.T) {
	var (
		sqldb *sql.DB
		err   error
		mock  sqlmock.Sqlmock
		db    *gorm.DB
	)
	sqldb, mock, err = sqlmock.New()
	if err != nil {
		t.Errorf("Unable to mock DB connection. %e", err)
	}

	db, err = gorm.Open(mysql.New(mysql.Config{
		DriverName:                "mysql",
		Conn:                      sqldb,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{SkipDefaultTransaction: true})

	if err != nil {
		t.Error(err)
	}

	SUT, err := newDB(db)
	if err != nil {
		t.Error(err)
	}

	links := []*Link{
		{Short: "short", Long: "long", ID: linkID("short")},
		{Short: "Foo.Bar", Long: "long", ID: linkID("Foo.Bar")},
	}

	mock.MatchExpectationsInOrder(false)
	for _, link := range links {
		time := sqlmock.AnyArg()
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `links` SET `created_at`=?,`updated_at`=?,`deleted_at`=?,`short`=?,`long`=?,`created`=?,`last_edit`=?,`owner`=? WHERE `links`.`deleted_at` IS NULL AND `id` = ?")).
			WithArgs(time, sqlmock.AnyArg(), nil, link.Short, link.Long, time, time, "", linkID(link.Short)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `links` WHERE id = ? AND `links`.`deleted_at` IS NULL ORDER BY `links`.`id` LIMIT 1")).
			WithArgs(linkID(link.Short)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "short", "long", "created_at", "last_edit"}).
				AddRow(linkID(link.Short), link.Short, link.Long, link.CreatedAt, link.LastEdit))

		if err := SUT.Save(link); err != nil {
			t.Error(err)
		}

		got, err := SUT.Load(link.Short)
		if err != nil {
			t.Error(err)
		}

		if !cmp.Equal(got, link, cmpopts.IgnoreFields(Link{}, "Created", "LastEdit", "UpdatedAt")) {
			t.Errorf("db save and load got %+v, want %+v", *got, *link)
		}
	}

	selected_rows := sqlmock.NewRows([]string{"id", "short", "long", "created_at", "last_edit"})
	for _, link := range links {
		selected_rows.AddRow(linkID(link.Short), link.Short, link.Long, link.CreatedAt, link.LastEdit)
	}

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM `links` WHERE `links`.`deleted_at` IS NUL")).
		WillReturnRows(selected_rows)

	got, err := SUT.LoadAll()
	if err != nil {
		t.Error(err)
	}

	sortLinks := cmpopts.SortSlices(func(a, b *Link) bool {
		return a.Short < b.Short
	})

	if !cmp.Equal(got, links, sortLinks, cmpopts.IgnoreFields(Link{}, "Created", "LastEdit", "UpdatedAt")) {
		t.Errorf("db.LoadAll got %+v, want %+v", got, links)
	}

	for _, link := range links {
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `links` SET `deleted_at`=? WHERE `links`.`id` = ? AND `links`.`deleted_at` IS NULL")).
			WithArgs().
			WillReturnResult(sqlmock.NewResult(0, 1))

		if err := SUT.Delete(link.Short); err != nil {
			t.Error(err)
		}
	}

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM `links` WHERE `links`.`deleted_at` IS NUL")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "short", "long", "created_at", "last_edit"}))

	got, err = SUT.LoadAll()
	if err != nil {
		t.Error(err)
	}

	want := []*Link(nil)
	if !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
		t.Errorf("db.LoadAll got %+v, want %+v", got, want)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

// Test saving, loading, and deleting stats for DB.
func Test_DB_SaveLoadDeleteStats(t *testing.T) {
	var (
		sqldb *sql.DB
		err   error
		mock  sqlmock.Sqlmock
		db    *gorm.DB
	)
	sqldb, mock, err = sqlmock.New()
	if err != nil {
		t.Errorf("Unable to mock DB connection. %e", err)
	}

	db, err = gorm.Open(mysql.New(mysql.Config{
		DriverName:                "mysql",
		Conn:                      sqldb,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{SkipDefaultTransaction: true})

	if err != nil {
		t.Error(err)
	}

	mock.MatchExpectationsInOrder(false)

	SUT, err := newDB(db)
	if err != nil {
		t.Error(err)
	}

	// preload some links
	links := []*Link{
		{Short: "a"},
		{Short: "B-c"},
	}

	time := sqlmock.AnyArg()
	for _, link := range links {
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `links` SET `created_at`=?,`updated_at`=?,`deleted_at`=?,`short`=?,`long`=?,`created`=?,`last_edit`=?,`owner`=? WHERE `links`.`deleted_at` IS NULL AND `id` = ?")).
			WithArgs(time, time, nil, link.Short, link.Long, time, time, "", linkID(link.Short)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		if err := SUT.Save(link); err != nil {
			t.Error(err)
		}
	}

	// Stats to record and then retrieve.
	// Stats to store do not need to be their canonical short name,
	// but returned stats always should be.
	stats := []ClickStats{
		{"a": 1},
		{"b-c": 1},
		{"a": 1, "bc": 2},
	}
	want := ClickStats{
		"a":   2,
		"B-c": 3,
	}

	for _, s := range stats {
		mock.ExpectBegin()
		for id, click := range s {
			mock.ExpectExec(regexp.QuoteMeta(
				"UPDATE `stats` SET `created_at`=?,`updated_at`=?,`deleted_at`=?,`created`=?,`clicks`=? WHERE `stats`.`deleted_at` IS NULL AND `id` = ?")).
				WithArgs(time, time, nil, time, click, linkID(id)).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}
		mock.ExpectCommit()
		if err := SUT.SaveStats(s); err != nil {
			t.Error(err)
		}
	}

	stats_rows := sqlmock.NewRows([]string{"id", "clicks"})
	stats_rows = stats_rows.AddRow("a", 2)
	stats_rows = stats_rows.AddRow("B-c", 3)
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT ID, SUM(Clicks) as Clicks FROM `stats` WHERE `stats`.`deleted_at` IS NULL GROUP BY `id`")).
		WillReturnRows(stats_rows)
	got, err := SUT.LoadStats()
	if err != nil {
		t.Error(err)
	}
	if !cmp.Equal(got, want) {
		t.Errorf("db.LoadStats got %v, want %v", got, want)
	}

	for k := range want {
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `stats` SET `deleted_at`=? WHERE `stats`.`id` = ? AND `stats`.`deleted_at` IS NULL")).
			WithArgs(sqlmock.AnyArg(), linkID(k)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		if err := SUT.DeleteStats(k); err != nil {
			t.Error(err)
		}
	}

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT ID, SUM(Clicks) as Clicks FROM `stats` WHERE `stats`.`deleted_at` IS NULL GROUP BY `id`")).
		WillReturnRows(sqlmock.NewRows([]string{"", ""}))
	got, err = SUT.LoadStats()
	if err != nil {
		t.Error(err)
	}
	want = ClickStats{}
	if !cmp.Equal(got, want) {
		t.Errorf("db.LoadStats got %v, want %v", got, want)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
