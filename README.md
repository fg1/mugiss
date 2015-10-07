µGISS - micro Geographic Information System Service
===================================================

`mugiss` is a small web service for doing offline reverse geocoding (and possibly more in the future). Given a latitude and longitude pair, `mugiss` will return you the corresponding city and country name.

While various alternatives are available for this type of tasks, they usually tend to be based on the use of an external database.
The goal of `mugiss` is to be simple to use, without complicated setup process, lightweight and fast.

*Warning:* `mugiss` is currently in alpha stage.


## Installation

On Debian/Ubuntu:
```
# apt-get install libgeos-dev
$ git clone https://github.com/fg1/mugiss.git
$ cd mugiss
$ go get
$ go build
```


## Usage

First you need to download the countries you are interested from the [gisgraph website](http://download.gisgraphy.com/openstreetmap/csv/cities/).

You can then start the webserver:
```
$ ./mugiss -d FR.tar.bz2
```
The server will then load the file (`FR.tar.bz2` in this case), and start serving HTTP requests once finished.

Example:
```
$ curl -s http://127.0.0.1:8080/rg/48.858222/2.2945 | aeson-pretty -s
{
    "city": "Paris",
    "country": "France",
    "country_iso3166-2": "FR",
    "country_iso3166-3": "FRA",
    "type": "city"
}
```


## Current GIS data sources

- City administrative boundaries: [gisgraphy.com](http://download.gisgraphy.com/openstreetmap/csv/cities/)
- Country administrative boundaries: [delight-im/FreeGeoDB](https://github.com/delight-im/FreeGeoDB)
- ISO-3166 country codes: [lukes/ISO-3166-Countries-with-Regional-Codes](https://github.com/lukes/ISO-3166-Countries-with-Regional-Codes)


## TODO list for future versions

Here is a list of possible future improvements.

- [ ] Better memory management
- [ ] Automatically download the GIS data
    - [ ] Embed a world map in the server to determine which country to download
    - [ ] Download the country data via a script or directly from the server
- [ ] Add parsers for data from other sources (ex: [Natural Earth](http://www.naturalearthdata.com/downloads/), [Global Administrative Areas](http://www.gadm.org/))
- [ ] Add more informations in the returned object
- [ ] Add an info page to manage the server
- [ ] Geocoding (address to lat/lng)


## Contributing

Contributions are welcome. Have a look at the TODO list above, or come with your own features.

1. [Fork the repository](https://github.com/fg1/mugiss/fork)
2. Create your feature branch (`git checkout -b my-feature`)
3. Format your changes (`go fmt`) and commit it (`git commit -am 'Commit message'`)
4. Push to the branch (`git push origin my-feature`)
5. Create a pull request

