# Data Attribution

This project uses open data from public sources. All event data is property of the respective data providers and is used here under open data terms with proper attribution.

## Primary Data Sources

### Ayuntamiento de Madrid (Madrid City Council)

**Source:** Madrid Open Data Portal
**URL:** https://datos.madrid.es/
**License:** Open data with attribution requirement
**Data Used:** Cultural events (exhibitions, theater, museums, cultural programming)

**Required Attribution:**
```
Ayuntamiento de Madrid – datos.madrid.es
```

**Dataset Details:**
- **Title:** Agenda de actividades y eventos
- **Format:** JSON, XML, CSV
- **Update Frequency:** Real-time/continuous
- **Coverage:** Cultural venues and activities across Madrid

**Terms:**
- Data is provided under Madrid's open data terms
- Attribution must be displayed on any public use
- Data may not be used for unlawful purposes
- No warranty provided for data accuracy or completeness

### EsMadrid.com (Madrid Tourism Board)

**Source:** EsMadrid Open Data
**URL:** https://www.esmadrid.com/opendata/
**License:** Open data
**Data Used:** Tourism and city events (festivals, outdoor activities, tourist events)

**Attribution:** Tourism Board of Madrid (EsMadrid.com)

**Dataset Details:**
- **Title:** Agenda v1
- **Format:** XML
- **Update Frequency:** Regular updates
- **Coverage:** Tourism-focused events and activities in Madrid

**Terms:**
- Data is provided for public and commercial use
- Attribution recommended
- Check EsMadrid.com for specific terms of use

## How Attribution is Implemented

**On the Website:**
- Attribution appears in the footer of every generated page
- Links back to the original data sources are provided
- Users can access the original data portals directly

**In the Code:**
- Attribution is hardcoded in the HTML template
- Cannot be removed or modified without editing source code
- Preserved across all site generations

## Upstream Data Rights

All event descriptions, titles, dates, locations, and other metadata belong to the original publishers (Ayuntamiento de Madrid, EsMadrid.com, and individual event organizers).

**This project provides:**
- Aggregation and filtering by geographic location
- Calendar presentation and formatting
- Static site generation tooling

**This project does NOT claim:**
- Ownership of event data or descriptions
- Copyright over original event information
- Any rights beyond fair use under open data licenses

## Contact for Data Issues

**For issues with event data accuracy:**
- Contact the original data providers (datos.madrid.es, esmadrid.com)
- This project only displays data as provided by upstream sources

**For issues with data presentation on this site:**
- Open an issue on [GitHub](https://github.com/ericphanson/plaza-espana-calendar/issues)

## Additional Acknowledgments

**Go Programming Language:**
- This project is built using Go (https://go.dev/)
- Uses only Go standard library (BSD-3-Clause license)

**Bootstrap Icons:**
- SVG icons from Bootstrap Icons (https://icons.getbootstrap.com/)
- Used for event category and section indicators on main event page
- Licensed under MIT License (https://github.com/twbs/icons/blob/main/LICENSE)
- Copyright (c) 2019-2024 The Bootstrap Authors

**NearlyFreeSpeech.NET:**
- Original deployment hosted on NFSN (https://www.nearlyfreespeech.net/)
- Not affiliated with or endorsed by NFSN

**OpenStreetMap:**
- Geographic coordinates verified using OpenStreetMap data
- OpenStreetMap® is open data, licensed under the Open Data Commons Open Database License (ODbL)
- https://www.openstreetmap.org/copyright

## License for This Project

The **source code** of this project (site generator, templates, styling) is licensed under the MIT License. See LICENSE file.

The **event data** remains property of the original data providers and is subject to their respective licenses and terms of use.

---

**Last Updated:** 2025-10-23
