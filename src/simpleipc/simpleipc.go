package simpleipc
/*
	// https://github.com/golang/gofrontend/blob/master/libgo/go/syscall/syscall_unix_test.go
	fd = syscall.Socketpair()
	f = os.NewFile(fd, "")
	conn = net.FileConn(f)
	connUnix = conn.(*net.UnixConn)
	// use home made protocol
	// see https://godoc.org/github.com/ftrvxmtrx/fd
*/

import (
	"encoding/binary"
	"github.com/ftrvxmtrx/fd"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

type Message struct {
	Data  []byte
	Files []*os.File
}

type Query interface {
	GetMessage() *Message
	Reply(m Message) <-chan error
}

type Channel interface {
	Close() error
	Request(m Message, bufferSize int) (<-chan *Message, <-chan error)
	Send(m Message) <-chan error
	Receive() <-chan Query
}

type message struct {
	Message
	Index     int
	IsReply   bool
	outgoing  chan<- *message
	sentError chan<- error
}

func (m *message) GetMessage() *Message {
	return &m.Message
}

func (m *message) Reply(res Message) <-chan error {
	errChan := make(chan error, 1)
	m.outgoing <- &message{
		sentError: errChan,
		Message:   res,
		Index:     m.Index,
		IsReply:   true,
	}
	return errChan
}

type channel struct {
	lock            sync.Mutex
	chanMsg         chan *message
	waitingRequests []chan *Message
	incoming        chan Query
	outgoing        chan *message
	errors          chan error
	quit            chan chan error
}

func NewChannel(cnx *net.UnixConn, recvBuffer, sendBuffer int) Channel {
	c := new(channel)
	c.chanMsg = make(chan *message, 0)
	c.incoming = make(chan Query, recvBuffer)
	c.outgoing = make(chan *message, sendBuffer)
	c.quit = make(chan chan error, 0)
	go c.receive(cnx)
	go c.send(cnx)
	go c.run()
	return c
}

func (c *channel) receive(cnx *net.UnixConn) {
	quitSignal := make(chan error)
	go func() {
		ch := <-c.quit:
		err := <-cnx.SetDeadline(time.Clock())
		if err != nil {
			ch <- err
		} else {
			<-quitSignal
			ch <- nil
		}
	}()
	for {
		var msg message
		var u32 uint32

		err := binary.Read(cnx, binary.BigEndian, &u32)
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			c.errors <- err
			continue
		} else if u32 == 0 {
			msg.IsReply = false
		} else if u32%2 == 1 {
			msg.IsReply = false
			msg.Index = int(int64(u32+1) / 2)
		} else {
			msg.IsReply = true
			msg.Index = int(u32 / 2)
		}

		err = binary.Read(cnx, binary.BigEndian, &u32)
		if err != nil && err == io.EOF {
			c.errors <- io.ErrUnexpectedEOF
			continue
		} else if err != nil {
			c.errors <- err
			continue
		}
		msg.Files, err = fd.Get(cnx, int(u32), nil)
		if err != nil {
			c.errors <- err
			continue
		}

		err = binary.Read(cnx, binary.BigEndian, &u32)
		if err != nil && err == io.EOF {
			c.errors <- io.ErrUnexpectedEOF
			continue
		} else if err != nil {
			c.errors <- err
			continue
		}
		msg.Data = make([]byte, u32)
		_, err = io.ReadFull(cnx, msg.Data)
		if err != nil && err == io.EOF {
			c.errors <- io.ErrUnexpectedEOF
			continue
		} else if err != nil {
			c.errors <- err
			continue
		}

		c.chanMsg <- &msg
	}
	c.chanMsg <- nil
	quitSignal <- nil
	close(quitSignal)
	close(c.chanMsg)
}

func (c *channel) send(cnx *net.UnixConn) {
	var broken bool = false
loop:
	for {
		select {
		case ch := <-c.quit:
			break loop
		case msg := <-c.outgoing:
			if broken {
				msg.sentError <- io.ErrClosedPipe
				close(msg.sentError)
				continue loop
			}

			broken = true
			var idx uint32 = 0
			if msg.IsReply {
				idx = uint32(msg.Index) * 2
			} else if msg.Index != 0 {
				idx = uint32(msg.Index)*2 - 1
			}

			err := binary.Write(cnx, binary.BigEndian, idx)
			if err != nil {
				msg.sentError <- err
				close(msg.sentError)
				continue loop
			}

			var nfiles uint32 = uint32(len(msg.Files))
			err = binary.Write(cnx, binary.BigEndian, nfiles)
			if err != nil {
				msg.sentError <- err
				close(msg.sentError)
				continue loop
			}
			err = fd.Put(cnx, msg.Files...)
			if err != nil {
				msg.sentError <- err
				close(msg.sentError)
				continue loop
			}

			var datalen uint32 = uint32(len(msg.Data))
			err = binary.Write(cnx, binary.BigEndian, datalen)
			if err != nil {
				close(msg.sentError)
				msg.sentError <- err
				continue loop
			}
			written, err := cnx.Write(msg.Data)
			if err != nil {
				msg.sentError <- err
				close(msg.sentError)
				continue loop
			} else if written != len(msg.Data) {
				msg.sentError <- io.ErrShortWrite
				close(msg.sentError)
				continue loop
			}

			close(msg.sentError)
			broken = false
		}
	}
}

func (c *channel) run() {
loop:
	closed := false
	for {
		inChan := c.chanMsg
		if closed {
			inChan = nil
		}
		select {
		case inMsg := <-c.chanMsg:
			if inMsg == nil {
				closed = true
				close(c.incoming)
			}

			var ch chan *Message
			if inMsg.IsReply {
				func() {
					c.lock.Lock()
					defer c.lock.Unlock()
					ch = c.waitingRequests[inMsg.Index-1]
					c.waitingRequests[inMsg.Index-1] = nil
				}()
			}
			if ch != nil {
				ch <- &inMsg.Message
				close(ch)
			} else if !inMsg.IsReply {
				inMsg.outgoing = c.outgoing
				c.incoming <- inMsg
			} else {
				// Drop message, reply to no request
			}
		case err := <-c.errors:
			log.Print(err)
		case ch := <-c.quit:
			close(c.incoming)
			ch <- nil
			break loop
		}
	}
}

func (c *channel) Close() error {
	res := make(chan bool)
	c.quit <- res
	c.quit <- res
	c.quit <- res
	close(c.quit)
	<-res
	<-res
	<-res
	return nil
}

// Find a free slot in m.waitingRequests and insert ch. Return index
func (c *channel) findSlot(ch chan *Message) int {
	c.lock.Lock()
	defer c.lock.Unlock()
	for i, v := range c.waitingRequests {
		if v == nil {
			c.waitingRequests[i] = ch
			return i
		}
	}
	i := len(c.waitingRequests)
	c.waitingRequests[i] = ch
	return i
}

func (c *channel) Request(m Message, bufSize int) (<-chan *Message, <-chan error) {
	res := make(chan *Message, bufSize)
	errChan := make(chan error, 1)
	c.outgoing <- &message{
		Message:   m,
		Index:     c.findSlot(res) + 1,
		IsReply:   false,
		sentError: errChan,
	}
	return res, errChan
}

func (c *channel) Send(m Message) <-chan error {
	errChan := make(chan error, 1)
	c.outgoing <- &message{
		Message:   m,
		Index:     0,
		IsReply:   false,
		sentError: errChan,
	}
	return errChan
}

func (c *channel) Receive() <-chan Query {
	return c.incoming
}
