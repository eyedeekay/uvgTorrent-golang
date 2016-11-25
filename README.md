# uvgTorrent-golang

uvgTorrent-golang is an educational torrent streaming client built with golang. It isn't intended for regular use, but demonstrates using a magnet link to connect to peers via the udp tracker protocol, downloading torrent metadata from those peers and then streaming a selected video file by downloading the file in sequential order and playing it through vnc. 

Besides demonstrating the basics of the torrent protocol uvgTorrent also shows off just how suited golang is for high concurrency, network heavy programs. Golang's built in concurrency support makes it easy to handle communication with each peer in a seperate goroutine, using channels to orchastrate loading metadata, dividing pieces between the peers for download, downloading the pieces, and saving them to the harddrive.

# requirements

- golang 1.6.3
- vlc (if you want to open vlc to stream the video file you select for download)

# usage

``bash
#!/bin/bash

export GOPATH=/absolute/path/to/uvgTorrent-golang
go run "magnet:magneturlgoeshere"
```

# torrent protocol

# technical description