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
   - Command: `/home/bin/buildsite -json-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json -xml-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml -csv-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv -out-dir /home/public -data-dir /home/data -lat 40.42338 -lon -3.71217 -radius-km 0.35 -timezone Europe/Madrid`
   - Schedule: Every hour (or `*/10` for 10-minute intervals)

## Updates

1. Build new binary: `./scripts/build-freebsd.sh`
2. Upload: `sftp put build/buildsite /home/bin/buildsite`
3. Binary will be used on next cron run
