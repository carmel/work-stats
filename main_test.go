package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/fatih/color"
)

func TestStats(t *testing.T) {
	main()
}

func TestQuery(t *testing.T) {
	db, err = sql.Open("sqlite3", conf.DB)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	var down int
	db.QueryRow("SELECT IFNULL(down_at,0) FROM record WHERE id=?", conf.Cursor).Scan(&down)

	fmt.Println(down)
}

func TestSubcommand(t *testing.T) {
	fooCmd := flag.NewFlagSet("foo", flag.ExitOnError)
	fooEnable := fooCmd.Bool("enable", false, "enable")
	fooName := fooCmd.String("name", "", "name")

	barCmd := flag.NewFlagSet("bar", flag.ExitOnError)
	barLevel := barCmd.Int("level", 0, "level")

	if len(os.Args) < 2 {
		fmt.Println("expected 'foo' or 'bar' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {

	case "foo":
		fooCmd.Parse(os.Args[2:])
		fmt.Println("subcommand 'foo'")
		fmt.Println("  enable:", *fooEnable)
		fmt.Println("  name:", *fooName)
		fmt.Println("  tail:", fooCmd.Args())
	case "bar":
		barCmd.Parse(os.Args[2:])
		fmt.Println("subcommand 'bar'")
		fmt.Println("  level:", *barLevel)
		fmt.Println("  tail:", barCmd.Args())
	default:
		fmt.Println("expected 'foo' or 'bar' subcommands")
		os.Exit(1)
	}
}

func TestTable(t *testing.T) {
	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := NewTable("ID", "Name", "Score")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	data := []map[string]any{
		{"ID": 1, "Name": "n1", "Score": 10},
		{"ID": 2, "Name": "n3", "Score": 20},
		{"ID": 3, "Name": "n333333333333", "Score": 100},
		{"ID": 4, "Name": "n44444444444444444", "Score": 10000000000000},
	}
	for _, d := range data {
		tbl.AddRow(d["ID"], d["Name"], d["Score"])
	}

	tbl.Print()
}
