[![CI](https://github.com/ericphanson/plazaespana.info/actions/workflows/ci.yml/badge.svg)](https://github.com/ericphanson/plazaespana.info/actions/workflows/ci.yml)

# [plazaespana.info](https://plazaespana.info)

This is a simple static site which displays events happening at or near [Plaza de España](https://www.esmadrid.com/informacion-turistica/plaza-de-espa%C3%B1a) in Madrid, with 7-day weather forecasts for each event.

This is powered by three data sources: event data from datos.madrid.es and esmadrid.com, plus weather forecasts from AEMET (Spanish State Meteorological Agency).
The site collects this data and renders it cleanly in a fast, static format.

## Motivation

This data is available on other sites, but some of them tend to be slow or ad-ridden.
I also wanted a hyper-specific site I could customize to my liking, and explore some static site generator architectures.

## Design

The code in `generator` is a custom static site generator written in Go by Claude Code. We build the generator then deploy it to a server.
Then we _run_ the generator hourly to generate fresh HTML with the latest events.

The actual site users interact with therefore is totally static, and in fact has no javascript at all, just basic HTML and CSS.
It gets its dynamism therefore by being re-generated periodically. This is far from a new idea but I wanted to explore it.

## Development

To build and run a local version of the site:

1. [Install Go](https://go.dev/doc/install)
2. Install [just](https://just.systems/man/en/packages.html)
3. Clone the repo
   ```sh
   git clone https://github.com/ericphanson/plazaespana.info.git
   ```
4. Build and serve the site: `just dev`.

Run `just` to see all the available commands.

## Configuration

See [config.toml](./config.toml) for main configuration.

### Weather Integration

Weather forecasts require an AEMET API key. To enable weather:

1. Register for a free API key at [AEMET OpenData](https://opendata.aemet.es/centrodedescargas/inicio)
2. Set the `AEMET_API_KEY` environment variable:
   ```sh
   export AEMET_API_KEY="your-api-key-here"
   ```
3. Run the generator - weather will be fetched automatically

If the API key is not set, the site will build successfully but without weather forecasts (graceful degradation).

## Deployment

I deploy the site to [NearlyFreeSpeech.NET](NearlyFreeSpeech.NET) by building a FreeBSD version of the site generator, then using their scheduled tasks feature to periodically regenerate the HTML/CSS,
which is served by an Apache web server.
If you have the credentials setup in `.envrc.local` and [direnv](https://direnv.net/) installed, you can deploy it by running `just deploy`.
Pushes to main also trigger deploys.

I also have per-PR previews set up. Making a PR from the repo (not a fork) triggers a deploy to `plazaespana.info/previews/PR$N` where `$N` is the PR number.

## Visitor statistics

I am trying something potentially weird, which is archiving aggregated anonymous visitors statistics [in repo](./awstats-data). No IPs or other personal information is stored. This data is updated irregularly.

## License and attribution

**Software License:** MIT License - See [LICENSE](LICENSE) file for details.

**Data Attribution:** Event and weather data is provided by:
- [Ayuntamiento de Madrid – datos.madrid.es](https://datos.madrid.es) (Cultural events)
- [EsMadrid.com](https://www.esmadrid.com/) (City events)
- [AEMET OpenData](https://www.aemet.es/en/datos_abiertos/AEMET_OpenData) (Weather forecasts)

Attribution is required per Spain Law 18/2015 and Madrid's open data terms. See [ATTRIBUTION.md](ATTRIBUTION.md) for complete details.
