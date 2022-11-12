# Set-Echo Service
This is a very basic service that gets broadcasts a request to all pods linked to a service. This can be useful when you want to communicate a state/config change to multiple running nodes. You could also do this using pub/sub (rabbitmq exchange, redis channels, etc.) but you can also do it by directly by sending requests using the kubernetes golang client as shown in this sample.

The endpoints supported are:

Get the state value
```
GET /get
```

Set the state value (expects body with value)
```
POST /set
```

Sync pod (by looking at value on redis). This is used internally only in this example
```
POST /sync
```
