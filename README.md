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

## Message customization

The alert messages can be customized by providing custom templates using the `-text-template` and `-html-template` flags.
The built-in default templates can be found in [the documentation][constants].

The icons and colors define the behaviour of the built-in `icon` and `color` templating functions.
They can be configured by providing a YAML file using `-icon-file` and `-color-file` respectively.
See [the documentation][variables] for the default values.

[constants]: https://pkg.go.dev/github.com/silkeh/alertmanager_matrix/bot#pkg-constants
[variables]: https://pkg.go.dev/github.com/silkeh/alertmanager_matrix/bot#pkg-variables
