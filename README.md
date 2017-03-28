# Spotscaler

[![Docker Repository on Quay](https://quay.io/repository/ryotarai/spotscaler/status "Docker Repository on Quay")](https://quay.io/repository/ryotarai/spotscaler)

Autoscaler for Amazon EC2 using spot instances

**This is working in production environment in Cookpad, but still heavily under development and refactoring**
**Documentation will be prepared in a few month**

## Getting Started

### 1. Configurate

1. Copy a sample config file: https://github.com/ryotarai/spotscaler/blob/master/config.sample.yml
2. Edit the file along the instructions in the sample config
3. Run validation of config: `spotscaler validate -config config.yml`

### 2. Run simulation

```
$ spotscaler simulate -config config.yml
```

This shows current status and a status to be after Spotscaler runs.

### 3. Start scaling

```
$ spotscaler start -config config.yml
```

This gets current status periodically and scale the instances out/in if needed.
Spotscaler stays in foreground and you can run this with a supervisor like systemd.

### HTTP API

```
$ spotscaler -config config.yml -server :8080
```

```
$ curl -XPOST -d '{"StartAt": "2016-10-05T09:00:00Z", "EndAt": "2016-10-05T10:00:00Z", "Capacity": 10}' localhost:8080/schedules
{"Key":"2016-10-05T09:45:59.315042705Z","StartAt":"2016-10-05T09:00:00Z","EndAt":"2016-10-05T10:00:00Z","Capacity":10}

$ curl localhost:8080/schedules
[{"Key":"2016-10-05T09:45:59.315042705Z","StartAt":"2016-10-05T09:00:00Z","EndAt":"2016-10-05T10:00:00Z","Capacity":10}]

$ curl -XDELETE 'localhost:8080/schedules?key=2016-10-05T09:45:59.315042705Z'
{"deleted":true,"key":"2016-10-05T09:45:59.315042705Z"}

$ curl localhost:8080/status
```

## Why not spot fleet?

TODO
