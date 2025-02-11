# Learning Multimedia Streaming

I am working on setting up a IP cameras to help monitor our home and new dog as we get him used to spending time alone. Happenstantially, I also am working on setting up a home server with ProxMox and want to start developing services I can run on that server. As a first project, I would like to build a service that is capable of streaming from the IP cameras on my LAN and live streaming the feed to a client application (likely an unpublished iOS app) and recording events such as barking, motion detections etc to the server. 

To start, I am learning the basics of multimedia such as:

- Live Streaming Architecture
- Codecs
- Containers
- Protocols

I am using a Reolink Wi-Fi enabled camera here at home. After some exploration, I have learned that it uses UID's to identify devices and a technique called hole punching to allow persistent communication with a remote server to allow the company's client apps to stream away from home. I'd like to disable this functionality and route the stream to a home server via exposed RTSP ports on the cameras and then republish the stream over HLS through a VPN or CloudFlare tunnel pending further research. 

## First Project - Exploring Client Options

As a first step, I'd like to write some simple client code to connect to a camera on my network and record the live stream to an playable file on my laptop. 

To do so, I am going to use [gortsplib](https://github.com/bluenviron/gortsplib) to connect to my device.

In the example code, we connect to a locally running Reolink IP camera, stream the MPEG-4 and H264 feeds and write the result to a TS file on the disk. I've implemented a simple backoff to allow for momentarily lapses in connection if the device is reset or unplugged.

## TODO: learn internals of above code and do write up


