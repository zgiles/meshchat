meshchat 
---
A simple peer-to-peer chat software written in Go and React.js


## What is it

This chat system established peer-to-peer relationships between all mesh chat servers, exchanging all messages in a large single chatroom.  

Any chat server can communicate directly with all others; Missed messages are not queued nor acknowledged. As such, this means it is lossy, but it should be self-healing, recover, and provide connectivity for even as low as multi-second outages.

As a major feature, users ( who self-select names ) each show a "status" indicator near their name as their chat-server goes offline, showing who is actually available vs "online". This could be a major positive in a real-time conservation flow where first-responders need to speak to someone now, something more akin to a phone-chat-room rather than a quick email. 

## Future ideas ( unordered )
- Servers self-discover each other
- User names are stored in browser local-store instead of random at load
- Server status indicator and stats mouse-over
- Clients connecting to multiple chat servers
- Support both internal or external NATS
- Partial Mesh for NATS

## Setup
- Git clone
- In frontend, run `npm install` and `npm run build`. ( Standard React.js )
- After, in the root, run `make`. This will copy relevant content from frontend into a go blob and compile.


## Run
- Both Intel and ARM ( Raspi ) binaries are created. 
- Run with peers listed ( to connect to others ):
	```
	Server1:
		./meshchat-amd64 -peers server2:4001
	Server2:
		./meshchat-arm -peers server1:4001
	```
- To run multiple instances on the same node, ensure different ports are chosen:
	```
		./meshchat-amd64 -natroutingport 4001 -natsport 4222 -peers localhost:4004 -httpport 8081 -clusterport 5001
		./meshchat-amd64 -natroutingport 4002 -natsport 4223 -peers localhost:4001 -httpport 8082 -clusterport 5002
		./meshchat-amd64 -natroutingport 4003 -natsport 4224 -peers localhost:4002 -httpport 8083 -clusterport 5003
		./meshchat-amd64 -natroutingport 4004 -natsport 4225 -peers localhost:4003 -httpport 8084 -clusterport 5004
	```

## License 
MIT/Expat  
See LICENSE


