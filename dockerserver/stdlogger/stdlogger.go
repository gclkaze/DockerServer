package stdlogger

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

type STDLogger struct {
	Errors []string
}

func NewSTDLogger() *STDLogger {
	return &STDLogger{}
}

/*
	func (rd *STDLogger) GetType() evalogger.LoggerType {
		return evalogger.STD
	}
*/
func (logger *STDLogger) Printf(id string, statementId string, format string, args ...interface{}) {
	log.Printf(format, args...)
}
func (logger *STDLogger) Errorf(id string, statementId string, format string, args ...interface{}) {
	log.Printf(format, args...)
	logger.Errors = append(logger.Errors, fmt.Sprintf(format, args...))
}
func (logger *STDLogger) Init(id string) {

}
func (logger *STDLogger) PutSuccessMessage(id string, result bool, message string) {
	log.Printf("id:%s,result:%t,message:%s", id, result, message)
}

func (logger STDLogger) IsOnError() bool {
	return false
}
func (logger *STDLogger) Clear() {
}

func (logger STDLogger) PrintMessage(id string, message string) {
	log.Printf("Execution Id : %s, message : %s", id, message)
}

func (logger STDLogger) PrintErrorMessage(id string, message string) {
	log.Printf("Execution Id : %s, message : %s", id, message)
}
