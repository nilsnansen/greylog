package main

import (
	sql "github.com/gwenn/gosqlite"
	"code.google.com/p/go.crypto/bcrypt"
	"errors"
)

var db *sql.Conn

func dbgetstoryversion(storyid int, lang string) *storyversion {
	var title, content, date string

	stmt, err := db.Prepare(`
	select title, content, date
		from storyversion natural join story
		where story = ? and lang = ?;
	`)

	if err != nil { panic(err) }

	err = stmt.Exec(storyid, lang)

	if !sql.Must(stmt.Next()) {
		return nil
	}

	err = stmt.Scan(&title, &content, &date)
	if err != nil { panic(err) }

	return &storyversion{date, title, content}
}

func dbfeeditems(reqlang string, ch chan *mainitem) {
	var (
		story string
		lang, date, title, summary string
		item *mainitem
	)

	stmt, err := db.Prepare(`
	select story, title, summary, date, lang
		from storyversion
			natural join
			(select * from story order by date desc);
	`)
	if err != nil { panic(err) }

	for sql.Must(stmt.Next()) {
		err = stmt.Scan(&story, &title, &summary, &date, &lang)

		if item == nil {
			item = new(mainitem)
			item.StoryID, item.Title, item.Summary, item.Date = story, title, summary, date
		} else if item.StoryID != story {
			ch <- item
			item = new(mainitem)
			item.StoryID, item.Title, item.Summary, item.Date = story, title, summary, date
		}


		if item.Langs == nil {
			item.Langs = make([]string, 0, 8)
		}
		if len(item.Langs) == cap(item.Langs) {
			tmp := make([]string, len(item.Langs), cap(item.Langs) * 2 + 1)
			for i, v := range item.Langs {
				tmp[i] = v
			}
			item.Langs = tmp
		}
		item.Langs = item.Langs[:len(item.Langs) + 1]
		if lang == reqlang {
			item.StoryID, item.Title, item.Summary, item.Date = story, title, summary, date
			tmp := item.Langs[0]
			item.Langs[0] = lang
			if len(item.Langs) != 1 {
				item.Langs[len(item.Langs) - 1] = tmp
			}
		} else {
			item.Langs[len(item.Langs) - 1] = lang
		}
	}
	if item != nil {
		ch <- item
	}
	close(ch)
}

func init() {
	var err error

	db, err = sql.Open("database.db")
	if err != nil { panic(err) }

	err = db.Exec(`
	create table if not exists story (
		story integer not null primary key,
		date integer not null
	);

	create table if not exists storyversion (
		story integer not null references story(story),
		lang text not null,
		title text not null,
		summary text not null,
		content text not null,
		primary key(story, lang)
	);
	
	create table if not exists user (
		username text not null primary key,
		passhash text not null
	);`)

	if err != nil { panic(err) }
}

func dbauthenticate(username, password string) error {
	var passhash string

	stmt, err := db.Prepare(`
	select passhash from user where username = ?;
	`)

	if err != nil { panic(err) }

	stmt.Exec(username)
	if !sql.Must(stmt.Next()) {
		return errors.New("No such user.")
	}

	err = stmt.Scan(&passhash)
	if err != nil { panic(err) }

	err = bcrypt.CompareHashAndPassword([]byte(passhash), []byte(password))

	return err
}
