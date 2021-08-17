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
	File:   nil,
}

func (l *cli) init() {

	f, err := os.OpenFile("commands.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0755)
	if err != nil {
		return
	}
	l.File = f

	l.SetOutput(f)

	l.SetNoLock()

	l.SetLevel(log.InfoLevel)

	l.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	l.Info("CLI Log Starting...")
}

func (l *cli) Trace(cmd string, execFunc func(cmd string) ([]byte, error)) ([]byte, error) {
	if l.File == nil {
		l.init()
	}

	l.WithField("command", cmd).Info()
	l.File.Sync()

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

	l.
		WithField("command", cmd).
		WithField("response", response).
		WithError(err).
		Info()

	l.File.Sync()

	return rsp, err
}
