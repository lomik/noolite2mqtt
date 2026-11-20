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

// responseBytes собирает пакет ответа модуля с корректной контрольной суммой
func responseBytes(mode, ch uint8) []byte {
	b := []byte{173, mode, 0, 0, ch, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 174}
	for i := 0; i < 15; i++ {
		b[15] += b[i]
	}
	return b
}

// Пакет в другом режиме (нажатие пульта в RX), пришедший сразу после
// запроса, не должен считаться ответом: writer обязан дождаться ответа
// в режиме запроса и только потом отправлять следующую команду
func TestWriterWaitsForMatchingResponse(t *testing.T) {
	const replyDelay = 600 * time.Millisecond

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	arrived := make(chan time.Time, 8)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, MessageLen)
		for {
			if _, err := io.ReadFull(conn, buf); err != nil {
				return
			}
			if buf[1] == ModeService {
				conn.Write(responseBytes(ModeService, 0))
				continue
			}
			arrived <- time.Now()
			// сразу - чужое событие от пульта, ответ на запрос - позже.
			// Отвечаем асинхронно, чтобы не задерживать прием следующего запроса
			mode, ch := buf[1], buf[4]
			conn.Write(responseBytes(ModeRX, 5))
			go func() {
				time.Sleep(replyDelay)
				conn.Write(responseBytes(mode, ch))
			}()
		}
	}()

	c := Connect(ln.Addr().String())
	defer c.Close()

	c.Send() <- &Request{Mode: ModeTX, Ch: 1, Cmd: CmdOn}
	c.Send() <- &Request{Mode: ModeTX, Ch: 2, Cmd: CmdOn}

	var times []time.Time
	for len(times) < 2 {
		select {
		case tm := <-arrived:
			times = append(times, tm)
		case <-time.After(5 * time.Second):
			t.Fatal("requests did not arrive")
		}
	}

	if gap := times[1].Sub(times[0]); gap < replyDelay {
		t.Fatalf("second request sent %s after the first, before the real response (%s)", gap, replyDelay)
	}
}
