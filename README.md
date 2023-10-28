# work-stats

Command Line Based Job Timing Tool.

## build

```sh
go build
```

## command usage

```sh
stats -c conf.yml
# get online
stats up
# get offline
stats down
# export as csv by year and month
stats out
# aggregate the record by year and month
stats
# list the record by year and month
stats ls
```

## args usage

```sh
Usage of ./stats:
  -c string
     configuration file (default "conf.yml")
  -m int
     stats month (default current month)
  -y int
     stats year (default current year)
```

## configuration

```yaml
db: sqlite3_db_path
project: project_name
cursor: current_up_id
```

## list

|   Project   | Year | Month |      Up At |    Down At |
| :---------: | ---: | ----: | ---------: | ---------: |
| cornerstone | 2023 |    10 | 1698466065 | 1698466107 |
| cornerstone | 2023 |    10 | 1698465826 | 1698466057 |

## aggregate

|   Project   | Year | Month | Hours |
| :---------: | ---: | ----: | ----: |
| cornerstone | 2023 |    10 |  5.00 |
