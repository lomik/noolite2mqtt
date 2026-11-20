package mtrf

import (
	"net"
	"testing"
	"time"
)

// После обрыва соединения со стороны модуля Connection должен
// переподключиться сам, не дожидаясь отправки следующей команды
func TestReconnectWithoutSend(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	accepted := make(chan net.Conn, 2)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			accepted <- conn
		}
	}()

	c := Connect(ln.Addr().String())
	defer c.Close()

	waitConn := func() net.Conn {
		select {
		case conn := <-accepted:
			return conn
		case <-time.After(5 * time.Second):
			t.Fatal("no connection from Connection")
			return nil
		}
	}

	// модуль рвет первое соединение
	waitConn().Close()

	// ожидаем переподключения без единой отправленной команды
	second := waitConn()
	second.Close()
}
