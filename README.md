
# Ngoperf

Ngoperf is a Go implemented CLI tool for [clodflare-2020-system-assignment](https://github.com/cloudflare-hiring/cloudflare-2020-systems-engineering-assignment).


## Overview

Ngoperf can perform two tasks:
- get: send one HTTP GET to a URL and print the response body
- profile: send multiple HTTP GET to a URL, and output a summary about status, time, and size

An example output is shown below:
![](https://i.imgur.com/E9WZyfp.png)

## Compilie and Install Ngoperf

Ngoperf is implemented by go, so you have to install go first. (1.15 is recommended)
- [official](https://golang.org/dl/)
- [ubuntu](https://github.com/golang/go/wiki/Ubuntu)

After go is installed,  you can download and extract the zip and then build and install it with the below commands. In the future, it will support "go get".
```bash
cd ngoperf-0.2  
make
```

To uninstall Ngoperf, run:
```bash
make clean
```

## Using Ngoperf

Ngoperf is built on [Cobra](https://github.com/spf13/cobra) to create the CLI interface.

The CLI pattern is `ngoperf COMMAND --FLAG`, and the commands are list below.

### Help command

```bash
ngoperf --help
ngoperf help get
ngoperf help profile
```

### Get command

The get command print the HTTP response body of a given URL.

#### flags

*   -u, --url
    *   request URL
*   -v, --verbose
    *   *print request and response header*
    *   *ngoperf print response body only by default*
*   -z, --http10
    *   *use HTTP/1.0 to request*
    *   *ngoperf use HTTP/1.1 by default*

#### example

```
ngoperf get --url=https://hi.wanghy917.workers.dev/links
```
![](https://i.imgur.com/vOgQo5v.png)

### Profile command

The profile command sends mutilple HTTP GET requests to a url, and output summary about status, time and size. By default, ngoperf will use 5 workers to make 100 requests, but it can be changed by providing the flags.

#### flags

*   -u, --url
    *   *request URL*
    *   *use HTTP/1.0 to request*
*   -z, --http10
*   -p, --np int
    *   *num of request (default 100)*
*   -w, --nw int
    *   *num of worker (default 5)*
*   -s, --sleep int
    *  *the sleep time in second (default 0)*
    *  *ngoperf randomly sleep 0 to s seconds between the requests*

#### example

```
ngoperf profile --url=hi.wanghy917.workers.dev -p=200 -w=10
```
![](https://i.imgur.com/2TzlZZB.png)

## Experiment

### Settings

* I profile 9 popular websites and [mine](https://hi.wanghy917.workers.dev/)
* To prevent the requests are blocked due to high-frequency request, the flags are set to: 
    * Number of requests: **100**
    * Number of workers: **1**
    * Sleep time between requests: **1** to **5** seconds
* I calculate the Time to First Byte ([TTFB](https://en.wikipedia.org/wiki/Time_to_first_byte)) of each request
* Although HTTP/1.1 can reuse the connection, I reconnect for each request for TTFB
* The Ngoperf is run on a Ubuntu AWS EC2 instance
    * The server does not run other tasks requiring CPU and network bandwidth
    * The network connection is stable

### Why TTFB

From the [Wiki page](https://en.wikipedia.org/wiki/Time_to_first_byte),
Time to first byte (TTFB) is a measurement used to indicate a web server's responsiveness or other network resource. This time is made up of:

* Socket connection time
* Time taken to send the HTTP request
* Time taken to get the first byte of the page

Although several groups reported that TTFB is not the most critical measure([ref1](https://blog.cloudflare.com/ttfb-time-to-first-byte-considered-meaningles/), [ref2](https://blog.nexcess.net/time-to-first-byte-ttfb/), [ref3](https://www.littlebizzy.com/blog/ttfb-meaningless)), it is a relatively fair metrics for a profiling tool like Ngoperf or other curl-like tools.

The reason is listed as follows:
* These tools send each request independently
* TTFB is less affected by the page size when the sites use the same encoding or compression method
* Slow TTFB can still indicate some issues about performance, such as slow DNS lookup or server processing time

In reality, I think [user centric metrics](https://web.dev/user-centric-performance-metrics/) has more worth to be monitored.

## Result

The two figures are the main results. Each figure is generated from 1000 requests, and 100 for each URL.

###  TTFB

The below [violin plot](https://en.wikipedia.org/wiki/Violin_plot) shows the result of TTFB. The x-axis is the log of TTFB in ms, and the y-axis shows the request URLs.
![](https://i.imgur.com/JbBT8sa.png)

### Respons size

The below [box plot](https://en.wikipedia.org/wiki/Box_plot) shows the result of the response size. The x-axis is the response size (header+body) in bytes, and the y-axis shows the request URLs.
![](https://i.imgur.com/AJgMcmR.png)


### Interesting Finding

* All profiling have a 100% success rate
* Most TTFB of the websites follow a normal distribution
* Most TTFB of the websites are less than 200 ms (10^2.3)
* Most TTFB shows a long and thin tail, which means some requests are much slower than the mean or median value
* [My website](https://hi.wanghy917.workers.dev/) has 150 ms TTFB on average 
* Wikipedia main page as the fastest TTFB, which is 5 ms on average, and the reason is:
    * It shows cache hit in the response header, `X-Cache-Status: hit-front.`
    * Only IMDb and Wiki response without `chunked transfer encoding (CTE)` and wiki's page is much smaller (80 kb) than IMDb (438 kb)
* Amazon is the second-fastest. One reason is Ngoperf running on EC2. But why it is not the fastest.
    * The reason should be using 'Content-Encoding: gzip`, which means the server needs more time to prepare the response.
   * Using content-encoding could reduce users' download time, and the sites could be loaded earlier.
* Reddit has a very large TTFB (1400 ms on average), so I checked the website manually.
    * I am wondering why it takes so long for this popular website
    * Later, I found it start to render HTML content very early, so the users still have a good user experience
    * As can be seen from the below figure, users can see grey and white boxes, so they will not feel blocked by the website, and all contents with a large size will appear gradually.
![](https://i.imgur.com/ubVula9.png)

## What if we use an aggressive profing setting?

As mentioned in the setting, I use 1 worker to send 100 requests for each website and sleep 0~5 seconds between each request. Here, I would like to share the results using 100 workers to send 1000 requests without sleep. As shown in the figure, the TTFB is much larger, and I think these websites have some mechanisms to prevent DDoS attacks or aggressive crawlers. 

In addition, some websites will start to return non-success status code, and the success rate is no longer 100%. For example, Google returns [302](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/302), IMDb returns [503](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/503), and stackoverflow (not shown in the figure) returns [429](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/429).

![](https://i.imgur.com/eDevCi7.png)

## Implement Detail

* I am more familiar with C++, but I think this is a good chance to learn Go, so I use Go (maybe RUST next time)
* The assigment requires that we should not use a HTTP library
    * For the most part, I read others' code, online documents, and implement my HTTP library
    * For the HTTP/1.1 [chunked transfer encoding](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Transfer-Encoding), I used Go's [source code](https://golang.org/src/net/http/internal/chunked.go), and I only make it simpler.
        * If using HTTP 1.0 only, chunked encoding is not needed 
* Possible extension of the tool in the future could be
    * Support new protocols, e.g. HTTP/3
    * Support user centric metrics e.g. Time to Interactive (TTI)
    * Use mutiple workers, and dynamically change IPs, so the tool can measure performance more efficiently
