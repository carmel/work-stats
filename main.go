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

const (
	config = "stats.yaml"
)

var (
	conf Config
	db   *sql.DB
	err  error
	now  = time.Now()

	// config   = flag.String("c", "stats.yaml", "configuration file")
	year     = flag.Int("y", now.Year(), "stats year")
	month    = flag.Int("m", int(now.Month()), "stats month")
	ls       = flag.NewFlagSet("ls", flag.ExitOnError)
	lsYear   = ls.Int("y", *year, "list by year")
	lsMonth  = ls.Int("m", *month, "list by month")
	out      = flag.NewFlagSet("out", flag.ExitOnError)
	outYear  = out.Int("y", *year, "export by year")
	outMonth = out.Int("m", *month, "export by month")
	// header = []string{"Project", "Year", "Month", "Up At", "Down At"}
)

func init() {

	var c []byte
	c, err = os.ReadFile(config)
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

}

func main() {

	db, err = sql.Open("sqlite3", conf.DB)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	flag.Parse()

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
		head, res := list(*year, *month)
		print(res, head...)
	} else {
		switch os.Args[1] {
		case "up":
			var down int
			db.QueryRow("SELECT IFNULL(down_at,0) FROM record WHERE id=?", conf.Cursor).Scan(&down)
			if down == 0 {
				log.Fatalln("failed to up, because the previous operation was not yet down.")
			}

			conf.Cursor = uuid.New().ID()
			_, err = db.ExecContext(ctx, "INSERT INTO record(id,project,year,month,up_at)VALUES(?,?,?,?,?)", conf.Cursor, conf.Project, *year, *month, now.Unix())
			if err != nil {
				log.Fatalln("[up] sql exec:", err)
			}
			var buf []byte
			buf, err = yaml.Marshal(&conf)
			if err != nil {
				log.Fatalln("marshal config model:", err)
			}
			err = os.WriteFile(config, buf, 0)
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
			head, res := list(*lsYear, *lsMonth)
			print(res, head...)

		case "out":
			out.Parse(os.Args[2:])
			head, res := list(*outYear, *outMonth)
			writeCSV(conf.Project, head, res)

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
		query = fmt.Sprintf(query, "project,year,month,printf('%.2f',SUM(down_at-up_at)/3600.0) AS hours")
		query = fmt.Sprintf("%s AND month=%d GROUP BY project,year,month", query, *month)

		var rows *sql.Rows
		rows, err = db.Query(query, conf.Project, *year)
		if err != nil {
			log.Fatalln("failed to select record:", err)
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
		query = fmt.Sprintf(query, "project,year,printf('%.2f',SUM(down_at-up_at)/3600.0) AS hours")
		query = fmt.Sprintf("%s GROUP BY project,year", query)

		var rows *sql.Rows
		rows, err = db.Query(query, conf.Project, *year)
		if err != nil {
			log.Fatalln("failed to select record:", err)
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

func list(year, month int) (head []string, res []map[string]any) {
	query := "SELECT project,year,month,up_at,IFNULL(down_at,0) FROM record WHERE project=? AND year=?"
	if month != 0 {
		query = fmt.Sprintf("%s AND month=%d", query, month)
	}
	var rows *sql.Rows
	rows, err = db.Query(query, conf.Project, year)
	if err != nil {
		log.Fatalln("failed to select record:", err)
	}

	head = []string{"Project", "Year", "Month", "Up", "Down"}

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

func writeCSV(name string, head []string, data []map[string]any) {
	var f *os.File
	name = fmt.Sprintf("%s.csv", name)
	f, err = os.Create(name)
	if err != nil {
		log.Fatalf("failed to create '%s': %s\n", name, err)
	}
	defer f.Close()
	_, err = f.WriteString("\xEF\xBB\xBF") // Marked as UTF-8 BOM
	if err != nil {
		log.Fatalln("failed to write:", err)
	}

	wr := csv.NewWriter(f)
	wr.Write(head) // write the header

	for _, v := range data {
		var r []string
		for _, h := range head {
			r = append(r, fmt.Sprintf("%v", v[h]))
		}
		wr.Write(r)
	}

	wr.Flush()
}
