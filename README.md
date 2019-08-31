Byte Stream Multicast
========

The service allows one to broadcast arbitrary stream of bytes to several clients over TCP.

Usage example:
--------

Stream a video from MacBook's built-in camera to external network clients over TCP.

1. Run the service on your MacBook:

```bash
go run . -listen-input tcp://127.0.0.1:1024 -listen-output tcp://0.0.0.0:4096
```

2. In another terminal run `ffmpeg` to capture a video from the camera and send it to the supplier's TCP socket of the service:

```bash
ffmpeg -f avfoundation -r 30 -i "default" -v 0 -vcodec mpeg4 -f mpegts - | netcat 127.0.0.1 1024
```

**Note:** Frame rate `-r` parameter may vary depending from the model of your camera.

3. On one or more other computers in your local network or Internet start to read the data from the consumer's TCP socket of the service and pass it to `ffplay`:

```bash
netcat <ip_address_of_the_service> 4096 | ffplay -
```

**Note:** Significant latency between video capture and video playback is caused by the buffer of `ffplay` itself, not by the buffer of the service. To verify that, pass the output of the `ffmpeg` right away to the `ffplay` bypassing the service and watch the result:

```bash
ffmpeg -f avfoundation -r 30 -i "default" -v 0 -vcodec mpeg4 -f mpegts - | ffplay -
```
