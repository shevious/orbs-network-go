# Troubleshooting Guide

** UNDER CONSTRUCTION **

[Back to README](README.md)

## Audience

* Orbs developers

*Eventually the audience will include also Orbs validators and Gamma users*

## Monitoring

We use Grafana Cloud to monitor the main net.
Presently used as an internal tool so it open to Orbs developers only.

### Dashboards

* Concise - [Orbs Production](https://orbsnetwork.grafana.net/d/a-3pW-3mk/orbs-production?orgId=1&refresh=15s&from=now-3h&to=now)
* Detailed - [Orbs DevOps](https://orbsnetwork.grafana.net/d/Eqvddt3iz/orbs-devops?orgId=1&refresh=15s&from=now-3h&to=now)

### Architecture

* Each node has a `/metrics` endpoint which returns metrics in JSON format
* A scraping process runs on an AWS machine and listens to requests to scrape
    * [source](https://github.com/orbs-network/metrics-processor/blob/master/src/prometheus-client.ts)
    * Runs on AWS machine ec2-user@34.212.7.2 using [pm2](http://pm2.keymetrics.io/) - one process per vchain
* A Prometheus server which runs on docker ([source](https://github.com/orbs-network/metrics-processor/blob/master/run-prometheus-docker.sh)) invokes the scraper, which in turn collects metrics from the nodes.
    * The Prometheus server invokes the scraping by calling the scraper process' own `/metrics` endpoint - the scraper process is idle unless invoked
    
* Grafana Cloud

TODO add link to Prism

## Logging

We use logz.io to collect logs from main net.
Login [here](https://app.logz.io/#/dashboard/kibana/discover/4501ce90-4638-11e9-b5c5-c306d6d38229?_g=())

*Presently logz.io is open to Orbs developers only.*

### Toggle Info Logs

By default, only **Error** logs are written to logz.io. To temporarily write **Info** logs to logz.io for debugging, use the following script: 

```
#!/bin/sh

for ip in $NODE_IPS
do
	curl -XPOST http://${ip}/vchains/${VCHAIN}/debug/logs/filter-off
	echo "Exit code from IP ${ip}: $?"
done

```

You will need to define the environment variables VCHAIN and NODE_IPS for this to work

To disable logs, use the same script, but with `filter-on` in the instead of `filter-off` in the `curl` command.

### Navigating logz.io

#### Creating and saving filters

#### Setting time limits
* Relative / Absolute

#### Showing node address on every line
* or any other property 

#### Free text search


