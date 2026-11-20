package mtrf

import (
	"bytes"
	"io"
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

// Сразу после подключения модулю должен уйти запрос сервисного режима,
// который выводит модуль из режима обновления ПО после подачи питания
func TestServiceRequestOnConnect(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	c := Connect(ln.Addr().String())
	defer c.Close()

	conn, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, MessageLen)
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ServiceRequest().Bytes(), buf) {
		t.Fatalf("first packet %v, expected service request %v", buf, ServiceRequest().Bytes())
	}
}
