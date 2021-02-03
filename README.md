# population

`population` is an HTTP service and a CLI utility written in Go, which returns
estimates of population sizes within a certain radius of a given place. The URL
to query the service is:

```
/population?place=place_name&radius=radius_in_kms
```

## Installation and running

1. Install [Go](https://golang.org/dl/), [gcc](https://gcc.gnu.org) and
   [make](https://www.gnu.org/software/make/) (macOS users can simply `brew
   install go`, make and gcc come pre-installed with the system).
2. Get some population data from
   [geonames](https://www.geonames.org/countries/) and populate the database:
```
> make release
> ./population load data_file.txt
```
3. Run `make release` and then simply `./population`. The service runs on port
   8080 by default, but you can specify a different port and some other
   options, see `-h`.

## How to update geographical information

`population` depends on country files downloaded from
[geonames](https://www.geonames.org/countries/). You can download additional
files (currently the DB contains 10 of the most populous countries), and then
the procedure is as follows:

```
> ./population dump-db
> ./population load new_data_file.txt
> pkger
```

Afterwards you can delete the dumped sqlite file.

