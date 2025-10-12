# DockerServer - Development
The __DockerServer__ is a __REST API server__ built with __Golang's gin__ that:
1. allows the creation & execution of Docker ubuntu containers on-demand,
2. allows the communication between a client and the deployed docker container through socket communication between the server and the client,
3. supports writing files inside the Docker containers after a POST REST request, where the server obtains the submitted file and then copying it in the container,
4. supports retrieving files from the Docker container back to client through the host machine,
5. safely cleaning up the docker containers on-demand and keeping track of orphan sessions. The sessions and the files do not persist.

# Purpose 
Needed to be able to spawn Ubuntu containers on demand in order to execute various terminal based commands 
without needed to care about "malicious commands" like "rm -rf *" that would be issued by an evil user.

By building a Docker server that keeps track of the docker containers running, and providing to the user an environment that could do as he/she pleases without the fear of impacting the host machine where the actual server is running.

# Technique
The communication between the executed docker container and the client is by opening a __TCP Unix socket__ and allowing
the client to send commands to the Docker Server which are passed to the running Docker container.
The stdout then is redirected back to the client. Then the client can issue another unix command/program execution
as long as the socket is open and the docker container is running.

# Configuration
Under __dockerserver/resources__, the file __application.properties__ exists.
```
workingDirectory= #The working directory
jobsPrefix= #Spawned container name prefix
host= #Hostname of the DockerServer instance
port= #Port of the DockerServer instance
wsFolder= #Default workspace folder that is the current working directory of all spawned Docker instances
```

# Requirements
___docker___,
___golang___

## Build it through Makefile
```
make build
```

## REST API
### 1. POST /create
Creates a new Docker container session and returns a listener's network address that can be used from the client in order to listen to the stdout of the newly created Docker container.

#### Parameter (passed in the request's Header for now)
___ProgramId:___ string, a unique id to be assigned to the freshly made Docker container.

#### **Response:**
```
{
  "socket": "192.0.2.1:25",
}
```

### 2. POST /stop
Stops the session (if it exists) that is assigned to the programId parameter value.

#### Parameter (passed in the request's Header for now)
___ProgramId___: string, a unique id to be assigned to the running made Docker container.

#### **Response:**
200 (OK) or 500 (Internal Server Error)

### 3. POST /writeFile
Writes the submitted data to a file with a specific file name (Filename) inside a running container associated with the value of the ProgramId parameter.

#### Parameters (passed in the request's Header for now)
1. ___ProgramId:___ string, a unique id associated with the running made Docker container.
2. ___Filename:___ the name of the new file inside the Docker container
3. ___Post body:___ the data to be written in the file

#### **Response:**
200 (OK) or 500 (Internal Server Error)

### 4. GET /getFile
Gets the data of a file that exists inside a running Docker container associated with a programId.

#### Parameters (passed in the request's Header for now)
1. ___ProgramId:___ string, a unique id associated with the running made Docker container.
2. ___Filename:___ the name of the new file inside the Docker container

#### **Response:**
The read file data-content.
