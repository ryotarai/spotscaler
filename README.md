# spot-autoscaler

Scale EC2 spot instances automatically.

## Usage

```
$ spot-autoscaler -config config.yml [-dry-run]
```

## API

```
$ spot-autoscaler -config config.yml -server :8080
```

```
$ curl -XPOST -d '{"StartAt": "2016-10-05T09:00:00Z", "EndAt": "2016-10-05T10:00:00Z", "Capacity": 10}' localhost:8080/schedules
{"Key":"2016-10-05T09:45:59.315042705Z","StartAt":"2016-10-05T09:00:00Z","EndAt":"2016-10-05T10:00:00Z","Capacity":10}

$ curl localhost:8080/schedules
[{"Key":"2016-10-05T09:45:59.315042705Z","StartAt":"2016-10-05T09:00:00Z","EndAt":"2016-10-05T10:00:00Z","Capacity":10}]

$ curl -XDELETE 'localhost:8080/schedules?key=2016-10-05T09:45:59.315042705Z'
{"deleted":true,"key":"2016-10-05T09:45:59.315042705Z"}
```

## Future works

- [ ] Selectable scaling strategy/policy
- [ ] OSS
