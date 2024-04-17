package connection

type Connection interface {
	Connect() error
	Listen()
	Write(data string)
	ReadStringFromConnection() string
	Disconnect() error
}
