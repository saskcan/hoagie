# Hoagie

Hoagie is the first piece of a distributed stock picking system. It is responsible for accepting job requests off of a RabbitMQ queue and scraping Yahoo Finance for historical price data. Results are returned to another RabbitMQ queue.

## Requirements

Hoagie is not a complete solution on its own. It requires a job producer and a data consumer to be useful. It also requires RabbitMQ to be up and running. For now, configuration is mostly hard-coded.

### Job Producer

Although not ready to run on its own, here's a javascript snippet that will get you on the right track:

```
var amqp = require('amqplib/callback_api');

amqp.connect('amqp://localhost', (err, conn) => {
  console.log(err);
  console.log(conn);
  conn.createChannel((err, ch) => {
    var q = 'jobs';

    ch.assertQueue(q, {durable: false});
    ch.sendToQueue(q, new Buffer('{"symbol":"MSFT","date_range":{"start":"2011-01-01T01:00:00+00:00","end":"2017-09-01T01:00:00+00:00"},"frequency":4}'));
    console.log(" [x] Sent job");

    setTimeout(() => {
      conn.close();
      process.exit(0);
    }, 500);
  });
});
```

### RabbitMQ

Probably the easiest thing to do is use RabbitMQ with Docker. The following set of bash commands is probably 90% of what you need to do. Of course you'll need to pull the container to begin with.

```
# stop running container
docker stop some-rabbit

# remove existing docker container if it exists
docker rm some-rabbit

# create new docker container
docker run -p 15672:15672 -p 5672:5672 -d --hostname my-rabbit --name some-rabbit rabbitmq

# enable admin tools
docker exec some-rabbit rabbitmq-plugins enable rabbitmq_management
```

### Consumer

You're on your own. Explore RabbitMQ's web management tools to see the format of the data you can consume. If you're running RabbitMQ locally: http://localhost:15672/#/queues

## Getting Started

Hoagie is written in Go. You will need your machine to be set up to run Go. There are many good articles on the web. Google away!

Hoagie uses Glide to manage dependencies. Install dependencies using:

```
glide install
```

If you don't have Glide on your machine already, use
```
go get glide
```
To run Hoagie, from the project directory, execute
```
go run main.go
```

## Future direction

If you have not already noticed, this project is in the early stages and will need to be improved before it can be considered ready for production.
