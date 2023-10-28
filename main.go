package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v2"
)

var (
	conf    Config
	db      *sql.DB
	err     error
	now     = time.Now()
	config  = flag.String("c", "conf.yml", "configuration file")
	year    = flag.Int("y", now.Year(), "stats year")
	month   = flag.Int("m", int(now.Month()), "stats month")
	ls      = flag.NewFlagSet("ls", flag.ExitOnError)
	lsYear  = ls.Int("y", now.Year(), "list by year")
	lsMonth = ls.Int("m", int(now.Month()), "list by month")
	// header = []string{"Project", "Year", "Month", "Up At", "Down At"}
)

func main() {
	flag.Parse()

	var c []byte
	c, err = os.ReadFile(*config)
	if err != nil {
		log.Fatalln(err)
	}

	err = yaml.UnmarshalStrict(c, &conf)
	if err != nil {
		log.Fatalln(err)
	}

	if conf.Project == "" {
		log.Fatalln("Please specify db file path!")
	}

	if conf.Project == "" {
		log.Fatalln("Please specify project name!")
	}

	db, err = sql.Open("sqlite3", conf.DB)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	ctx := context.Background()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS "record" (
			"id"	    INTEGER not null primary key,
			"project"	VARCHAR(120),
			"year"	    INTEGER,
			"month"	    INTEGER,
			"up_at"	    INTEGER,
			"down_at"	INTEGER
		)`,
	)

	if len(os.Args) < 2 {
		head, res := list()
		print(res, head...)
	} else {
		switch os.Args[1] {
		case "up":
			conf.Cursor = uuid.New().ID()
			_, err = db.ExecContext(ctx, "INSERT INTO record(id,project,year,month,up_at)VALUES(?,?,?,?,?)", conf.Cursor, conf.Project, now.Year(), now.Month(), now.Unix())
			if err != nil {
				log.Fatalln("[up] sql exec:", err)
			}
			var buf []byte
			buf, err = yaml.Marshal(&conf)
			if err != nil {
				log.Fatalln("marshal config model:", err)
			}
			err = os.WriteFile(*config, buf, 0)
			if err != nil {
				log.Fatalln("write cursor to yaml:", err)
			}
		case "down":
			if conf.Cursor == 0 {
				log.Fatalln("can not find current up cursor!")
			}
			_, err = db.ExecContext(ctx, "UPDATE record SET down_at=? WHERE id=?", now.Unix(), conf.Cursor)
			if err != nil {
				log.Fatalln("[down] sql exec:", err)
			}
		case "ls":
			ls.Parse(os.Args[2:])
			head, res := list()
			print(res, head...)

		case "out":
			var out *os.File
			name := fmt.Sprintf("%s.csv", conf.Project)
			out, err = os.Create(name)
			if err != nil {
				log.Fatalf("fail to create '%s': %s\n", name, err)
			}

			head, res := list()

			wr := csv.NewWriter(out)
			defer wr.Flush()
			wr.Write(head)

			r := make([]string, len(head))
			for _, v := range res {
				for _, h := range head {
					r = append(r, fmt.Sprintf("%v", v[h]))
				}
				wr.Write(r)
			}
		default:
			head, res := agg()
			print(res, head...)
		}
	}
}

func print(res []map[string]any, head ...string) {
	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := NewTable(head...)
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, d := range res {
		var row []interface{}
		for _, v := range head {
			row = append(row, d[v])
		}
		tbl.AddRow(row...)
	}

	tbl.Print()
}

func agg() (head []string, res []map[string]any) {
	query := "SELECT %s FROM record WHERE project=? AND year=? AND down_at IS NOT NULL"
	if *month != 0 {
		head = []string{"Project", "Year", "Month", "Hours"}
		query = fmt.Sprintf(query, "project,year,month,printf('%.2f',SUM(down_at-up_at)/3600) AS hours")
		query = fmt.Sprintf("%s AND month=%d GROUP BY project,year,month", query, *month)

		var rows *sql.Rows
		rows, err = db.Query(query, conf.Project, *year)
		if err != nil {
			log.Fatalln("fail to select record:", err)
		}

		var r Record
		for rows.Next() {
			rows.Scan(&r.Project, &r.Year, &r.Month, &r.Hours)
			res = append(res, map[string]any{
				head[0]: r.Project,
				head[1]: r.Year,
				head[2]: r.Month,
				head[3]: r.Hours,
			})
		}

	} else {
		head = []string{"Project", "Year", "Hours"}
		query = fmt.Sprintf(query, "project,year,printf('%.2f',SUM(down_at-up_at)/3600) AS hours")
		query = fmt.Sprintf("%s GROUP BY project,year", query)

		var rows *sql.Rows
		rows, err = db.Query(query, conf.Project, *year)
		if err != nil {
			log.Fatalln("fail to select record:", err)
		}

		var r Record
		for rows.Next() {
			rows.Scan(&r.Project, &r.Year, &r.Hours)
			res = append(res, map[string]any{
				head[0]: r.Project,
				head[1]: r.Year,
				head[2]: r.Hours,
			})
		}
	}
	return
}

func list() (head []string, res []map[string]any) {
	query := "SELECT project,year,month,up_at,IFNULL(down_at,0) FROM record WHERE project=? AND year=?"
	if *lsMonth != 0 {
		query = fmt.Sprintf("%s AND month=%d", query, *lsMonth)
	}
	var rows *sql.Rows
	rows, err = db.Query(query, conf.Project, *lsYear)
	if err != nil {
		log.Fatalln("fail to select record:", err)
	}

	head = []string{"Project", "Year", "Month", "Up At", "Down At"}

	var r Record
	for rows.Next() {
		rows.Scan(&r.Project, &r.Year, &r.Month, &r.UpAt, &r.DownAt)
		res = append(res, map[string]any{
			head[0]: r.Project,
			head[1]: r.Year,
			head[2]: r.Month,
			head[3]: r.UpAt,
			head[4]: r.DownAt,
		})
	}
	return
}
