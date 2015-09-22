package skype

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/godbus/dbus"
)

const (
	AttachAttemptMax   = 5
	AttachAttemptRetry = 5
)

type Connection struct {
	conn *dbus.Conn
	obj  *dbus.Object

	Events chan Event
}

type Listener struct {
	conn *Connection
}

type Event string

// Launches a new Skype instance, authenticates, and attaches to the session.
// The 'skype' bin must be found somewhere on the $PATH.
// More than one Skype client is unsupported.
func RunAndAttach(username, password string) (*Connection, error) {
	skypeBin, err := exec.LookPath("skype")
	if err != nil {
		return nil, fmt.Errorf("Could not find skype bin in path: %v", err)
	}

	cmd := exec.Command(skypeBin, "--pipelogin")
	cmd.Stdin = strings.NewReader(fmt.Sprintf(`%s %s`, username, password))

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("Failed to launch skype bin: %v", err)
	}

	go func(c *exec.Cmd) {
		if err := c.Wait(); err != nil {
			if osErr, ok := err.(*exec.ExitError); ok {
				fmt.Printf("skype bin exited: #%v", osErr)
				return
			}
			fmt.Println(err)
		}
	}(cmd)

	for i := 0; i < AttachAttemptMax; i++ {
		time.Sleep(AttachAttemptRetry * time.Second)
		conn, err := Attach()
		if err != nil {
			return nil, err
		}

		if err := conn.SetName("skype4go"); err != nil {
			return nil, err
		}

		if err := conn.SetProtocol(7); err != nil {
			return nil, err
		}
		return conn, nil
	}

	if cmd.Process != nil {
		cmd.Process.Kill()
	}

	return nil, fmt.Errorf("Failed to attach to skype after %d attempts", AttachAttemptMax)
}

// Attaches to the running Skype instance.
// Client must already be logged-in.
// More than one Skype client is unsupported.
func Attach() (*Connection, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}

	c := &Connection{}
	c.Events = make(chan Event, 10)
	c.conn = conn
	c.obj = conn.Object("com.Skype.API", "/com/Skype")

	l := Listener{conn: c}
	if err := conn.Export(l, "/com/Skype/Client", "com.Skype.API.Client"); err != nil {
		return nil, err
	}

	if err := c.SetName("skype4go"); err != nil {
		return nil, err
	}

	if err := c.SetProtocol(7); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Connection) invoke(cmd string) (string, error) {
	var out string
	if err := c.obj.Call("com.Skype.API.Invoke", 0, cmd).Store(&out); err != nil {
		return "", err
	}
	log.Println(out)
	return out, nil
}

func (c *Connection) SetName(name string) error {
	if _, err := c.invoke(fmt.Sprintf("NAME %s", name)); err != nil {
		return err
	}
	return nil
}

func (c *Connection) SetProtocol(id int) error {
	_, err := c.invoke(fmt.Sprintf("PROTOCOL %d", id))
	return err
}

func (l Listener) Notify(event string) (string, *dbus.Error) {
	l.conn.Events <- Event(event)
	return "", nil
}
