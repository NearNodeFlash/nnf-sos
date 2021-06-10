package logging

import (
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type cli struct {
	*log.Logger
	*os.File
}

var Cli = cli{
	Logger: log.New(),
}

func init() {

	f, err := os.OpenFile("commands.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0755)
	if err != nil {
		return
	}
	Cli.File = f

	Cli.SetOutput(f)

	Cli.SetNoLock()

	Cli.SetLevel(log.InfoLevel)

	Cli.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	Cli.Info("CLI Log Starting...")
}

func (log *cli) Trace(cmd string, execFunc func(cmd string) ([]byte, error)) ([]byte, error) {
	log.WithField("command", cmd).Info()
	log.File.Sync()

	rsp, err := execFunc(cmd)

	isPrintable := func(rsp []byte) bool {
		for _, r := range string(rsp) {
			if !strconv.IsPrint(r) {
				switch r {
				case '\n', '\r', '\t':
					continue
				}

				return false
			}
		}

		return true
	}

	response := "(bytes)"
	if isPrintable(rsp) {
		response = strconv.QuoteToASCII(string(rsp))
	}

	log.
		WithField("command", cmd).
		WithField("response", response).
		WithError(err).
		Info()

	log.File.Sync()

	return rsp, err
}
