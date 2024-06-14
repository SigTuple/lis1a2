package connection

type Connection interface {
	Connect() error
	IsConnected() bool
	Listen()
	Write(data string)
	ReadStringFromConnection() (string, error)
	Disconnect() error
}
