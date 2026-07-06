package main

import (
	"fmt"
	"github.com/godbus/dbus/v5"
	"os"
	"os/user"
)

func main() {
	u, _ := user.Lookup("engene")
	os.Setenv("DBUS_SESSION_BUS_ADDRESS", fmt.Sprintf("unix:path=/run/user/%s/bus", u.Uid))
	conn, err := dbus.SessionBus()
	if err != nil {
		fmt.Println("Error connecting to bus:", err)
		return
	}
	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	call := obj.Call("org.freedesktop.Notifications.Notify", 0, "NetPulse", uint32(0), "", "Test", "Test body", []string{}, map[string]dbus.Variant{}, int32(5000))
	if call.Err != nil {
		fmt.Println("Failed to send notification:", call.Err)
	} else {
		fmt.Println("Success!")
	}
}
