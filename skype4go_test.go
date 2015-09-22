package skype

import "testing"

func setup() *Connection {
	conn, err := Attach()
	if err != nil {
		panic(err)
	}
	return conn
}

func TestIntrospect(t *testing.T) {
	if _, err := Attach(); err != nil {
		t.Fatal(err)
	}
}

func TestInitialize(t *testing.T) {
	c := setup()
	if err := c.SetName("skype4go"); err != nil {
		t.Fatal(err)
	}
	if err := c.SetProtocol(7); err != nil {
		t.Fatal(err)
	}
	select {}
}
