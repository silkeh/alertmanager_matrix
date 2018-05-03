# alertmanager-matrix
Service for sending alerts from the Alertmanager webhook to a Matrix room
and managing Alertmanager.

## Usage
The service is configured either through command line arguments or environment variables.
With the provided systemd service file (`alertmanager_matrix.service`),
the configuration is done in `/etc/default/alertmanager_matrix` as follows:

```sh
ARGS=""
HOMESERVER=http://localhost:8008
USER_ID=@bot:example.com
TOKEN=<token>
```

See `alertmanager_matrix -help` for all possible arguments.

Configure Alertmanager with a webhook to this service:

```yaml
receivers:
- name: matrix
  webhook_configs:
  - url: "http://localhost:4051/<room_id>"
```

When the `-rooms` option is provided the bot will join the listed rooms and
only allow commands from these rooms.
The service will *not* automatically join the room given in a webhook.

The icons and colors can be configured by providing a JSON file.
The defaults are:

```json
{
	"alert":       "🔔️",
	"information": "ℹ️",
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
