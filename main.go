package main


import (
	"fmt"
	"net/http"

	"github.com/gclkaze/DockerServer/dockerserver"
	"github.com/gin-gonic/gin"
)

func createNewSession(config string, programId string, _ http.Header, c *gin.Context) {
	sock, err := dockerserver.TheServer.CreateSession(config, programId)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}

	//c.IndentedJSON(http.StatusOK, )
	d := []byte(`{"socket":"` + sock + `"}`)
	c.Data(http.StatusOK, "application/json", d)

	dockerserver.TheServer.OpenSession(sock, config, programId)
}

func stopCurrentSession(config string, programId string, header http.Header, c *gin.Context) {
	_, err := dockerserver.TheServer.StopSession(config, programId)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}

	c.IndentedJSON(http.StatusOK, config)
}

func doWriteFile(data string, programId string, header http.Header, c *gin.Context, filename string) {

	res, err := dockerserver.TheServer.WriteFile(data, programId, filename)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}

	c.IndentedJSON(http.StatusOK, res)
}

func createSession(c *gin.Context) {
	jsonData, err := c.GetRawData()

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	h := c.Request.Header
	prId := h["Programid"][0]

	config := string(jsonData[:])

	defer logExecutionError()
	createNewSession(config, prId, h, c)
}

func writeFile(c *gin.Context) {
	jsonData, err := c.GetRawData()

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	h := c.Request.Header
	prId := h["Programid"][0]
	filename := h["Filename"][0]
	config := string(jsonData[:])

	defer logExecutionError()
	doWriteFile(config, prId, h, c, filename)
}

func stopSession(c *gin.Context) {
	jsonData, err := c.GetRawData()

	if err != nil {
		//Handle Error
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	h := c.Request.Header
	prId := h["Programid"][0]
	config := string(jsonData[:])

	go func() {
		defer logExecutionError()
		stopCurrentSession(config, prId, h, c)
	}()

	c.IndentedJSON(http.StatusOK, prId)
}

func logExecutionError() {
	if r := recover(); r != nil {
		fmt.Println("Caught:", r)
	}
}

func main() {
	dockerserver.InitDockerServer()

	router := gin.Default()
	router.POST("/create", createSession)
	router.POST("/stop", stopSession)
	router.POST("/writeFile", writeFile)

	router.Run(dockerserver.TheServer.GetAddress())

}
