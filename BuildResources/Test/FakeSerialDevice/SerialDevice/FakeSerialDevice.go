/*
* @Author: Ximidar
* @Date:   2018-12-12 14:33:07
* @Last Modified by:   Ximidar
* @Last Modified time: 2018-12-12 16:33:21
 */

package FakeSerialDevice

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/pkg/term/termios"
)

const (
	// SerialName is the name we are going to assign to our fake serial device
	SerialName = "/tmp/fakeprinter"
)

// FakeSerial is an object that will emulate a serial device
type FakeSerial struct {
	ptyMaster   *os.File
	ptySlave    *os.File
	ptySettings *syscall.Termios

	// Streams
	ReceiveStream chan byte
	SendStream    chan byte
}

// NewFakeSerial will construct a new fake serial device
func NewFakeSerial() *FakeSerial {
	fs := new(FakeSerial)
	var err error
	fs.ptyMaster, fs.ptySlave, err = termios.Pty()
	if err != nil {
		panic(err)
	}
	fmt.Println("Master: ", fs.ptyMaster.Name())
	fmt.Println("Slave: ", fs.ptySlave.Name())

	// Change the file mods
	if _, err := os.Stat(SerialName); !os.IsNotExist(err) {
		os.RemoveAll(SerialName)
	}
	err = os.Chmod(fs.ptySlave.Name(), 0660)
	if err != nil {
		panic(err)
	}
	err = os.Symlink(fs.ptySlave.Name(), SerialName)
	if err != nil {
		panic(err)
	}

	// Set up fake device
	//setNonBlock(fs.ptyMaster)
	fs.ptySettings = new(syscall.Termios)
	termios.Tcgetattr(fs.ptyMaster.Fd(), fs.ptySettings)
	termios.Tcsetattr(fs.ptyMaster.Fd(), termios.TCSADRAIN, fs.ptySettings)
	fmt.Println(fs.ptySettings.Ispeed)
	fmt.Println(fs.ptySettings.Ospeed)

	// make streams
	fs.ReceiveStream = make(chan byte)
	fs.SendStream = make(chan byte)

	return fs
}

func setNonBlock(fd *os.File) {
	err := syscall.SetNonblock(int(fd.Fd()), true)
	if err != nil {
		panic(err)
	}
}

// ReadMaster will read all available bytes coming in over the serial device
func (fs *FakeSerial) ReadMaster() {
	for {
		buf := make([]byte, 1)
		_, err := io.ReadAtLeast(fs.ptyMaster, buf, 1)
		if err != nil {
			continue
		}

		fs.ReceiveStream <- buf[0]
	}
}

func (fs *FakeSerial) SendBytes(buf []byte) {
	_, err := fs.ptyMaster.Write(buf)
	if err != nil {
		fmt.Println(err.Error())
	}
}

// SendMaster will send any bytes that come over the stream
func (fs *FakeSerial) SendMaster() {
	for {
		select {
		case buf := <-fs.SendStream:
			buffy := make([]byte, 1)
			buffy[0] = buf
			_, err := fs.ptyMaster.Write(buffy)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	}
}
