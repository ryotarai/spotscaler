# Spotscaler

Autoscaler for Amazon EC2 using spot instances

## Usage

First, create config YAML file like https://github.com/ryotarai/spotscaler/blob/master/config.sample.yml

```
$ spotscaler -config config.yml [-dry-run]
```

### HTTP API

```
$ spotscaler -config config.yml
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
