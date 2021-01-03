package miniquet

type Logger interface {
	WriteMsgLog(string, ...interface{})
	WriteErrLog(string, ...interface{})
}
