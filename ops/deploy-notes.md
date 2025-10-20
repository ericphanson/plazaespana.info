# Deployment to NearlyFreeSpeech.NET

## Initial Setup

1. **Build FreeBSD binary locally:**
   ```bash
   ./scripts/build-freebsd.sh
   ```

2. **Upload via SFTP:**
   ```bash
   sftp username@ssh.phx.nearlyfreespeech.net
   put build/buildsite /home/bin/buildsite
   put config.toml /home/config.toml
   put templates/index.tmpl.html /home/templates/index.tmpl.html
   put ops/htaccess /home/public/.htaccess
   ```

3. **Set permissions:**
   ```bash
   ssh username@ssh.phx.nearlyfreespeech.net
   chmod +x /home/bin/buildsite
   mkdir -p /home/data /home/public/assets /home/templates
   ```

4. **Configure cron (Scheduled Tasks in NFSN web UI):**
   - Command: `/home/bin/buildsite -config /home/config.toml`
   - Schedule: Every hour (or `*/10` for 10-minute intervals)

   **Alternative (legacy CLI flags):**
   ```bash
   /home/bin/buildsite -json-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json -xml-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml -csv-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv -out-dir /home/public -data-dir /home/data -lat 40.42338 -lon -3.71217 -radius-km 0.35 -timezone Europe/Madrid
   ```

## Configuration File

The `config.toml` file contains all site configuration:

```toml
[cultural_events]
# datos.madrid.es cultural programming
json_url = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json"
xml_url = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml"
csv_url = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv"

[city_events]
# esmadrid.com tourism/city events
xml_url = "https://www.esmadrid.com/opendata/agenda_v1_es.xml"

[filter]
# Plaza de España coordinates
latitude = 40.42338
longitude = -3.71217
radius_km = 0.35

# Distrito filtering
distritos = ["CENTRO", "MONCLOA-ARAVACA"]

# Time filtering
past_events_weeks = 2  # Exclude events started >2 weeks ago

[output]
html_path = "public/index.html"
json_path = "public/events.json"

[snapshot]
data_dir = "data"

[server]
# For development only
port = 8080
```

**Validate config before deploying:**
```bash
just build
just config
```

## Updates

1. Build new binary: `./scripts/build-freebsd.sh`
2. Upload binary and config:
   ```bash
   sftp username@ssh.phx.nearlyfreespeech.net
   put build/buildsite /home/bin/buildsite
   put config.toml /home/config.toml
   ```
3. Binary will be used on next cron run
