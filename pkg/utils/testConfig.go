package utils

import (
	"os"
	log "github.com/sirupsen/logrus"
)

var testScript string = `-test.run TestAdamOnBoard
-test.run TestControllerSetConfig
-test.run TestControllerGetConfig
-test.run TestControllerLogs
-test.run TestControllerInfo
-test.run TestBaseImage
-test.run TestNetworkInstance
-test.run TestApplication
`

//GenerateTestSript is a function to generate default script for testing
func GenerateTestSript(filePath string) error {
        file, err := os.Create(filePath)
        if err != nil {
                log.Fatal(err, filePath)
        }
        defer file.Close()
	_, err = file.WriteString(testScript)
        if err != nil {
                log.Fatal(err, filePath)
        }
	log.Info("Default test script generated: ", filePath)
        return err
}
