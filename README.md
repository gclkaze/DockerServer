# DockerServer
The DockerServer is a REST API server built with Golang's gin that:
1. allows the creation & execution of Docker ubuntu containers on-demand,
2. allows the communication between a client and the deployed docker container through socket communication between the server and the client,
3. supports writing files inside the Docker containers after a POST REST request, where the server obtains the submitted file and then copying it in the container,
4. supports retrieving files from the Docker container back to client through the host machine,
5. safely cleaning up the docker containers on-demand and keeping track of orphan sessions. The sessions and the files do not persist.

## Build it through Makefile
```
make build
```

## REST API
1. POST /create
Creates a new Docker container session and returns a listener's network address that can be used from the client in order to listen to the stdout of the newly created Docker container.

#Parameter
ProgramId: string, a unique id to be assigned to the freshly made Docker container.

**Response:**
{
  "socket": "192.0.2.1:25",
}

3. POST /stop
4. POST /writeFile
5. GET /getFile
