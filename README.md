# alertmanager-matrix
Service for sending alerts from the Alertmanager webhook to a Matrix room.

## Usage
The service is configured either through command line arguments or environment variables.
With the provided systemd service file (`alertmanager_matrix.service`),
the configuration is done in `/etc/default/alertmanager_matrix` as follows:

```sh
HOMESERVER=http://localhost:8008
USER_ID=@bot:example.com
TOKEN=<token>
```

Configure Alertmanager with a webhook to this service:

```yaml
receivers:
- name: mail
  email_configs:
  - to: root+alerts@slxh.eu
- name: matrix
  webhook_configs:
  - url: "http://localhost:4051/<room_id>"
```

The service will *not* automatically join configured rooms.

The icons and colors can be configured by providing a JSON file.
The defaults are:

```json
{
	"alert":       "🔔️",
	"information": "ℹ",
	"warning":     "⚠️",
	"critical":    "🚨",
	"ok":          "✅",
	"silenced":    "🔕"
}
```

```json
{
	"alert":       "black",
	"information": "blue",
	"warning":     "orange",
	"critical":    "red",
	"ok":          "green",
	"silenced":    "gray"
}
```
