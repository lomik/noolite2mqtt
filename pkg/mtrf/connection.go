package mtrf

import (
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/jacobsa/go-serial/serial"
)

// Connection ...
type Connection struct {
	exit      chan struct{}
	exitOnce  sync.Once
	addr      string
	sendQueue chan *Request
	recvQueue chan *Response
	recvEvent *Event
}

// Connect подключается к модулю (локальному девайсу или tcp)
func Connect(addr string) *Connection {
	c := &Connection{
		addr:      addr,
		exit:      make(chan struct{}),
		sendQueue: make(chan *Request, 64),
		recvQueue: make(chan *Response, 64),
		recvEvent: NewEvent(),
	}
	go c.worker()
	return c
}

// Close закрывает соединение
func (c *Connection) Close() {
	c.exitOnce.Do(func() {
		close(c.exit)
	})
}

func (c *Connection) connectTCP() io.ReadWriteCloser {
	for {
		select {
		case <-c.exit:
			return nil
		default:
		}

		conn, err := net.DialTimeout("tcp", c.addr, time.Second)
		if err != nil {
			log.Println(err)
			select {
			case <-c.exit:
				return nil
			default:
			}
			time.Sleep(time.Second)
			continue
		}

		return conn
	}
}

func (c *Connection) connectSerial() io.ReadWriteCloser {
	options := serial.OpenOptions{
		PortName:        c.addr,
		BaudRate:        9600,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 17,
	}

	for {
		select {
		case <-c.exit:
			return nil
		default:
		}

		conn, err := serial.Open(options)
		if err != nil {
			log.Println(err)
			select {
			case <-c.exit:
				return nil
			default:
			}
			time.Sleep(time.Second)
			continue
		}

		return conn
	}
}

func (c *Connection) connect() io.ReadWriteCloser {
	if strings.HasPrefix(c.addr, "/") {
		return c.connectSerial()
	}

	return c.connectTCP()
}

func (c *Connection) worker() {
	for {
		conn := c.connect()
		if conn == nil { // exit
			return
		}
		log.Println("connected")
		var wg sync.WaitGroup
		abortChan := make(chan struct{}, 0)
		var abortOnce sync.Once

		abort := func() {
			abortOnce.Do(func() {
				conn.Close()
				close(abortChan)
			})
		}

		wg.Add(3)

		go func() {
			defer wg.Done()
			log.Println("writer started")
			c.writer(conn)
			abort()
		}()

		go func() {
			defer wg.Done()
			c.reader(conn)
			abort()
		}()

		go func() {
			defer wg.Done()
			select {
			case <-abortChan:
				return
			case <-c.exit:
				abort()
				return
			}
		}()

		wg.Wait()
	}
}

func (c *Connection) writer(conn io.ReadWriteCloser) {
	for {
		select {
		case <-c.exit:
			return
		case m := <-c.sendQueue:
			body := m.Bytes()

			// log.Printf("send msg: %s\n", m.String())
			c.recvEvent.Clear()

			if _, err := conn.Write(body); err != nil {
				log.Println(err)
				return
			}

			if c.recvEvent.Wait(time.Second) {
				// add extra timeout between messages
				time.Sleep(200 * time.Millisecond)
			}
		}
	}
}

func (c *Connection) reader(conn io.ReadWriteCloser) {
	var buf [17]byte
	for {
		_, err := io.ReadAtLeast(conn, buf[:], 17)
		if err != nil {
			log.Println(err)
			return
		}

		c.recvEvent.Raise()

		rs, err := NewResponse(buf[:])
		if err != nil {
			log.Printf("recv error: %s, raw: %#v\n", err.Error(), buf)
			return
		}

		// log.Printf("recv msg: %s\n", rs.String())

		select {
		case c.recvQueue <- rs:
			// pass
		default:
			log.Println("recv queue is full, skipped")
		}
	}
}

// Recv возвращает канал для получения данных от модуля
func (c *Connection) Recv() <-chan *Response {
	return c.recvQueue
}

// Send возвращает каналь для оправки команд в модуль
func (c *Connection) Send() chan<- *Request {
	return c.sendQueue
}
